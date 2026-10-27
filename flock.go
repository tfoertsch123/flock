/*
   Copyright 2026 Torsten Foertsch

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
 */

// Package flock implements a simple interface to [flock(2)] and [fcntl(2)]
// open file description locks on Linux.
//
// # Lock Semantics
//
// These locks are linked to file descriptors not processes. That means a
// lock can be inherited by child processes. It can be sent to unrelated
// processes via [sendmsg(2)] or similar. The file descriptor duplicated via
// [dup(2)] similarly inherits the lock. In all these cases, the lock is then
// shared by all file descriptors. If a lock is held this way by a number
// of file  descriptors, it is released either when ANY one calls
// [*Lock.Unlock] or [*Lock.UnlockRange] (or the corresponding syscalls)
// or ALL of the file descriptors are closed.
//
// These locks are advisory locks. They do not prevent any operation on
// the related files. The locks implemented by [*Lock.LockEx], [*Lock.LockSh]
// and  [*Lock.Unlock] represent exactly one lock per file. They are based
// on [flock(2)] and are somewhat portable across POSIX OS.
//
// Range locks as provided by [*Lock.LockRangeEx], [*Lock.LockRangeSh] and
// [*Lock.UnlockRange] are based on [fcntl(2)] and specific to Linux. They
// allow multiple independent locks on the same file descriptor.
//
// [fcntl(2)] implements similar process based locks, see F_SETLK, F_SETLKW,
// and F_GETLK on the man page. These locks are NOT implemented by this
// package because their semantic is different.
//
// # Range Locks
//
// Range locks are good if you want to tie multiple locks to the same
// file. Every file position can be an independent lock. The lock range
// is not related to the file size. The file can have a size of 0 while
// you lock the range from 100 to 200. If you just need a single lock
// file, use [flock(2)] based lock.
//
// # Blocking
//
// This package has Lock... functions and TryLock... versions. They behave
// the same except that the Lock version waits until the lock becomes
// available while TryLock returns an error in this case.
//
// In most cases, you likely don't want to use the blocking version in a
// Go program.
//
// There is no [context.Context] based version or any other way to wait for a
// lock with a timeout. The only reliable way to implement that is by polling.
//
// # Ownership
//
// A [Lock] object might or might not OWN its file descriptor. If the object
// owns it, then the file descriptor is closed when [*Lock.Close] is called.
// Otherwise, the owner of the descriptor is responsible for closing it.
// See [*Lock.File] and [WithFd] for more information.
//
// As a general rule, the Lock object owns the file descriptor if it has
// opened it.
//
// # Thread safety
//
// The internal data structure is NOT protected by any means.
//
// [fcntl(2)]: https://man7.org/linux/man-pages/man2/fcntl.2.html
// [flock(2)]: https://man7.org/linux/man-pages/man2/flock.2.html
// [dup(2)]: https://man7.org/linux/man-pages/man2/dup.2.html
// [sendmsg(2)]: https://man7.org/linux/man-pages/man2/sendmsg.2.html
package flock

import (
	"os"
	"errors"

	"golang.org/x/sys/unix"
)

// that way the user does not have to import "unix" or similar

// SEEK_SET, SEEK_CUR and SEEK_END represent the same values as used in
// Linux' [lseek(2)] syscall. They are used as parameters for range locks
// like [*Lock.TryLockRangeEx].
//
// [lseek(2)]: https://man7.org/linux/man-pages/man2/lseek.2.html
const SEEK_SET int16 = unix.SEEK_SET
const SEEK_CUR int16 = unix.SEEK_CUR
const SEEK_END int16 = unix.SEEK_END

// State describes the state of the lock. It regulates closing the file
// descriptor and prevents accidental reopening.
//
//                   +-----------+
//               +---+ Init      |             // not yet opened
//               |   +-----------+
//               |         |
//      WithFd() |         | Open()
//               |         V
//               |   +-----------+
//               |   | Owned     +---+
//               |   +-----------+   |
//               |         |         |
//               |         | File()  |
//               |         V         |
//               |   +-----------+   |
//               +-->| Borrowed  |   |
//                   +-----------+   |
//                         |         |
//                 Close() |         | Close()
//                         V         |
//                   +-----------+   |
//                   | Closed    |<--+
//                   +-----------+
type State uint8
const (
	Invalid State = iota
	Init						// not yet opened
	Owned						// we are responsible for closing it
	Borrowed					// we are NOT responsible for closing it
	Closed						// once closed, it cannot be reopened
)
var state2string []string = []string{
	"Invalid",
	"Init",
	"Owned",
	"Borrowed",
	"Closed",
}

// String() returns the string representation of the state
func (t State) String() string {
	if int(t) >= len(state2string) {
		return state2string[0]
	}
	return state2string[int(t)]
}

// A Lock represents the lock object.
type Lock struct {
	path        string
	pathAt      int				// dirfd for openat
	fd          int
	retryOnINTR bool
	openMode    int
	perms       uint32
	state       State
}

type opts struct {
	path        string
	pathAt      int				// dirfd for openat
	fd          int
	closeOnExec bool
	isDir       bool
	retryOnINTR bool
	create      bool
	truncate    bool
	append      bool
	openMode    int
	perms       uint32
	state       State
}

// An Option represents an option passed to the constructor [New].
// All options are functions starting with With...
type Option func(*opts)

// WithPath passes the name of the file, see [New]. This is optional.
// However, if no path is passed, a file descriptor number needs to
// be provided, see [WithFd].
func WithPath(param string) Option {return func(o *opts){o.path = param}}

// WithPathAt optionally passes a directory file descriptor relative to
// which the path passed with [WithPath] is interpreted.
func WithPathAt(param int) Option {return func(o *opts){o.pathAt = param}}

// WithFd allows you to pass an open file descriptor to the constructor [New].
// This can be combined with [WithPath]. However, the path in this case will
// never be used for anything. If a file descriptor is passed this way, it
// is not owned by the [Lock] object. So, [*Lock.Close] will not close the
// file. All the modifiers for opening the file, will also be ignored,
// [WithRdWr], [WithTruncate], ...
//
// Passing a negative parameter, which is an invalid file descriptor, is
// ignored. [*Lock.Fd] will return -1. The state of this lock will be Closed.
func WithFd(param int) Option {return func(o *opts){
	if param >= 0 {
		o.state = Borrowed
		o.fd = param
	} else {
		o.state = Closed
	}
}}

// WithCloseOnExec corresponds to O_CLOEXEC in [open(2)].
//
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func WithCloseOnExec() Option {return func(o *opts){o.closeOnExec = true}}

// WithIsDir corresponds to O_DIRECTORY in [open(2)].
//
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func WithIsDir() Option {return func(o *opts){o.isDir = true}}

// WithRdWr corresponds to O_RDWR in [open(2)].
//
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func WithRdWr() Option {return func(o *opts){o.openMode = unix.O_RDWR}}

// WithWrOnly corresponds to O_WRONLY in [open(2)].
//
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func WithWrOnly() Option {return func(o *opts){o.openMode = unix.O_WRONLY}}

// WithTruncate corresponds to O_TRUNC in [open(2)].
//
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func WithTruncate() Option {return func(o *opts){o.truncate = true}}

// WithAppend corresponds to O_APPEND in [open(2)].
//
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func WithAppend() Option {return func(o *opts){o.append = true}}

// WithCreate corresponds to O_CREAT in [open(2)]. The parameters are ORed
// together to build the file mode. If completely omitted, the default
// file permissions are 0o600 or unix.S_IRUSR + unix.S_IWUSR or "rw-------".
//
// The example below sets the file mode to 0o603 or "rw-----wx" (which
// does not make much sense, though)
//
// Example:
//
//   lck := flock.New(
//       flock.WithCreate(unix.S_IRUSR, unix.S_IWUSR, 3),
//       ...
//   )
//
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func WithCreate(perms ...uint32) Option {return func(o *opts){
	if len(perms) > 0 {
		o.perms = 0
		for _, p := range perms {o.perms |= p}
	} else {
		o.perms = 0o600
	}
	o.create = true
}}

// WithoutIntrRetry turns off the internal retry mechanism if the syscall
// comes back with EINTR. Mainly, this applies to the blocking syscalls.
func WithoutIntrRetry() Option {return func(o *opts){o.retryOnINTR = false}}

// New creates the [Lock] object. At this time the file is not opened, yet.
// The file is only opened (unless an open file descriptor is passed), when
// either [*Lock.Open] is called or any of the Lock functions is called for
// the first time.
//
// Example:
//
//   lck := flock.New(
//       flock.WithPath("/my/lock/file"),
//       flock.WithCreate(unix.S_IRUSR, unix.S_IWUSR),
//       flock.WithRdWr(),
//       flock.WithCloseOnExec(),
//   )
//
// Note, if you use [WithFd] to pass an open file descriptor, that file
// descriptor will not be closed by [*Lock.Close].
//
// It is cheap to create these objects. No syscall involved.
func New(opt ...Option) *Lock {
	opts := &opts{
		pathAt: unix.AT_FDCWD,
		fd: -1,
		state: Init,
		retryOnINTR: true,
		openMode: unix.O_RDONLY,
	}
	for _, o := range(opt) {o(opts)}

	flags := opts.openMode
	if opts.create {flags |= unix.O_CREAT}
	if opts.append {flags |= unix.O_APPEND}
	if opts.truncate {flags |= unix.O_TRUNC}
	if opts.isDir {flags |= unix.O_DIRECTORY}
	if opts.closeOnExec {flags |= unix.O_CLOEXEC}

	return &Lock{
		path:        opts.path,
		pathAt:      opts.pathAt,
		fd:          opts.fd,
		retryOnINTR: opts.retryOnINTR,
		openMode:    flags,
		perms:       opts.perms,
		state:       opts.state,
	}
}

// ErrAlreadyClosed is returned by [*Lock.Close] if the underlying file
// descriptor is deemed to be already closed.
// [*Lock.Open] also returns this error on the attempt to reopen an already
// closed lock.
var ErrAlreadyClosed = errors.New("file is already closed")

// Close closes the underlying file descriptor if it is owned by the [Lock]
// object. Closing an already closed lock returns [ErrAlreadyClosed]
//
// Note, Close does NOT unlock anything prior to closing the file. So, if the
// file descriptor is still open as in has been inherited by a child process
// or [dup(2)]ed or similar, the corresponding locks are still being held.
//
// [dup(2)]: https://man7.org/linux/man-pages/man2/dup.2.html
func (l *Lock) Close() error {
	if l.state == Closed {
		return ErrAlreadyClosed
	}

	var err error
	if l.state == Owned {
		err = unix.Close(l.fd)
	}
	l.fd = -1
	l.state = Closed

	return err
}

// ErrAlreadyOpen is returned by [*Lock.Open] if the underlying file
// descriptor is deemed to be open. There is no syscall performed to actually
// check if the file descriptor is already open. Just if the stored file
// descriptor number is negative, the error is triggered.
var ErrAlreadyOpen error = errors.New("file is already open")

// Open opens the underlying file. If the file is deemed to be already open,
// [ErrAlreadyOpen] is returned. Otherwise, the underlying operating system
// error is returned. On success, nil is returned.
//
// This function is called automatically when a locking function is called
// and the file is not yet open.
func (l *Lock) Open() error {
	if l.state == Closed {return ErrAlreadyClosed}
	if l.state != Init {return ErrAlreadyOpen}

	f, err := unix.Openat(l.pathAt, l.path, l.openMode, l.perms)
	if err != nil {return err}
	l.fd = f
	l.state = Owned
	return nil
}

// Path returns the file path stored in the [Lock] object.
func (l *Lock) Path() string {return l.path}

// Fd returns the underlying file descriptor number. If the file is closed,
// -1 is returned.
func (l *Lock) Fd() int {return l.fd}

// OwningFd can be used to determine if [*Lock.Close] will close the
// underlying file descriptor.
func (l *Lock) OwningFd() bool {return l.state == Owned || l.state == Init}

// IsClosed returns true if the lock is closed
func (l *Lock) IsClosed() bool {return l.state == Closed}

// IsOpen returns true if the underlying file descriptor is open
// state == Owned || state == Borrowed
func (l *Lock) IsOpen() bool {
	return l.state == Owned || l.state == Borrowed
}

// File returns an [*os.File] object representing the underlying file.
// Calling this function disowns the file descriptor. After that, the
// [*os.File] object is responsible for closing it.
// If the lock is not in Owned state, nil is returned. This prevents
// the creation of multiple [*os.File] objects for the same open file.
// It also prevents creating an [*os.File] for an invalid file.
func (l *Lock) File() *os.File {
	if l.state != Owned {return nil}

	// From now on the os.File object owns the fd.
	l.state = Borrowed
	return os.NewFile(uintptr(l.fd), l.path)
}

// Classical flock(2) behavior

func shieldIntr(f func() error) error {
	for {
		err := f()
		if err == unix.EINTR {continue}
		return err
	}
}

func (l *Lock) _flock(how int) error {
	what := func() error {return unix.Flock(l.fd, how)}
	if l.retryOnINTR {
		return shieldIntr(what)
	} else {
		return what()
	}
}

func (l *Lock) flock(how int) error {
	if l.fd < 0 {if err := l.Open(); err != nil {return err}}
	return l._flock(how)
}

// LockEx exclusively locks the file using [flock(2)].
// If the file is not yet
// opened it is opened automatically. The returned error represents either
// an error in [flock(2)] itself or a failed [open(2)]. On success nil is
// returned.
//
// Note, this is a blocking syscall. If the lock is not available, it will
// block the calling GO routine. There is no good way to interrupt that
// syscall.
//
// [flock(2)]: https://man7.org/linux/man-pages/man2/flock.2.html
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func (l *Lock) LockEx() error {return l.flock(unix.LOCK_EX)}

// TryLockEx is the non-blocking version of [*Lock.LockEx]. If the lock
// cannot be acquired [unix.EWOULDBLOCK] is returned.
func (l *Lock) TryLockEx() error {return l.flock(unix.LOCK_EX|unix.LOCK_NB)}

// LockSh represents a shared lock on the file using [flock(2)].
// If the file is not yet
// opened it is opened automatically. The returned error represents either
// an error in [flock(2)] itself or a failed [open(2)]. On success nil is
// returned.
//
// Note, this is a blocking syscall. If the lock is not available, it will
// block the calling GO routine. There is no good way to interrupt that
// syscall.
//
// [flock(2)]: https://man7.org/linux/man-pages/man2/flock.2.html
// [open(2)]: https://man7.org/linux/man-pages/man2/open.2.html
func (l *Lock) LockSh() error {return l.flock(unix.LOCK_SH)}

// TryLockSh is the non-blocking version of [*Lock.LockSh]. If the lock
// cannot be acquired [unix.EWOULDBLOCK] is returned.
func (l *Lock) TryLockSh() error {return l.flock(unix.LOCK_SH|unix.LOCK_NB)}

// Unlock releases the lock. It is not an error to unlock an already released
// lock.
//
// If the underlying file is closed, it will not be opened and nil is returned.
//
// Any error comes from the underlying [flock(2)] syscall.
//
// [flock(2)]: https://man7.org/linux/man-pages/man2/flock.2.html
func (l *Lock) Unlock() error {
	if l.fd < 0 {return nil}
	return l._flock(unix.LOCK_UN)
}

// Local Variables:
// tab-width: 4
// End:

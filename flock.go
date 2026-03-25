package flock

import (
	"os"
	"errors"

	"golang.org/x/sys/unix"
)

// that way the user does not have to import "unix" or similar
const SEEK_SET int16 = unix.SEEK_SET
const SEEK_CUR int16 = unix.SEEK_CUR
const SEEK_END int16 = unix.SEEK_END

type Lock struct {
	path        string
	fd          int
	retryOnINTR bool
	openMode    int
	perms       uint32
	toBeClosed  bool
}

type Opts struct {
	path        string
	fd          int
	closeOnExec bool
	isDir       bool
	retryOnINTR bool
	create      bool
	truncate    bool
	append      bool
	openMode    int
	perms       uint32
	toBeClosed  bool
}

type Option func(*Opts)

func WithPath(param string) Option {return func(o *Opts){o.path = param}}
func WithFd(param int) Option {return func(o *Opts){
	if param >= 0 {
		o.toBeClosed = false
		o.fd = param
	}
}}
func WithCloseOnExec() Option {return func(o *Opts){o.closeOnExec = true}}
func WithIsDir() Option {return func(o *Opts){o.isDir = true}}
func WithRdWr() Option {return func(o *Opts){o.openMode = unix.O_RDWR}}
func WithWrOnly() Option {return func(o *Opts){o.openMode = unix.O_WRONLY}}
func WithTruncate() Option {return func(o *Opts){o.truncate = true}}
func WithAppend() Option {return func(o *Opts){o.append = true}}
func WithCreate(perms ...uint32) Option {return func(o *Opts){
	if len(perms) > 0 {
		o.perms = 0
		for _, p := range perms {o.perms |= p}
	} else {
		o.perms = 0o600
	}
	o.create = true
}}
func WithoutIntrRetry() Option {return func(o *Opts){o.retryOnINTR = false}}

func New(opts ...Option) *Lock {
	os := &Opts{
		fd: -1,
		toBeClosed: true,
		retryOnINTR: true,
		openMode: unix.O_RDONLY,
	}
	for _, o := range(opts) {o(os)}

	flags := os.openMode
	if os.create {flags |= unix.O_CREAT}
	if os.append {flags |= unix.O_APPEND}
	if os.truncate {flags |= unix.O_TRUNC}
	if os.isDir {flags |= unix.O_DIRECTORY}
	if os.closeOnExec {flags |= unix.O_CLOEXEC}

	return &Lock{
		path:        os.path,
		fd:          os.fd,
		retryOnINTR: os.retryOnINTR,
		openMode:    flags,
		perms:       os.perms,
		toBeClosed:  os.toBeClosed,
	}
}

func (l *Lock) Close() error {
	if l.fd >= 0 && l.toBeClosed {
		f := l.fd
		l.fd = -1
		l.toBeClosed = true
		return unix.Close(f)
	}
	l.fd = -1
	l.toBeClosed = true
	return nil
}

var ErrAlreadyOpen error = errors.New("file is already open")
func (l *Lock) Open() error {
	if l.fd >= 0 {return ErrAlreadyOpen}
	f, err := unix.Open(l.path, l.openMode, l.perms)
	if err != nil {return err}
	l.fd = f
	l.toBeClosed = true
	return nil
}

func (l *Lock) Path() string {return l.path}
func (l *Lock) Fd() int {return l.fd}

func (l *Lock) File() *os.File {
	// From now on the os.File object owns the fd.
	l.toBeClosed = false
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

func (l *Lock) LockEx() error {return l.flock(unix.LOCK_EX)}
func (l *Lock) TryLockEx() error {return l.flock(unix.LOCK_EX|unix.LOCK_NB)}
func (l *Lock) LockSh() error {return l.flock(unix.LOCK_SH)}
func (l *Lock) TryLockSh() error {return l.flock(unix.LOCK_SH|unix.LOCK_NB)}
func (l *Lock) Unlock() error {
	if l.fd < 0 {return nil}
	return l._flock(unix.LOCK_UN)
}

// Local Variables:
// tab-width: 4
// End:

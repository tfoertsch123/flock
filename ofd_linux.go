//go:build linux
package flock

import (
	"golang.org/x/sys/unix"
)

// Byte range locks with flock(2) semantics (the lock is tied to the fd
// not the process. Inheritance via fork/dup also works).

func (l *Lock) lockRange(
	how int16, op int,
	start, len int64, whence []int16,
) error {
	if l.fd < 0 {if err := l.Open(); err != nil {return err}}
	return l._lockRange(how, op, start, len, whence)
}

func (l *Lock) _lockRange(
	how int16, op int,
	start, len int64, whence []int16,
) error {
	w := int16(unix.SEEK_SET)
	if whence != nil {w = whence[0]}
	cmd := unix.Flock_t{
		Type:   how,
		Whence: w,
		Start:  start,
		Len:    len,
	}
	what := func() error {
		return unix.FcntlFlock(uintptr(l.fd), op, &cmd)
	}
	if l.retryOnINTR {
		return shieldIntr(what)
	} else {
		return what()
	}
}

// LockRangeEx exclusively locks the range from start to start+len-1. The
// position at start+len is not locked. So, LockRangeEx(10, 1) locks
// exactly 1 byte position.
//
// If len==0, the range from the start position up to positive infinity
// is locked. This does not depend on the file size.
//
// The optional whence parameter defines how to interpret the start parameter.
// If omitted, [SEEK_SET] is used. If multiple whence parameters are passed
// only the first one is used.
//
//  - [SEEK_SET]: start is interpreted relative to the beginning of the file.
//  - [SEEK_CUR]: start is interpreted relative to the current file position.
//  - [SEEK_END]: start is interpreted relative to the current end of the file.
//
// Both, start and len can be negative as long as the resulting starting
// position is not negative.
func (l *Lock) LockRangeEx(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_WRLCK, unix.F_OFD_SETLKW, start, len, whence)
}

// TryLockRangeEx is the non-blocking version of [*Lock.LockRangeEx]. If the
// lock cannot be acquired [unix.EWOULDBLOCK] is returned.
func (l *Lock) TryLockRangeEx(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_WRLCK, unix.F_OFD_SETLK, start, len, whence)
}

// LockRangeSh acquires a shared lock. The parameters are the same as for
// [*Lock.LockRangeEx]
func (l *Lock) LockRangeSh(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_RDLCK, unix.F_OFD_SETLKW, start, len, whence)
}

// TryLockRangeSh is the non-blocking version of LockRangeSh.
// The parameters are the same as for [*Lock.LockRangeEx]. If the lock
// cannot be acquired [unix.EWOULDBLOCK] is returned.
func (l *Lock) TryLockRangeSh(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_RDLCK, unix.F_OFD_SETLK, start, len, whence)
}

// UnlockRange releases a range lock.
// The parameters are the same as for [*Lock.LockRangeEx].
//
// If you simply want to release all existing range locks for a file, you
// can do this:
//
//  lck.UnlockRange(0, 0)
//
// This releases all locks no matter if exclusive or shared.
// Since len==0, the entire range from start up to infinity is released.
// Since start==0 and no whence parameter is passed implying [SEEK_SET],
// this identifies the start of the file.
func (l *Lock) UnlockRange(start, len int64, whence ...int16) error {
	if l.fd < 0 {return nil}
	return l._lockRange(unix.F_UNLCK, unix.F_OFD_SETLK, start, len, whence)
}

// Local Variables:
// tab-width: 4
// End:

//go:build linux
package flock

import (
	"errors"
	"golang.org/x/sys/unix"
)

// Byte range locks with flock(2) semantics (the lock is tied to the fd
// not the process. Inheritance via fork/dup also works).

var ErrMode error = errors.New("file is read-only")
func (l *Lock) lockRange(
	how int16, op int,
	start, len int64, whence []int16,
) error {
	if l.fd < 0 {
		if how == unix.F_WRLCK && l.openMode == unix.O_RDONLY {return ErrMode}
		if err := l.Open(); err != nil {return err}
	}
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

func (l *Lock) LockRangeEx(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_WRLCK, unix.F_OFD_SETLKW, start, len, whence)
}

func (l *Lock) TryLockRangeEx(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_WRLCK, unix.F_OFD_SETLK, start, len, whence)
}

func (l *Lock) LockRangeSh(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_RDLCK, unix.F_OFD_SETLKW, start, len, whence)
}

func (l *Lock) TryLockRangeSh(start, len int64, whence ...int16) error {
	return l.lockRange(unix.F_RDLCK, unix.F_OFD_SETLK, start, len, whence)
}

func (l *Lock) UnlockRange(start, len int64, whence ...int16) error {
	if l.fd < 0 {return nil}
	return l.lockRange(unix.F_UNLCK, unix.F_OFD_SETLK, start, len, whence)
}

// Local Variables:
// tab-width: 4
// End:

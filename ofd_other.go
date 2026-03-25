//go:build !linux
package flock

import (
	"errors"
)

// Byte range locks with flock(2) semantics (the lock is tied to the fd
// not the process. Inheritance via fork/dup also works).

var ErrUnsupported error = errors.New("range locks not supported by OS")
func (l *Lock) LockRangeEx(start, len int64, whence ...int16) error {
	return ErrUnsupported
}

func (l *Lock) TryLockRangeEx(start, len int64, whence ...int16) error {
	return ErrUnsupported
}

func (l *Lock) LockRangeSh(start, len int64, whence ...int16) error {
	return ErrUnsupported
}

func (l *Lock) TryLockRangeSh(start, len int64, whence ...int16) error {
	return ErrUnsupported
}

func (l *Lock) UnlockRange(start, len int64, whence ...int16) error {
	return ErrUnsupported
}

// Local Variables:
// tab-width: 4
// End:

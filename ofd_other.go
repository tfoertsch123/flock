//go:build !linux
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

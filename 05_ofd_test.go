//go:build linux
package flock

import (
	"testing"
	"os"
	"golang.org/x/sys/unix"
)

// This only runs on linux. So, we can use linux-specific stuff.
// OFD locks are linux-specific anyway.

func TestOFD1(t *testing.T) {
	tmpf, err := os.CreateTemp("", "ofd-test")
	if err != nil {t.Fatalf(`CreateTemp: %v`, err)}
	defer os.Remove(tmpf.Name()) // clean up	
	defer tmpf.Close()

	// RDWR or WRONLY is required for exclusive locking
	l1 := New(WithPath(tmpf.Name()), WithRdWr())
	defer l1.Close()

	l2 := New(WithPath(tmpf.Name()), WithRdWr())
	defer l2.Close()

	l3 := New(WithPath(tmpf.Name()), WithWrOnly(), WithoutIntrRetry())
	defer l3.Close()

	if err := l1.LockRangeEx(0, 1); err != nil {
		t.Fatalf(`LockRangeEx(l1, 0, 1): %v`, err)
	}

	if err := l1.LockRangeSh(2, 1); err != nil {
		t.Fatalf(`LockRangeSh(l1, 2, 1): %v`, err)
	}

	if err := l2.TryLockRangeEx(1, 1); err != nil {
		t.Fatalf(`TryLockRangeEx(l2, 1, 1): %v`, err)
	}
	
	if err := l2.TryLockRangeSh(2, 1); err != nil {
		t.Fatalf(`TryLockRangeSh(l2, 2, 1): %v`, err)
	}

	// Now, 0:1 is ex-locked by l1, 1:1 is ex-locked by l2 and 2:1
	// is sh-locked by both. So, l3 should NOT be able to:
	// sh-lock 0:1
	// sh-lock 1:1
	// ex-lock 2:1

	if err := l3.TryLockRangeSh(0, 1); err == nil {
		t.Error(`TryLockRangeSh(l3, 0, 1): unexpected success`)
	}
	if err := l3.TryLockRangeSh(1, 1); err == nil {
		t.Error(`TryLockRangeSh(l3, 1, 1): unexpected success`)
	}
	if err := l3.TryLockRangeEx(2, 1); err == nil {
		t.Error(`TryLockRangeEx(l3, 2, 1): unexpected success`)
	}

	// Unlocking both pieces of l1 in one call
	if err := l1.UnlockRange(0, 3); err != nil {
		t.Fatalf(`UnlockRange(l1, 0, 3): %v`, err)
	}

	// len=0 means from start to the end of the file
	if err := l2.UnlockRange(0, 0); err != nil {
		t.Fatalf(`UnlockRange(l2, 0, 0): %v`, err)
	}

	// now locking the entire range should succeed
	if err := l3.TryLockRangeEx(0, 3); err != nil {
		t.Errorf(`TryLockRangeEx(l3, 0, 3): %v`, err)
	}

	// At this point, the only lock held is 0:3 by l3. Now we test the
	// optional whence parameter.

	if err := tmpf.Truncate(4); err != nil {
		t.Fatalf(`tmpf.Truncate(4): %v`, err)
	}

	// EOF is at 4. So, this unlocks just 1 byte, 2:1 because the last byte
	// before EOF is not locked
	if err := l3.UnlockRange(0, -2, SEEK_END); err != nil {
		t.Fatalf(`UnlockRange(l3, 0, -2, SEEK_END): %v`, err)
	}

	// This tries the range 1:1. Since 0:2 is still locked, it should fail.
	if err := l1.TryLockRangeEx(-2, -1, SEEK_END); err == nil {
		t.Fatalf(`TryLockRangeEx(l1, -2, -1, SEEK_END): unexpected success`)
	}

	// This tries the range 2:1. Since only 0:2 is locked, it should succeed.
	if err := l1.TryLockRangeSh(-2, 1, SEEK_END); err != nil {
		t.Fatalf(`TryLockRangeSh(l1, -2, 1, SEEK_END): %v`, err)
	}

	if err := l3.UnlockRange(0, -3, SEEK_END); err != nil {
		t.Fatalf(`UnlockRange(l3, 0, -3, SEEK_END): %v`, err)
	}

	// At this point, 0:1 is ex-locked by l3 and 2:1 is sh-locked by l1.
	// So, l2 should be able to sh-lock 1:3.

	if _, err := unix.Seek(l2.Fd(), 2, unix.SEEK_SET); err != nil {
		t.Fatalf(`Seek: %v`, err)
	}

	// This tries the range 2:1. Since only 0:2 is locked, it should succeed.
	// Note, len=0 means to the end of the file, no matter how large the file
	// grows.
	if err := l2.TryLockRangeSh(-2, 0, SEEK_CUR); err == nil {
		t.Fatal(`TryLockRangeSh(l2, -2, 0, SEEK_CUR): unexpected success`)
	}

	if err := l2.TryLockRangeSh(-1, 0, SEEK_CUR); err != nil {
		t.Fatalf(`TryLockRangeSh(l2, -1, 0, SEEK_CUR): %v`, err)
	}

	// At this point, 0:1 is ex-locked by l3 and 2:1 is sh-locked by l1.
	// 1:inf is sh-locked by l2.
	// So, ex-locking 100:100 by l1 should fail while sh-locking should
	// succeed.
	if err := l1.TryLockRangeEx(100, 100, SEEK_SET); err == nil {
		t.Fatal(`TryLockRangeEx(l1, 100, 100, SEEK_SET): unexpected success`)
	}

	if err := l1.TryLockRangeSh(100, 100, SEEK_SET); err != nil {
		t.Fatalf(`TryLockRangeSh(l1, 100, 100, SEEK_SET): %v`, err)
	}

	// Unlock on closed lock
	l3.Close()
	if err := l3.UnlockRange(0, 0); err != nil {
		t.Fatalf(`UnlockRange(closed): %v`, err)
	}
}

func TestOFD2(t *testing.T) {
	l := New(WithPath("."))
	if err := l.TryLockRangeEx(0, 1); err != unix.EBADF {
		t.Errorf(`TryLockRangeEx(read-only): %v`, err)
	}
	l = New(WithPath("."), WithRdWr())
	if err := l.TryLockRangeEx(0, 1); err == nil {
		t.Errorf(`TryLockRangeEx(read-only): %v`, err)
	}
	defer l.Close()
}

// Local Variables:
// tab-width: 4
// End:

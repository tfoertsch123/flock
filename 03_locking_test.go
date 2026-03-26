package flock

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestLock(t *testing.T) {
	if New(WithPath("."), WithRdWr()).LockSh() == nil {
		t.Fatal(`Opening a directory for writing should fail`)
	}

	l1 := New(WithPath("."), WithoutIntrRetry())
	if err := l1.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	defer l1.Close()

	if err := l1.LockSh(); err != nil {t.Fatalf(`LockSh(.): %v`, err)}

	l2 := New(WithPath("."), WithoutIntrRetry())
	if err := l2.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	defer l2.Close()

	// Should succeed togther with the sh-locked l1
	if err := l2.TryLockSh(); err != nil {t.Fatalf(`TryLockSh(.): %v`, err)}
	if err := l2.Unlock(); err != nil {t.Fatalf(`Unlock(.): %v`, err)}

	// Should fail because l1 is sh-locked
	if err := l2.TryLockEx(); err != unix.EWOULDBLOCK {
		t.Fatalf(`TryLockEx(.): %v`, err)
	}

	// Now unlock l1
	if err := l1.Unlock(); err != nil {t.Fatalf(`Unlock(.): %v`, err)}

	// ex-lock should succeed
	if err := l2.LockEx(); err != nil {t.Fatalf(`LockEx(.): %v`, err)}

	// should fail because l2 is locked
	if err := l1.TryLockSh(); err != unix.EWOULDBLOCK {
		t.Fatalf(`TryLockSh(.): %v`, err)
	}

	if err := l2.Unlock(); err != nil {t.Fatalf(`Unlock(.): %v`, err)}

	// Unlock on a closed lock
	l2.Close()
	if err := l2.Unlock(); err != nil {t.Fatalf(`Unlock(closed): %v`, err)}
}

// Local Variables:
// tab-width: 4
// End:

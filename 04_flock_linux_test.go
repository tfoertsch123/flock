//go:build linux
package flock

import (
	"time"
	"testing"
	"os"
	"os/exec"
)

func run(t *testing.T, p ...string) {
    cmd := exec.Command(p[0], p[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        t.Fatalf("sleep failed: %v", err)
    }
}

func now() string {return time.Now().Format("2006-01-02 15:04:05.000000")}

func TestCloExec1(t *testing.T) {
	// create 2 locks. One with and the other without CLOEXEC.
	l1 := New(WithPath("."))
	if err := l1.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	defer l1.Close()

	// locking the one without CLOEXEC
	if err := l1.TryLockSh(); err != nil {t.Fatalf(`TryLockSh(.): %v`, err)}

	l2 := New(WithPath("."), WithCloseOnExec())
	if err := l2.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	defer l2.Close()

	if err := l2.TryLockEx(); err == nil {t.Fatalf(`TryLockEx(.): succeeded`)}

	c := make(chan string, 10)

	// Now we create 2 subprocesses one sleeps for 100ms the other for 300ms.
	// Each one drops a message into the channel when done. Then we ex-lock l2.
	// If CLOEXEC (or rather its absence) works, we should get the lock only
	// after both processes exit. Since the 100ms process exits significantly
	// earlier than the 300ms process, its exit signal should always arrive
	// first in the channel. In particular, it must come before we can get
	// the lock.

	t.Logf("%s: l1.fd: %d, l2.fd: %d", now(), l1.Fd(), l2.Fd())
	go func(){
		t.Logf("%s: waiting for l2", now())
		if err := l2.LockEx(); err != nil {t.Fatalf(`LockEx(.): %v`, err)}
		c <- "locked"
	}()
	go func(){
		run(t, "sleep", ".3")
		c <- "slept 300"
	}()
	go func(){
		run(t, "sleep", ".1")
		c <- "slept 100"		// this should arrive first
	}()
	// go func(){
	// 	run(t, "ls", "-l", "/proc/self/fd")
	// }()

	tm := time.NewTimer(50*time.Millisecond)
	var first *string
	OUTER: for {
		select {
		case <-tm.C:
			// We can't unlock here. That would release the lock.
			// We need to close the file without unlocking.
			l1.Close()
			t.Logf("%s: l1 closed", now())
		case s := <-c:
			t.Logf("%s: %s", now(), s)
			if first == nil {first = &s}
			if s == "slept 300" {break OUTER}
		}
	}
	if *first != "slept 100" {t.Fatalf(`unexpected order: %v`, first)}
}

func TestCloExec2(t *testing.T) {
	// same as above. Only now both have CLOEXEC set
	l1 := New(WithPath("."), WithCloseOnExec())
	if err := l1.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	defer l1.Close()

	// locking the one without CLOEXEC
	if err := l1.TryLockSh(); err != nil {t.Fatalf(`TryLockSh(.): %v`, err)}

	l2 := New(WithPath("."), WithCloseOnExec())
	if err := l2.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	defer l2.Close()

	if err := l2.TryLockEx(); err == nil {t.Fatalf(`TryLockEx(.): succeeded`)}

	c := make(chan string, 10)

	// In this case we only need 1 subprocess. It sleeps for 100ms. When done
	// a message is dropped in the channel. Then we ex-lock l2. With CLOEXEC
	// the subprocess should not know about the file descriptor. So, we should
	// get l2 as soon as l1 is released.

	t.Logf("%s: l1.fd: %d, l2.fd: %d", now(), l1.Fd(), l2.Fd())
	go func(){
		t.Logf("%s: waiting for l2", now())
		if err := l2.LockEx(); err != nil {t.Fatalf(`LockEx(.): %v`, err)}
		c <- "locked"
	}()
	go func(){
		run(t, "sleep", ".2")
		c <- "slept 200"
	}()
	// go func(){
	// 	run(t, "ls", "-l", "/proc/self/fd")
	// }()

	tm := time.NewTimer(50*time.Millisecond)
	var first *string
	OUTER: for {
		select {
		case <-tm.C:
			// We can't unlock here. That would release the lock.
			// We need to close the file without unlocking.
			l1.Close()
			t.Logf("%s: l1 closed", now())
		case s := <-c:
			t.Logf("%s: %s", now(), s)
			if first == nil {first = &s}
			if s == "slept 200" {break OUTER}
		}
	}
	if *first != "locked" {t.Fatalf(`unexpected order: %v`, *first)}
}

// Local Variables:
// tab-width: 4
// End:

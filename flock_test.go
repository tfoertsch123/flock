//go:build linux
package flock_test

import (
	"os"
	"os/exec"
	"log"
	"sync"
	"fmt"
	"github.com/tfoertsch123/flock"
)

func start(cmdline string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf(`"%s" failed: %v`, cmdline, err)
	}
	return cmd
}

func Example_flock () {
	var wg sync.WaitGroup

	lck := flock.New(
		flock.WithPath("."),
		flock.WithIsDir(),		// require path to be a directory
	)

	// Open the file/directory and acquire a shared lock.
	if err := lck.TryLockSh(); err != nil {
		log.Fatalf(`Couldn't lock(p1) .: %v`, err)
	}

	// The shell inherits the lock. (no WithCloseOnExec())
	p1 := start("sleep .5; echo P1: slept")
	wg.Go(func() {p1.Wait()})

	// this only closes the file. It does not release the lock
	lck.Close()

	lck = flock.New(
		flock.WithPath("."),
	)

	// Just open, no lock
	if err := lck.Open(); err != nil {
		log.Fatalf(`Couldn't open(p2) .: %v`, err)
	}

	// "flock -s" at the shell level is the same as flock.LockSh().
	// This acquires a shared lock with does not conflict with the
	// lock held by p1. So, the "echo" will not wait for p1 to finish.
	p2 := start(fmt.Sprintf("flock -s %d; echo P2: lock acquired", lck.Fd()))
	wg.Go(func() {p2.Wait()})
	
	lck.Close()

	// Open the file yet another time
	lck = flock.New(
		flock.WithPath("."),
	)

	// Just open
	if err := lck.Open(); err != nil {
		log.Fatalf(`Couldn't open(p3) .: %v`, err)
	}

	// "flock -x" at the shell level is the same as flock.LockEx().
	// This acquires an exclusive lock with does conflict with the
	// lock held by p1. So, the "echo" will wait for p1 to finish.
	p3 := start(fmt.Sprintf("flock -x %d; echo P3: lock acquired", lck.Fd()))
	wg.Go(func() {p3.Wait()})
	
	lck.Close()

	wg.Wait()

	// Output:
	// P2: lock acquired
	// P1: slept
	// P3: lock acquired
}

// Local Variables:
// tab-width: 4
// End:

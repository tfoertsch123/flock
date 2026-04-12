//go:build linux
package flock

import (
	"testing"
	"golang.org/x/sys/unix"
)

// This is a rather artificial test. It's actually pretty hard to reliably
// make a syscall return EINTR in go.

func TestEINTR(t *testing.T) {
	errs := []error{
		unix.EINTR,
		unix.EINTR,
		unix.EWOULDBLOCK,
		nil,
	}
	i := 0
	res := shieldIntr(func() error {
		i++
		if i<=len(errs) {return errs[i-1]}
		t.Fatal("errs out of bound")
		return nil
	})

	if res != unix.EWOULDBLOCK {
		t.Errorf("expected EWOULDBLOCK, got %v", res)
	}
}

// Local Variables:
// tab-width: 4
// End:

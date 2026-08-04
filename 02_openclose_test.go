package flock

import (
	"os"
	"errors"
	"path/filepath"
	"testing"
	"runtime"
	"golang.org/x/sys/unix"
)

func checkFdValid(t *testing.T, fd int, exp_ok bool) {
	// check if the fd is valid
	stat := &unix.Stat_t{}
	err := unix.Fstat(fd, stat)
	if (err == nil) != exp_ok {
		_, file, line, ok := runtime.Caller(1)
		if ! ok {
			file = "invalid stack frame"
			line = 0
		}
		t.Errorf(`%s(%d): valid fd: expected %v, got %v`, file, line,
			exp_ok, err == nil)
	}
}

func TestOpen1(t *testing.T) {
	if New().Open() == nil {t.Fatal(`Open without path should fail`)}
	if New(WithPath("."), WithRdWr()).Open() == nil {
		t.Fatal(`Opening a directory for writing should fail`)
	}
	
	var l *Lock
	l = New(WithPath("."))
	if err := l.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	if !l.toBeClosed {t.Fatal(`toBeClosed != true`)}

	if err := l.Open(); err != ErrAlreadyOpen {
		t.Fatalf(`opening the same object twice should return ErrAlreadyOpen`)
	}

	fd := l.Fd()
	checkFdValid(t, fd, true)

	if err := l.Close(); err != nil {t.Fatalf("Close: %v", err)}
	checkFdValid(t, fd, false)
}

func TestOpen2(t *testing.T) {
	if New().Open() == nil {t.Fatal(`Open without path should fail`)}
	
	var l *Lock
	l = New(WithPath("."))
	if err := l.Open(); err != nil {t.Fatalf(`Open(.): %v`, err)}
	if !l.toBeClosed {t.Fatal(`toBeClosed != true`)}

	fd := l.Fd()
	checkFdValid(t, fd, true)

	fh := l.File()
	if fh == nil {t.Fatal(`File() returns nil`)}
	if l.toBeClosed {t.Fatal(`toBeClosed != false`)}

	if err := l.Close(); err != nil {t.Fatalf("Close: %v", err)}
	// fd must still be valid. It is now owned by fh
	checkFdValid(t, fd, true)

	if l.File() != nil {t.Fatal("File() after Close() must return nil")}

	if err := l.Open(); err != nil {t.Fatalf(`Open after Close(): %v`, err)}
	if !l.toBeClosed {t.Fatal(`toBeClosed != true`)}
	fd2 := l.Fd()
	checkFdValid(t, fd2, true)
	if err := l.Close(); err != nil {t.Fatalf("Close: %v", err)}
	checkFdValid(t, fd2, false)
	if fd == fd2 {
		t.Fatal(`fd after re-open should differ b/c fd is not closed yet`)
	}

	// closing the *os.File closes it finally
	fh.Close()
	checkFdValid(t, fd, false)
}

func TestWithFd(t *testing.T) {
	tmpf, err := os.CreateTemp("", "WithFd-test")
	if err != nil {t.Fatalf(`CreateTemp: %v`, err)}
	defer os.Remove(tmpf.Name()) // clean up	

	fd1 := int(tmpf.Fd())
	checkFdValid(t, fd1, true)

	// A lock created with WithFd() does not own the file descriptor.
	// So, Close() must leave it intact.
	l1 := New(
		WithPath(tmpf.Name()),
		WithFd(fd1),
	)
	l1.Close()

	checkFdValid(t, fd1, true)
}

func TestOpenAt(t *testing.T) {
	dir := t.TempDir()
	dirfh, err := os.Open(dir)
	if err != nil {
		t.Fatalf("Cannot open temp dir %v", err)
	}

	nm := "testlock"

	l := New(WithPath(filepath.Join(dir, nm)))
	err = l.Open()
	if !errors.Is(err, unix.ENOENT) {
		t.Fatalf("Expecting ENOENT, got %v", err)
	}

	l = New(WithPath(nm), WithPathAt(int(dirfh.Fd())), WithCreate())
	err = l.Open()
	if err != nil {
		t.Fatalf("Cannot open lock %v", err)
	}
	l.Close()

	l = New(WithPath(filepath.Join(dir, nm)))
	err = l.Open()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
}

// Local Variables:
// tab-width: 4
// End:

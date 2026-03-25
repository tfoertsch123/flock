package flock

import (
	"testing"
	"golang.org/x/sys/unix"
)

func TestNewEmpty(t *testing.T) {
	var l *Lock
	l = New()
	if l == nil {t.Fatal(`New() != nil`)}
	if l.path != "" {t.Fatal(`New().path != ""`)}
	if l.fd != -1 {t.Fatal(`New().fd != -1`)}
	if !l.retryOnINTR {t.Fatal(`New().retryOnINTR != true`)}
	if l.openMode != unix.O_RDONLY {t.Fatal(`New().openMode != O_RDONLY`)}
	if l.perms != 0 {t.Fatal(`New().perms != 0`)}
	if !l.toBeClosed {t.Fatal(`New().toBeClosed != true`)}
}

func TestNewAllOpenModifiers(t *testing.T) {
	var l *Lock
	l = New(
		WithCloseOnExec(),
		WithIsDir(),
		WithAppend(),
		WithTruncate(),
		WithCreate(),
	)
	m := unix.O_RDONLY | unix.O_CREAT | unix.O_DIRECTORY | unix.O_CLOEXEC |
		 unix.O_APPEND | unix.O_TRUNC
	if l.openMode != m {t.Fatalf(`openMode %x != %x`, l.openMode, m)}
}

func TestNewPerms(t *testing.T) {
	var l *Lock
	l = New(
		WithRdWr(),
		WithCreate(unix.S_IWUSR, unix.S_IRUSR, unix.S_IXOTH, unix.S_IWGRP),
	)
	m := unix.O_RDWR | unix.O_CREAT
	if l.openMode != m {t.Fatalf(`openMode %x != %x`, l.openMode, m)}
	p := uint32(unix.S_IWUSR|unix.S_IRUSR|unix.S_IXOTH|unix.S_IWGRP)
	if l.perms != p {t.Fatalf(`perms %o != %o`, l.perms, p)}
}

func TestNewRDWR_WRONLY(t *testing.T) {
	var l *Lock
	// We allow both RdWr and WrOnly in the same call. The last one wins.
	l = New(
		WithRdWr(),
		WithWrOnly(),
	)
	m := unix.O_WRONLY
	if l.openMode != m {t.Fatalf(`openMode %x != %x`, l.openMode, m)}
}

func TestNewPathFd(t *testing.T) {
	var l *Lock
	l = New(
		WithoutIntrRetry(),
		WithPath("."),
		WithFd(15),
	)
	if l.retryOnINTR {t.Fatal(`retryOnINTR != false`)}
	if l.path != "." {t.Fatal(`path != "."`)}
	if l.fd != 15 {t.Fatal(`fd != 15`)}
	if l.toBeClosed {t.Fatal(`toBeClosed != false`)}
	if l.Path() != "." {t.Fatal(`Path() != "."`)}
	if l.Fd() != 15 {t.Fatal(`Fd() != 15`)}

	l = New(
		WithFd(-15),
	)
	if l.fd != -1 {t.Fatal(`WithFd(<0) is invalid`)}
	if !l.toBeClosed {t.Fatal(`toBeClosed != true`)}
}

// Local Variables:
// tab-width: 4
// End:

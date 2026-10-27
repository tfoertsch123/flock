package flock

import (
	"testing"
	"golang.org/x/sys/unix"
)

func TestState(t *testing.T) {
	tests := []struct {
		i State
		s string
	}{
		{Invalid, "Invalid"},
		{Init, "Init"},
		{Owned, "Owned"},
		{Borrowed, "Borrowed"},
		{Closed, "Closed"},
	}

	for _, x := range tests {
		if x.i.String() != x.s {
			t.Errorf(`State(%d), exp: %s`, x.i, x.s)
		}
	}
	if State(len(tests)).String() != "Invalid" {
		t.Error(`State(>max).String() != "Invalid"`)
	}
}

func TestNewEmpty(t *testing.T) {
	var l *Lock
	l = New()
	if l == nil {t.Error(`New() != nil`)}
	if l.path != "" {t.Error(`New().path != ""`)}
	if l.fd != -1 {t.Error(`New().fd != -1`)}
	if !l.retryOnINTR {t.Error(`New().retryOnINTR != true`)}
	if l.openMode != unix.O_RDONLY {t.Error(`New().openMode != O_RDONLY`)}
	if l.perms != 0 {t.Error(`New().perms != 0`)}
	if l.state != Init {t.Error(`New().state != Init`)}
	if !l.OwningFd() {t.Error(`New().OwningFd() != true`)}
	if l.IsOpen() {t.Error(`New().IsOpen() != false`)}
	if l.IsClosed() {t.Error(`New().IsClosed() != false`)}
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
	if l.openMode != m {t.Errorf(`openMode %x != %x`, l.openMode, m)}
	if l.perms != 0o600 {t.Errorf(`WithCreate(): perms: %o`, l.perms)}
}

func TestNewPerms(t *testing.T) {
	var l *Lock
	l = New(
		WithRdWr(),
		WithCreate(unix.S_IWUSR, unix.S_IRUSR, unix.S_IXOTH, unix.S_IWGRP, 3),
	)
	m := unix.O_RDWR | unix.O_CREAT
	if l.openMode != m {t.Errorf(`openMode %x != %x`, l.openMode, m)}
	p := uint32(0o623)
	if l.perms != p {t.Errorf(`perms %o != %o`, l.perms, p)}
}

func TestNewRDWR_WRONLY(t *testing.T) {
	var l *Lock
	// We allow both RdWr and WrOnly in the same call. The last one wins.
	l = New(
		WithRdWr(),
		WithWrOnly(),
	)
	m := unix.O_WRONLY
	if l.openMode != m {t.Errorf(`openMode %x != %x`, l.openMode, m)}
}

func TestNewPathFd(t *testing.T) {
	var l *Lock
	l = New(
		WithoutIntrRetry(),
		WithPath("."),
		WithFd(15),
	)
	if l.retryOnINTR {t.Error(`retryOnINTR != false`)}
	if l.path != "." {t.Error(`path != "."`)}
	if l.fd != 15 {t.Error(`fd != 15`)}
	if l.state != Borrowed {t.Error(`state != Borrowed`)}
	if l.OwningFd() {t.Error(`OwningFd() != false`)}
	if !l.IsOpen() {t.Error(`IsOpen() != true`)}
	if l.IsClosed() {t.Error(`IsClosed() != false`)}
	if l.Path() != "." {t.Error(`Path() != "."`)}
	if l.Fd() != 15 {t.Error(`Fd() != 15`)}

	l = New(
		WithFd(-15),
	)
	if l.fd != -1 {t.Fatal(`WithFd(<0) is invalid`)}
	if l.state != Closed {t.Fatal(`state != Closed`)}
}

// Local Variables:
// tab-width: 4
// End:

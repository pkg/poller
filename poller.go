// Package poller provides level triggered readiness notification and
// reliable closing of file descriptors.
package poller

import (
	"io"
	"syscall"
)

// A Poller provides readiness notification and reliable closing of
// registered file descriptors.
type Poller struct {
	poller
}

// New creates a new Poller.
func New() (*Poller, error) {
	p, err := newEpoll()
	return &Poller{poller: p}, err
}

// Register registers a file describtor with the Poller and returns a
// Pollable which can be used for reading/writing as well as readiness
// notification.
//
// File descriptors registered with the poller will be placed into
// non-blocking mode.
func (p *Poller) Register(fd uintptr) (*Pollable, error) {
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return nil, err
	}
	return p.register(fd)
}

// Pollable represents a file descriptor that can be read/written
// and polled/waited for readiness notification.
type Pollable struct {
	fd     uintptr
	cr, cw chan error
	poller
}

// Read reads up to len(b) bytes from the underlying fd. It returns the number of
// bytes read and an error, if any. EOF is signaled by a zero count with
// err set to io.EOF.
//
// Callers to Read will block if there is no data available to read.
func (p *Pollable) Read(b []byte) (int, error) {
	n, e := p.read(b)
	if n < 0 {
		n = 0
	}
	if n == 0 && len(b) > 0 && e == nil {
		return 0, io.EOF
	}
	if e != nil {
		return n, e
	}
	return n, nil
}

func (p *Pollable) read(b []byte) (int, error) {
	for {
		n, e := syscall.Read(int(p.fd), b)
		if e != syscall.EAGAIN {
			return n, e
		}
		if err := p.WaitRead(); err != nil {
			return 0, err
		}
	}
}

// Write writes len(b) bytes to the fd. It returns the number of bytes
// written and an error, if any. Write returns a non-nil error when n !=
// len(b).
//
// Callers to Write will block if there is no buffer capacity available.
func (p *Pollable) Write(b []byte) (int, error) {
	n, e := p.write(b)
	if n < 0 {
		n = 0
	}
	if n != len(b) {
		return n, io.ErrShortWrite
	}
	if e != nil {
		return n, e
	}
	return n, nil
}

func (p *Pollable) write(b []byte) (int, error) {
	for {
		// TODO(dfc) this is wrong
		n, e := syscall.Write(int(p.fd), b)
		if e != syscall.EAGAIN {
			return n, e
		}
		if err := p.WaitWrite(); err != nil {
			return 0, err
		}
	}
}

// WaitRead waits for the Pollable to become ready for
// reading.
func (p *Pollable) WaitRead() error {
	if err := p.poller.waitRead(p); err != nil {
		return err
	}
	return <-p.cr
}

// WaitWrite waits for the Pollable to become ready for
// writing.
func (p *Pollable) WaitWrite() error {
	if err := p.poller.waitWrite(p); err != nil {
		return err
	}
	return <-p.cw
}

func (p *Pollable) wake(mode int, err error) {
	if mode == 'r' {
		p.cr <- err
	} else {
		p.cw <- err
	}
}

type poller interface {
	register(fd uintptr) (*Pollable, error)
	waitRead(*Pollable) error
	waitWrite(*Pollable) error
}

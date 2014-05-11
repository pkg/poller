package poller

// epoll(2) poller

import (
	"os"
	"syscall"
	"unsafe"
)

// New creates a new Poller.
func New() (*Poller, error) {
        p, err := newEpoll()
        return &Poller{poller: p}, err
}

// newEpoll returns an epoll(2) poller implementation.
func newEpoll() (poller, error) {
	fd, err := epollCreate1()
	if err != nil {
		return nil, err
	}
	e := epoll{
		pollfd: fd,
	}
	go e.loop()
	return &e, nil
}

// epoll implements an epoll(2) based poller.
type epoll struct {
	pollfd   uintptr
	eventbuf [0x10]event
	events   []event
}

func (e *epoll) register(fd uintptr) (*Pollable, error) {
	p := Pollable{
		fd:     fd,
		cr:     make(chan error),
		cw:     make(chan error),
		poller: e,
	}
	ev := event{
		events: syscall.EPOLLERR | syscall.EPOLLHUP,
	}
	ev.setdata(&p)
	if err := epollctl(e.pollfd, syscall.EPOLL_CTL_ADD, fd, &ev); err != nil {
		return nil, err
	}
	debug("epoll: register: fd: %v, %p", p.fd, &p)
	return &p, nil
}

func (e *epoll) deregister(p *Pollable) error {
	// TODO(dfc) // wakeup all other waiters ?
	return epollctl(e.pollfd, syscall.EPOLL_CTL_DEL, p.fd, nil)
}

func (e *epoll) loop() {
	defer syscall.Close(int(e.pollfd))
	for {
		ev, err := e.wait()
		if err != nil {
			println(err.Error())
			return
		}
		if ev == nil {
			// timeout / wakeup ?
			continue
		}
		mode := int('r')
		if ev.events&syscall.EPOLLOUT == syscall.EPOLLOUT {
			mode = 'w'
		}
		ev.getdata().wake(mode, nil)
	}
}

func (e *epoll) wait() (*event, error) {
	for len(e.events) == 0 {
		const msec = -1
		n, err := epollwait(e.pollfd, e.eventbuf[0:], msec)
		debug("epoll: epollwait: %v, %v", n, err)
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			return nil, os.NewSyscallError("epoll_wait", err)
		}
		if n == 0 {
			return nil, nil
		}
		e.events = e.eventbuf[0:n]
	}
	ev := e.events[0]
	e.events = e.events[1:]
	debug("epoll: wait: %0x, %p, fd: %v", ev.events, ev.getdata(), ev.getdata().fd)
	return &ev, nil
}

func (e *epoll) waitRead(p *Pollable) error {
	ev := event{
		events: syscall.EPOLLONESHOT | syscall.EPOLLIN,
	}
	ev.setdata(p)
	debug("epoll: waitRead: %d,  %0x, %p", p.fd, ev.events, ev.getdata())
	return epollctl(e.pollfd, syscall.EPOLL_CTL_MOD, p.fd, &ev)
}
func (e *epoll) waitWrite(p *Pollable) error {
	ev := event{
		events: syscall.EPOLLONESHOT | syscall.EPOLLOUT,
	}
	ev.setdata(p)
	debug("epoll: waitWrite: %0x, %p", ev.events, ev.getdata())
	return epollctl(e.pollfd, syscall.EPOLL_CTL_MOD, p.fd, &ev)
}

func epollCreate1() (uintptr, error) {
	fd, _, e := syscall.Syscall(syscall.SYS_EPOLL_CREATE1, syscall.EPOLL_CLOEXEC, 0, 0)
	if e != 0 {
		return 0, e
	}
	return fd, nil
}

// Single-word zero for use when we need a valid pointer to 0 bytes.
var _zero uintptr

func epollwait(epfd uintptr, events []event, msec int) (n int, err error) {
	var _p0 unsafe.Pointer
	if len(events) > 0 {
		_p0 = unsafe.Pointer(&events[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	r0, _, e1 := syscall.Syscall6(syscall.SYS_EPOLL_WAIT, epfd, uintptr(_p0), uintptr(len(events)), uintptr(msec), 0, 0)
	n = int(r0)
	if e1 != 0 {
		err = e1
	}
	return
}

func epollctl(epfd uintptr, op int, fd uintptr, event *event) (err error) {
	_, _, e1 := syscall.RawSyscall6(syscall.SYS_EPOLL_CTL, uintptr(epfd), uintptr(op), fd, uintptr(unsafe.Pointer(event)), 0, 0)
	if e1 != 0 {
		err = e1
	}
	return
}

type event struct {
	events uint32
	data   [2]uint32
}

func (e *event) setdata(p *Pollable) {
	*(**Pollable)(unsafe.Pointer(&e.data[0])) = p
}

func (e *event) getdata() *Pollable {
	return *(**Pollable)(unsafe.Pointer(&e.data[0]))
}

package poller

import (
	"unsafe"
)

// The epoll event stucture is defined, on all platforms, to be a uint32 followed directly by
// a uint64. The former is used by epoll for flags and status, the latter is defined to be the
// target of a C union, but is effectively 64 bits of data that can be used as a key for
// the caller.
//
// But, there is a problem. Ideally we'd like to write this as
//
//    type event struct {
//          events uint32
//          data   *Pollable
//    }
//
// and then epollwait will return us *event's that contain the pointer to the *Pollable that
// the event describes -- however it's not that simple.
//
// 1. On 64bit platforms, a pointer is 64bits, so will be 64bit alliged, as the field before
//    it is 32bits wide, there will be padding added by 6g and that means that the size of
//    event is 16 bytes, not 12, so a []event will be incorrectly aligned.
//
//    Also, because of the invisible padding, the pointer stored in data is truncated, ie only
//    the bottom 4 bytes are preserved, so calling a method on the *Pollable returned will
//    segfault.
//
// 2. On 32bit platforms the opposite is true. The size of data is 32bits, and thus requires no
//    padding for alignment. So, anything pointer stored in data will be returned faithfully, but
//    a []event will be incorrectly aligned because the size of event on 32bit platforms is only
//    8 bytes, not the expected 12.
//
// To overcome the problem, this declaration handles data as 8 bytes of memory then we use
// unsafe to convert the value to/from a *Pollable as required. For 32 bit platforms, the
// declaration is simple and includes the required padding.
//
// On both platforms getdata and setdata inline so the cost of this slight of hand is minimal.
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

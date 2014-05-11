package poller

import (
	"unsafe"
)

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

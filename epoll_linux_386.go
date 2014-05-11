package poller

type event struct {
	events uint32
	_      uint32
	data   *Pollable
}

func (e *event) setdata(p *Pollable) {
	e.data = p
}

func (e *event) getdata() *Pollable {
	return e.data
}

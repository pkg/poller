package poller

type event struct {
	events uint32
	data   *Pollable
}

// +build none

package poller

// select(2) poller

import (
	"os"
)

// newSelect returns a select(2) poller implementation.
func newSelect() (poller, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	_ = r
	return &_select{
		wakefd: w,
	}, nil
}

// _select implements a select(2) based poller.
// The underscore is due to an unfortunate conflict with
// the select keyword.
type _select struct {
	wakefd *os.File
}

package poller

import (
	"os"
	"testing"
	"time"
)

func TestWaitRead(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}
	//defer p.Close()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	r, err := p.Register(pr.Fd())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	w, err := p.Register(pw.Fd())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	ready := make(chan bool)
	go func() {
		<-ready
		<-time.After(50 * time.Millisecond)
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Error(err)
		}
	}()

	read := make(chan []byte)
	go func() {
		var buf [5]byte
		if n, err := r.Read(buf[:]); err != nil || n != 5 {
			t.Error(n, err)
		}
		read <- buf[:]
	}()
	close(ready)
	select {
	case buf := <-read:
		if string(buf) != "hello" {
			t.Fatalf("expected %q, got %q", "hello", buf)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout")
	}
}

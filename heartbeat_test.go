package muxado

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestHeartbeatFast is a regression test for a 0ms
// timeout being detectable
func TestHeartbeatFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, serv := net.Pipe()
	clientHeartbeats := make(chan time.Duration, 1)
	stopServerHBs := make(chan struct{})
	go func() {
		sess := Server(serv, nil)
		typed := NewTypedStreamSession(sess)
		hb := NewHeartbeat(typed, func(d time.Duration) {
			if d == 0 {
				panic("zero")
			}
		}, &HeartbeatConfig{
			Interval:  5 * time.Millisecond,
			Tolerance: 100 * time.Millisecond,
			Type:      defaultStreamType,
		})
		str, err := hb.AcceptTypedStream()
		if err != nil {
			panic(err)
		}
		<-stopServerHBs
		str.Close()

		<-ctx.Done()
		sess.Close()
	}()

	clientSess := Client(client, nil)
	clientTyped := NewTypedStreamSession(clientSess)
	hb := NewHeartbeat(clientTyped, func(d time.Duration) {
		clientHeartbeats <- d
	}, &HeartbeatConfig{
		Interval:  5 * time.Millisecond,
		Tolerance: 500 * time.Millisecond,
		Type:      defaultStreamType,
	})
	hb.Start()

	for i := 0; i < 10; i++ {
		dur := hb.Beat()
		if dur == 0 {
			t.Fatal("beat failed")
		}
		clientT := <-clientHeartbeats
		if clientT == 0 {
			t.Fatal("beat failed")
		}
	}

}

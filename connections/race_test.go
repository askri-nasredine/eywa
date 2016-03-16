package connections

import (
	. "github.com/vivowares/eywa/Godeps/_workspace/src/github.com/smartystreets/goconvey/convey"
	. "github.com/vivowares/eywa/configs"
	. "github.com/vivowares/eywa/utils"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRaceConditions(t *testing.T) {

	SetConfig(&Conf{
		Connections: &ConnectionsConf{
			Registry:      "memory",
			NShards:       4,
			InitShardSize: 8,
			Websocket: &WsConnectionConf{
				RequestQueueSize: 8,
				Timeouts: &WsConnectionTimeoutConf{
					Write:    &JSONDuration{2 * time.Second},
					Read:     &JSONDuration{300 * time.Second},
					Request:  &JSONDuration{1 * time.Second},
					Response: &JSONDuration{2 * time.Second},
				},
				BufferSizes: &WsConnectionBufferSizeConf{
					Write: 1024,
					Read:  1024,
				},
			},
		},
	})

	h := func(c Connection, m *Message, e error) {}
	meta := make(map[string]interface{})

	Convey("burst various sends for race condition test, with wg", t, func() {
		InitializeCM()
		defer CloseCM()
		ws := &fakeWsConn{randomErr: false}
		conn, _ := NewWebsocketConnection("test", ws, h, meta)

		concurrency := 1000
		var wg sync.WaitGroup
		wg.Add(concurrency)
		errs := make([]error, concurrency)
		for i := 0; i < concurrency; i++ {
			go func(index int) {
				var msg []byte
				var err error
				switch rand.Intn(3) {
				case 0:
					msg = []byte("async" + strconv.Itoa(index))
					err = conn.Send(msg)
				case 1:
					msg = []byte("resp" + strconv.Itoa(index))
					err = conn.Response(msg)
				case 2:
					msg = []byte("sync" + strconv.Itoa(index))
					_, err = conn.Request(msg, Config().Connections.Websocket.Timeouts.Response.Duration)
				}
				errs[index] = err
				wg.Done()
			}(i)
		}

		wg.Wait()
		conn.Close()
		conn.Wait()
		So(Count(), ShouldEqual, 0)

		So(ws.closed, ShouldBeTrue)
		So(conn.msgChans.len(), ShouldEqual, 0) //?
		hasClosedConnErr := false
		for _, err := range errs {
			if err != nil && strings.Contains(err.Error(), "connection is closed") {
				hasClosedConnErr = true
			}
		}
		So(hasClosedConnErr, ShouldBeFalse)
	})

	Convey("burst various sends for race condition test, without wg", t, func() {
		InitializeCM()
		ws := &fakeWsConn{randomErr: false}
		conn, _ := NewWebsocketConnection("test", ws, h, meta)

		concurrency := 1000
		errs := make([]error, concurrency)
		for i := 0; i < concurrency; i++ {
			go func(index int) {
				var msg []byte
				var err error
				switch rand.Intn(3) {
				case 0:
					msg = []byte("async" + strconv.Itoa(index))
					err = conn.Send(msg)
				case 1:
					msg = []byte("resp" + strconv.Itoa(index))
					err = conn.Response(msg)
				case 2:
					msg = []byte("sync" + strconv.Itoa(index))
					_, err = conn.Request(msg, Config().Connections.Websocket.Timeouts.Response.Duration)
				}
				errs[index] = err
			}(i)
		}

		CloseCM()
		So(Count(), ShouldEqual, 0)
		So(ws.closed, ShouldBeTrue)
	})

	Convey("successfully closes all created ws connections.", t, func() {
		InitializeCM()

		concurrency := 100
		wss := make([]*fakeWsConn, concurrency)
		for i := 0; i < concurrency; i++ {
			wss[i] = &fakeWsConn{}
		}
		var wg sync.WaitGroup
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(iter int) {
				NewWebsocketConnection("test"+strconv.Itoa(iter), wss[iter], h, meta)
				wg.Done()
			}(i)
		}
		wg.Wait()
		CloseCM()

		So(Count(), ShouldEqual, 0)

		allClosed := true
		for _, ws := range wss {
			if ws.closed == false {
				allClosed = false
			}
		}
		So(allClosed, ShouldBeTrue)
	})

	Convey("real life race conditions, close all underlying ws conn.", t, func() {
		concurrency := 1000
		InitializeCM()
		wss := make([]*fakeWsConn, concurrency)
		for i := 0; i < concurrency; i++ {
			wss[i] = &fakeWsConn{randomErr: rand.Intn(4) == 0}
		}
		conns := make([]*WebsocketConnection, concurrency)
		errs := make([]error, concurrency)
		var wg sync.WaitGroup
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(iter int) {
				time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
				conn, err := NewWebsocketConnection("test"+strconv.Itoa(iter), wss[iter], h, meta)
				conns[iter] = conn
				errs[iter] = err
				switch rand.Intn(3) {
				case 0:
					conn.Send([]byte("async" + strconv.Itoa(iter)))
				case 1:
					conn.Response([]byte("resp" + strconv.Itoa(iter)))
				case 2:
					conn.Request([]byte("sync"+strconv.Itoa(iter)), Config().Connections.Websocket.Timeouts.Response.Duration)
				}
				wg.Done()
			}(i)
		}

		CloseCM()
		So(Count(), ShouldEqual, 0)

		time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second)
		wg.Wait()
		allClosed := true
		for i, ws := range wss {
			if errs[i] == nil && ws.closed == false {
				allClosed = false
			}
		}
		So(allClosed, ShouldBeTrue)
	})

	Convey("successfully closes all created http connections.", t, func() {
		InitializeCM()

		concurrency := 1000
		chs := make([]chan []byte, concurrency)
		for i := 0; i < concurrency; i++ {
			chs[i] = make(chan []byte, 1)
		}
		var wg sync.WaitGroup
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(iter int) {
				NewHttpConnection("test"+strconv.Itoa(iter), chs[iter], func(Connection, *Message, error) {}, nil)
				wg.Done()
			}(i)
		}

		time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second)
		CloseCM()
		wg.Wait()

		So(Count(), ShouldEqual, 0)

		select {
		case <-time.After(3 * time.Second):
			So(false, ShouldBeTrue)
		default:
			for _, ch := range chs {
				<-ch
			}
		}
	})
}

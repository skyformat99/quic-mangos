package quic

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"sync/atomic"

	radix "github.com/armon/go-radix"
	quic "github.com/lucas-clemente/quic-go"
)

type netlocator interface {
	Netloc() string
}

type netloc struct{ *url.URL }

func (n netloc) Netloc() string { return n.Host }

type sessionDropper interface {
	DelSession(net.Addr)
}

// multiplexer provides an interface to multiplex sockets onto QUIC sessions
type multiplexer interface {
	sync.Locker

	GetListener(netlocator) (*refcntListener, bool)
	AddListener(netlocator, *refcntListener)
	DelListener(netlocator)

	GetSession(netlocator) (*refcntSession, bool)
	AddSession(net.Addr, *refcntSession)
	sessionDropper

	RegisterPath(string, chan<- net.Conn) error
	UnregisterPath(string)
	Serve(quic.Session)
}

// dialMuxer is a subset of multiplexer, used for mangos.PipeDialer
type dialMuxer interface {
	sync.Locker

	GetSession(netlocator) (*refcntSession, bool)
	AddSession(net.Addr, *refcntSession)
	sessionDropper
}

type (
	listenNegotiator interface {
		ReadHeaders() (string, error)
		Abort(int, string) error
		Accept() error
	}

	dialNegotiator interface {
		WriteHeaders(string) error
		Ack() error
	}
)

type negotiator struct {
	io.ReadWriteCloser
}

func newNegotiator(pipe io.ReadWriteCloser) *negotiator {
	return &negotiator{ReadWriteCloser: pipe}
}

func (n negotiator) WriteHeaders(path string) (err error) {
	buf := bytes.NewBufferString(path + "\n")
	_, err = io.Copy(n, buf)
	return
}

func (n negotiator) Ack() (err error) {
	scanner := bufio.NewScanner(n)

	if !scanner.Scan() {
		err = io.EOF
	} else if err = scanner.Err(); err == nil {
		if data := scanner.Text(); data != "" {
			err = errors.New(data)
		}
	}

	return
}

func (n negotiator) ReadHeaders() (path string, err error) {
	scanner := bufio.NewScanner(n)

	if !scanner.Scan() {
		err = io.EOF
	} else if err = scanner.Err(); err == nil {
		path = scanner.Text()
	}

	return
}

func (n negotiator) Abort(status int, message string) error {
	buf := bytes.NewBufferString(fmt.Sprintf("%d:%s", status, message))
	_, _ = io.Copy(n, buf) // best-effort
	return n.Close()
}

func (n negotiator) Accept() (err error) {
	_, err = n.Write([]byte("\n"))
	return
}

type router struct {
	sync.RWMutex
	routes *radix.Tree
}

func newRouter() *router { return &router{routes: radix.New()} }

func (r *router) Get(path string) (ch chan<- net.Conn, ok bool) {
	r.RLock()
	defer r.RUnlock()

	var v interface{}
	if v, ok = r.routes.Get(path); ok {
		ch = v.(chan<- net.Conn)
	}

	return
}

func (r *router) Add(path string, ch chan<- net.Conn) (ok bool) {
	r.Lock()
	if _, ok = r.routes.Get(path); !ok {
		r.routes.Insert(path, ch)
	}
	r.Unlock()
	ok = !ok // turn "value not found" into "value successfully inserted"
	return
}

func (r *router) Del(path string) {
	r.Lock()
	r.routes.Delete(path)
	r.Unlock()
}

type refcntSession struct {
	gc     func()
	refcnt int32
	quic.Session
}

func newRefCntSession(sess quic.Session, d sessionDropper) *refcntSession {
	return &refcntSession{
		Session: sess,
		gc:      func() { d.DelSession(sess.RemoteAddr()) },
	}
}

func (r *refcntSession) Incr() *refcntSession {
	atomic.AddInt32(&r.refcnt, 1)
	return r
}

func (r *refcntSession) DecrAndClose() (err error) {
	if i := atomic.AddInt32(&r.refcnt, -1); i == 0 {
		err = r.Close(nil)
		r.gc()
	} else if i < 0 {
		panic(errors.New("already closed"))
	}
	return
}

package quic

import (
	"io"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

const (
	// OptionTLSConfig maps to a *tls.Config value
	OptionTLSConfig = "QUIC-TLS-CONFIG"
	// OptionQUICConfig maps to a *quic.Config value
	OptionQUICConfig = "QUIC-UDP-CONFIG"
)

type options struct {
	sync.RWMutex
	opt map[string]interface{}
}

// GetOption retrieves an option value.
func (o *options) get(name string) (interface{}, error) {
	o.RLock()
	defer o.RUnlock()

	if o.opt == nil {
		return nil, mangos.ErrBadOption
	}
	v, ok := o.opt[name]
	if !ok {
		return nil, mangos.ErrBadOption
	}
	return v, nil
}

// SetOption sets an option.  We have none, so just ErrBadOption.
func (o *options) set(name string, val interface{}) error {
	o.Lock()
	defer o.Unlock()
	return mangos.ErrBadOption
}

type quicTrans struct {
	opt *options
}

func (quicTrans) Scheme() string { return "quic" }

func (t quicTrans) NewDialer(addr string, sock mangos.Socket) (mangos.PipeDialer, error) {
	u, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}

	u.Path = filepath.Clean(u.Path)

	return &dialer{
		u:    u,
		opt:  t.opt,
		sock: sock,
	}, nil
}

func (t quicTrans) NewListener(addr string, sock mangos.Socket) (mangos.PipeListener, error) {
	u, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}

	u.Path = filepath.Clean(u.Path)

	return &listener{
		u:    u,
		ch:   make(chan io.ReadWriteCloser, 1),
		opt:  t.opt,
		sock: sock,
	}, nil
}

// NewTransport allocates a new quic:// transport.
func NewTransport() mangos.Transport {
	return &quicTrans{
		opt: &options{opt: make(map[string]interface{})},
	}
}
package dovecot

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
)

const DefaultPort int = 42001

type DictServer struct {
	be     Backend
	logger *zap.SugaredLogger
}

func NewDictServer(be Backend, logger *zap.SugaredLogger) (*DictServer, error) {
	h := &DictServer{
		be:     be,
		logger: logger,
	}
	return h, nil
}

func (h *DictServer) ListenAndServe(addr string) error {
	return h.ListenAndServeContext(context.Background(), addr)
}

func (h *DictServer) ListenAndServeContext(ctx context.Context, addr string) error {
	if addr == "" {
		addr = fmt.Sprintf(":%d", DefaultPort)
	}
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return h.Serve(l)
}

func (h *DictServer) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		c := h.newClient(conn)
		go c.handleClient()
	}
}

func (h *DictServer) newClient(conn net.Conn) *clientImpl {
	c := &clientImpl{
		h:      h,
		conn:   conn,
		logger: h.logger,
		be:     h.be,

		transactions: make(map[int]interface{}),
	}
	return c
}

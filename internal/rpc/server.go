package rpc

import (
	"clipboard/internal/enc"
	"errors"
	"net"
	"net/rpc"
)

type serverAddr struct {
	network string
	addr    string
}

func (a *serverAddr) Network() string {
	return "tcp"
}

func (a *serverAddr) String() string {
	return a.addr
}

type Server struct {
	addr      serverAddr
	key       []byte
	rpcServer *rpc.Server
	listener  net.Listener
}

func NewServer(addr string, key []byte) *Server {
	return &Server{
		addr: serverAddr{
			network: "tcp",
			addr:    addr,
		},
		key:       key,
		rpcServer: rpc.NewServer(),
	}
}

func (s *Server) Addr() net.Addr {
	return &s.addr
}

func (s *Server) Register(srv any) error {
	return s.rpcServer.Register(srv)
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen(s.addr.network, s.addr.addr)
	if err != nil {
		return err
	}
	s.listener = listener

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			continue
		}
		if s.key != nil {
			conn = enc.NewEncConn(conn, s.key)
		}
		go s.rpcServer.ServeConn(conn)
	}
}

func (s *Server) Close() error {
	return s.listener.Close()
}

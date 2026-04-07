package rpc

import (
	"clipboard/internal/enc"
	"errors"
	"net"
	"net/rpc"
)

type Server struct {
	addr      *net.TCPAddr
	key       []byte
	rpcServer *rpc.Server
	listener  *net.TCPListener
}

func NewServer(addr string, key []byte) (*Server, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		addr:      tcpAddr,
		key:       key,
		rpcServer: rpc.NewServer(),
	}, nil
}

func (s *Server) Addr() net.Addr {
	return s.addr
}

func (s *Server) Register(srv any) error {
	return s.rpcServer.Register(srv)
}

func (s *Server) ListenAndServe() error {
	listener, err := net.ListenTCP(s.addr.Network(), s.addr)
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

package rpc

import (
	"clipboard/internal/enc"
	"errors"
	"net"
	"net/rpc"
	"sync"
)

type Client struct {
	addr   *net.TCPAddr
	key    []byte
	client *rpc.Client
	lock   sync.Mutex
}

func NewClient(addr string, key []byte) (*Client, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Client{
		addr: tcpAddr,
		key:  key,
	}, nil
}

func NewClients(addrs []string, key []byte) ([]*Client, error) {
	var clients []*Client
	for _, v := range addrs {
		client, err := NewClient(v, key)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, nil
}

func (c *Client) Addr() net.Addr {
	return c.addr
}

func (c *Client) getClient() (*rpc.Client, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.client != nil {
		return c.client, nil
	}
	tcpConn, err := net.DialTCP(c.addr.Network(), nil, c.addr)
	if err != nil {
		return nil, err
	}
	conn := net.Conn(tcpConn)
	if c.key != nil {
		conn = enc.NewEncConn(conn, c.key)
	}
	c.client = rpc.NewClient(conn)
	return c.client, nil
}

func (c *Client) Call(serviceMethod string, args any, reply any) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}

	err = client.Call(serviceMethod, args, reply)
	if err != nil {
		_, ok := errors.AsType[*rpc.ServerError](err)
		if !ok {
			c.lock.Lock()
			if c.client == client {
				c.client = nil
			}
			c.lock.Unlock()
			_ = client.Close()
		}
	}
	return err
}

func (c *Client) Close() error {
	c.lock.Lock()
	client := c.client
	c.client = nil
	c.lock.Unlock()
	if client != nil {
		return client.Close()
	}
	return nil
}

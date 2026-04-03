package rpc

import (
	"clipboard/internal/enc"
	"errors"
	"net"
	"net/rpc"
	"sync"
)

type Client struct {
	addr   string
	key    []byte
	client *rpc.Client
	lock   sync.Mutex
}

func NewClient(addr string, key []byte) *Client {
	return &Client{
		addr: addr,
		key:  key,
	}
}

func (c *Client) getClient() (*rpc.Client, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.client != nil {
		return c.client, nil
	}
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return nil, err
	}
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

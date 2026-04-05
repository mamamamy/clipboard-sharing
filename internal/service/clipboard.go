package service

import (
	"bytes"
	"clipboard/internal/clipboard"
	"clipboard/internal/rpc"
	"sync"
)

const (
	RPCNameClipboardService = "RPCClipboardService"
)

type localClipboardService struct {
	info    *clipboard.Info
	notify  *sync.Cond
	lock    sync.Mutex
	version int64
}

var LocalClipboardService = func() *localClipboardService {
	c := &localClipboardService{
		info: &clipboard.Info{},
	}
	c.notify = sync.NewCond(&c.lock)
	return c
}()

func (c *localClipboardService) WaitLatestDigest() []byte {
	c.lock.Lock()
	defer c.lock.Unlock()
	version := c.version
	for version == c.version {
		c.notify.Wait()
	}
	digest := c.info.Digest
	return digest
}
func (c *localClipboardService) CompareAndGetInfo(digest []byte) *clipboard.Info {
	c.lock.Lock()
	defer c.lock.Unlock()
	if bytes.Equal(digest, c.info.Digest) {
		return nil
	}
	return c.info
}

func (c *localClipboardService) GetDigest() []byte {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.info.Digest
}

func (c *localClipboardService) GetInfo() *clipboard.Info {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.info
}

func (c *localClipboardService) SetInfo(info *clipboard.Info) {
	c.lock.Lock()
	c.info = info
	c.version++
	c.lock.Unlock()
	c.notify.Broadcast()
}

func (c *localClipboardService) PushInfo(info *clipboard.Info) {
	clipboard.Write(info)
}

type RPCClipboardService struct{}

func (c *RPCClipboardService) WaitLatestDigest(_ *struct{}, digest *[]byte) error {
	*digest = LocalClipboardService.WaitLatestDigest()
	return nil
}

func (c *RPCClipboardService) CompareAndGetInfo(digest *[]byte, info *clipboard.Info) error {
	r := LocalClipboardService.CompareAndGetInfo(*digest)
	if r != nil {
		*info = *r
	}
	return nil
}

func (c *RPCClipboardService) PushInfo(info *clipboard.Info, _ *struct{}) error {
	LocalClipboardService.PushInfo(info)
	return nil
}

type remoteClipboardService struct{}

var RemoteClipboardService = &remoteClipboardService{}

func (c *remoteClipboardService) WaitLatestDigest(client *rpc.Client) ([]byte, error) {
	var digest []byte
	err := client.Call(RPCNameClipboardService+".WaitLatestDigest", &struct{}{}, &digest)
	if err != nil {
		return nil, err
	}
	return digest, nil
}

func (c *remoteClipboardService) CompareAndGetInfo(client *rpc.Client, digest []byte) (*clipboard.Info, error) {
	var info clipboard.Info
	err := client.Call(RPCNameClipboardService+".CompareAndGetInfo", &digest, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *remoteClipboardService) PushInfo(client *rpc.Client, info *clipboard.Info) error {
	err := client.Call(RPCNameClipboardService+".PushInfo", info, nil)
	if err != nil {
		return err
	}
	return nil
}

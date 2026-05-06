package service

import (
	"clipboard/internal/clipboard"
	"clipboard/internal/rpc"
	"clipboard/internal/sync"
)

const (
	RPCName           = "RPC"
	RPCPushMeta       = RPCName + ".PushMeta"
	RPCPushData       = RPCName + ".PushData"
	RPCWaitLatestMeta = RPCName + ".WaitLatestMeta"
	RPCLatestMeta     = RPCName + ".LatestMeta"
	RPCPullData       = RPCName + ".PullData"
)

type Remote struct {
}

func (r *Remote) PushMeta(client *rpc.Client, meta *sync.Meta) (bool, error) {
	var ok bool
	err := client.Call(RPCPushMeta, meta, &ok)
	return ok, err
}

func (r *Remote) PushData(client *rpc.Client, data *clipboard.Data) {
	_ = client.Call(RPCPushData, data, &struct{}{})
}

func (r *Remote) WaitLatestMeta(client *rpc.Client) (*sync.Meta, error) {
	var meta sync.Meta
	err := client.Call(RPCWaitLatestMeta, struct{}{}, &meta)
	return &meta, err
}

func (r *Remote) LatestMeta(client *rpc.Client) (*sync.Meta, error) {
	var meta sync.Meta
	err := client.Call(RPCLatestMeta, struct{}{}, &meta)
	return &meta, err
}

func (r *Remote) PullData(client *rpc.Client, digest string) (*clipboard.Data, error) {
	var data clipboard.Data
	err := client.Call(RPCPullData, digest, &data)
	return &data, err
}

func NewRemote() *Remote {
	return &Remote{}
}

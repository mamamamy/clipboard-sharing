package service

import (
	"clipboard/internal/clipboard"
	"clipboard/internal/sync"
)

type RPC struct {
	local *Local
}

func NewRPC(local *Local) *RPC {
	return &RPC{
		local: local,
	}
}

func (r *RPC) PushMeta(meta *sync.Meta, ok *bool) error {
	*ok = r.local.RecvMeta(meta)
	return nil
}

func (r *RPC) PushData(data *clipboard.Data, _ *struct{}) error {
	r.local.RecvData(data)
	return nil
}

func (r *RPC) WaitLatestMeta(_ struct{}, meta *sync.Meta) error {
	*meta = *r.local.WaitLatestMeta()
	return nil
}

func (r *RPC) LatestMeta(_ struct{}, meta *sync.Meta) error {
	m := r.local.LatestMeta()
	if m == nil {
		return nil
	}
	*meta = *m
	return nil
}

func (r *RPC) PullData(digest string, data *clipboard.Data) error {
	*data = *r.local.PullData(digest)
	return nil
}

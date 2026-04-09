package service

import (
	"clipboard/internal/cache"
	"clipboard/internal/clipboard"
	"clipboard/internal/rpc"
	"clipboard/internal/sync"
	"context"
	"log"
	"time"
)

type Local struct {
	ctx       context.Context
	cancel    context.CancelFunc
	clients   []*rpc.Client
	sync      *sync.Manager
	cache     *cache.Cache
	readChan  chan *clipboard.Data
	writeChan chan *clipboard.Data
	ready     chan struct{}
	remote    *Remote
}

func (l *Local) Start(ctx context.Context) {
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.writeChan, l.readChan = clipboard.WatchWithWrite(l.ctx)
	l.StartWaitRemoteChange()
	close(l.ready)
	for {
		select {
		case data := <-l.readChan:
			l.LocalChange(data)
		case <-l.ctx.Done():
			return
		}
	}
}

func (l *Local) Stop() {
	l.ready = make(chan struct{})
	l.cancel()
}

func (l *Local) Ready() <-chan struct{} {
	return l.ready
}

func (l *Local) LocalChange(data *clipboard.Data) {
	log.Printf("[INFO] ✏️ local change %s", data.Digest)
	meta := l.sync.LocalChange(data.Digest)
	l.cache.Put(data)
	l.PushToRemote(data, meta)
}

func (l *Local) PushToRemote(data *clipboard.Data, meta *sync.Meta) {
	for _, client := range l.clients {
		go func(client *rpc.Client) {
			// 先推送元数据
			ok, err := l.remote.PushMeta(client, meta)
			if err != nil {
				return
			}
			if ok {
				// 对端需要时推送数据
				l.remote.PushData(client, data)
			}

		}(client)
	}
}

func (l *Local) RecvMeta(meta *sync.Meta) bool {
	if meta == nil {
		return false
	}
	// 尝试添加到元数据更新列
	ok := l.sync.TryAddRemoteMeta(meta)
	if !ok {
		return false
	}
	log.Printf("[INFO] 📡 receive meta %s", meta.Digest)
	data := l.cache.Get(meta.Digest)
	if data == nil {
		return true
	}
	// 缓存中存在，直接清理并写入剪贴板
	l.sync.TruncateUpdatesByDigest(meta.Digest)
	l.WriteToClipboard(data)
	l.PushToRemote(data, meta) // 主动推送
	return false
}

func (l *Local) RecvData(data *clipboard.Data) {
	if data == nil {
		return
	}
	meta := l.sync.TruncateUpdatesByDigest(data.Digest)
	if meta == nil {
		return
	}
	log.Printf("[INFO] 📥 receive data %s", meta.Digest)
	// data在更新列中存在，写入缓存，写入剪贴板，推送给所有peer
	l.cache.Put(data)
	l.WriteToClipboard(data)
	l.PushToRemote(data, meta) // 主动推送
}

func (l *Local) WriteToClipboard(data *clipboard.Data) {
	l.writeChan <- data
}

func (l *Local) WaitLatestMeta() *sync.Meta {
	return l.sync.WaitLatestMeta()
}

func (l *Local) PullData(digest string) *clipboard.Data {
	return l.cache.Get(digest)
}

func (l *Local) StartWaitRemoteChange() {
	for _, client := range l.clients {
		go func(client *rpc.Client) {
			sleep := func() {
				time.Sleep(1 * time.Second)
			}
			for {
				select {
				case <-l.ctx.Done():
					return
				default:
				}
				// 等待对端最新的元信息
				meta, err := l.remote.WaitLatestMeta(client)
				if err != nil {
					sleep()
					continue
				}
				// 同对端主动推送逻辑
				ok := l.RecvMeta(meta)
				if !ok {
					continue
				}
				// ok == true，需要拉取数据
				data, err := l.remote.PullData(client, meta.Digest)
				if err != nil {
					sleep()
					continue
				}
				// 同对端主动推送数据逻辑
				l.RecvData(data)
			}
		}(client)
	}
}

func NewLocal(peer []string, key []byte) (*Local, error) {
	l := &Local{}
	rpcClients, err := rpc.NewClients(peer, key)
	if err != nil {
		return nil, err
	}
	l.clients = rpcClients
	l.sync = sync.NewManager()
	l.cache = cache.NewCache(32)
	l.ready = make(chan struct{})
	l.remote = NewRemote()
	return l, nil
}

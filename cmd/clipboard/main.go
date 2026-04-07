package main

import (
	"clipboard/internal/clipboard"
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/argon2"
)

func NewKey(password string) []byte {
	return argon2.IDKey(
		[]byte(password),
		[]byte("clipboard-sharing"),
		3,
		64*1024,
		4,
		32,
	)
}

type Meta struct {
	Digest  string
	Version int64
}

type SyncManager struct {
	updates []*Meta
	version atomic.Int64
	lock    sync.Mutex
}

func (sm *SyncManager) AddLocalUpdate(digest string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	m := &Meta{
		Digest:  digest,
		Version: sm.version.Add(1),
	}
	sm.updates = append(sm.updates, m)
}

func (sm *SyncManager) AddRemoteUpdate(m *Meta) bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	localVersion := sm.version.Load()
	if localVersion >= m.Version {
		return false
	}
	if !sm.version.CompareAndSwap(localVersion, m.Version) {
		return false
	}
	sm.updates = append(sm.updates, m)
	return true
}

func (sm *SyncManager) GetLatestMeta() *Meta {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if len(sm.updates) == 0 {
		return nil
	}
	return sm.updates[len(sm.updates)-1]
}

func main() {
	//rpc.NewServer(":6666", NewKey("123456"))

	_, out := clipboard.WatchWithWrite(context.Background())

	for {
		for info := range out {
			fmt.Println(info)
		}
	}
}

package main

import (
	"clipboard/internal/clipboard"
	"context"
	"fmt"
	"sync"

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
	version int64
	lock    sync.RWMutex
}

func NewSyncManager() *SyncManager {
	return &SyncManager{}
}

func (sm *SyncManager) AddLocalUpdate(digest string) *Meta {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.version++
	m := &Meta{
		Digest:  digest,
		Version: sm.version,
	}
	sm.updates = append(sm.updates, m)
	return m
}

func (sm *SyncManager) AddRemoteUpdate(m *Meta) bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if sm.version >= m.Version {
		return false
	}
	sm.version = m.Version
	sm.updates = append(sm.updates, m)
	return true
}

func (sm *SyncManager) GetLatestMeta() *Meta {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	if len(sm.updates) == 0 {
		return nil
	}
	return sm.updates[len(sm.updates)-1]
}

func (sm *SyncManager) TruncateByDigest(digest string) bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for i := len(sm.updates) - 1; i >= 0; i-- {
		if sm.updates[i].Digest == digest {
			sm.updates = sm.updates[:i]
			return true
		}
	}
	return false
}

type LocalClipboardService struct {
	sm    *SyncManager
	cache *clipboard.Cache
}

func NewClipboardService() *LocalClipboardService {
	return &LocalClipboardService{
		sm:    NewSyncManager(),
		cache: clipboard.NewCache(32),
	}
}

func (c *LocalClipboardService) PushMeta(m *Meta) bool {
	return c.sm.AddRemoteUpdate(m)
}

func main() {
	sm := NewSyncManager()

	_, out := clipboard.WatchWithWrite(context.Background())

	for {
		for info := range out {
			meta := sm.AddLocalUpdate(info.Digest)
			fmt.Println(meta)
		}
	}
}

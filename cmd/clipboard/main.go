package main

import (
	"clipboard/internal/clipboard"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/rpc"
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
	latest  *Meta
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
	sm.latest = m
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
	sm.latest = m
	return true
}

func (sm *SyncManager) GetLatestMeta() *Meta {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	return sm.latest
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

type ClipboardService struct {
	sm      *SyncManager
	cache   *clipboard.Cache
	clients []*rpc.Client
}

func NewClipboardService(peer []string) *ClipboardService {
	return &ClipboardService{
		sm:    NewSyncManager(),
		cache: clipboard.NewCache(32),
	}
}

func (c *ClipboardService) PushMeta(m *Meta) bool {
	return c.sm.AddRemoteUpdate(m)
}

func main() {
}

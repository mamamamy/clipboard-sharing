package sync

import "sync"

type Meta struct {
	Digest  string
	Version int64
}

type Manager struct {
	latest  *Meta
	updates []*Meta
	version int64
	lock    sync.RWMutex
	notify  struct {
		*sync.Cond
		version int64
	}
}

func (m *Manager) LocalChange(digest string) *Meta {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.version++
	meta := &Meta{
		Digest:  digest,
		Version: m.version,
	}
	m.latest = meta
	m.notify.version++
	m.notify.Broadcast()
	return meta
}

func (m *Manager) TryAddRemoteMeta(meta *Meta) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.version >= meta.Version {
		return false
	}
	m.version = meta.Version
	m.updates = append(m.updates, meta)
	m.latest = meta
	m.notify.version++
	m.notify.Broadcast()
	return true
}

func (m *Manager) GetLatestMeta() *Meta {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.latest
}

func (m *Manager) TruncateUpdatesByDigest(digest string) *Meta {
	m.lock.Lock()
	defer m.lock.Unlock()
	for i := len(m.updates) - 1; i >= 0; i-- {
		if m.updates[i].Digest == digest {
			meta := m.updates[i]
			m.updates = m.updates[:i]
			return meta
		}
	}
	return nil
}

func (m *Manager) WaitLatestMeta() *Meta {
	m.notify.L.Lock()
	defer m.notify.L.Unlock()
	version := m.notify.version
	for m.notify.version == version {
		m.notify.Wait()
	}
	return m.latest
}

func (m *Manager) LatestMeta() *Meta {
	m.notify.L.Lock()
	defer m.notify.L.Unlock()
	return m.latest
}

func NewManager() *Manager {
	m := &Manager{}
	m.notify.Cond = sync.NewCond(m.lock.RLocker())
	return m
}

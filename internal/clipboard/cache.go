package clipboard

import "sync"

type Cache struct {
	data []*Info
	lock sync.Mutex
	size int
}

func NewCache(size int) *Cache {
	return &Cache{
		size: size,
	}
}

func (c *Cache) Get(digest string) *Info {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i := len(c.data) - 1; i >= 0; i-- {
		info := c.data[i]
		if c.data[i].Digest == digest {
			c.data = append(c.data[:i], c.data[i+1:]...)
			c.data = append(c.data, info)
			return info
		}
	}

	return nil
}

func (c *Cache) Put(info *Info) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i := len(c.data) - 1; i >= 0; i-- {
		if c.data[i].Digest == info.Digest {
			c.data = append(c.data[:i], c.data[i+1:]...)
			c.data = append(c.data, info)
			return
		}
	}

	if len(c.data) >= c.size {
		c.data = c.data[1:]
	}

	c.data = append(c.data, info)
}

func (c *Cache) Latest() *Info {
	c.lock.Lock()
	defer c.lock.Unlock()

	if len(c.data) > 0 {
		return c.data[len(c.data)-1]
	}

	return nil
}

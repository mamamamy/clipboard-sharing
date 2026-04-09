package cache

import (
	"clipboard/internal/clipboard"
	"sync"
)

type Cache struct {
	data []*clipboard.Data
	lock sync.Mutex
	size int
}

func (c *Cache) Get(digest string) *clipboard.Data {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i := len(c.data) - 1; i >= 0; i-- {
		data := c.data[i]
		if c.data[i].Digest == digest {
			c.data = append(c.data[:i], c.data[i+1:]...)
			c.data = append(c.data, data)
			return data
		}
	}

	return nil
}

func (c *Cache) Put(data *clipboard.Data) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i := len(c.data) - 1; i >= 0; i-- {
		if c.data[i].Digest == data.Digest {
			c.data = append(c.data[:i], c.data[i+1:]...)
			c.data = append(c.data, data)
			return
		}
	}

	if len(c.data) >= c.size {
		c.data = c.data[1:]
	}

	c.data = append(c.data, data)
}

func NewCache(size int) *Cache {
	return &Cache{
		size: size,
	}
}

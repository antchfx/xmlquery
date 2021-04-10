package xmlquery

import (
	"bufio"
)

type CachedReader struct {
	buffer *bufio.Reader
	cache []byte
	cacheCap int
	cacheLen int
	caching bool
}

func NewCachedReader(r *bufio.Reader) *CachedReader {
	return &CachedReader{
		buffer:   r,
		cache:    make([]byte, 4096),
		cacheCap: 4096,
		cacheLen: 0,
		caching:  false,
	}
}

func (c *CachedReader) StartCaching() {
	c.cacheLen = 0
	c.caching = true
}

func (c *CachedReader) ReadByte() (byte, error) {
	if !c.caching {
		return c.buffer.ReadByte()
	}
	b, err := c.buffer.ReadByte()
	if err != nil {
		return b, err
	}
	if c.cacheLen < c.cacheCap {
		c.cache[c.cacheLen] = b
		c.cacheLen++
	}
	return b, err
}

func (c *CachedReader) Cache() []byte {
	return c.cache[:c.cacheLen]
}

func (c *CachedReader) StopCaching() {
	c.caching = false
}

func (c *CachedReader) Read(p []byte) (int, error) {
	n, err := c.buffer.Read(p)
	if err != nil {
		return n, err
	}
	if c.caching && c.cacheLen < c.cacheCap {
		for i := 0; i < n; i++ {
			c.cache[c.cacheLen] = p[i]
			c.cacheLen++
			if c.cacheLen >= c.cacheCap {
				break
			}
		}
	}
	return n, err
}


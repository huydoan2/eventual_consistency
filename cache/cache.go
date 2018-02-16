package cache

import (
	"github.com/huydoan2/eventual_consistency/vectorclock"
)

// key-value store cache
type Value struct {
	Val   string
	Clock vectorclock.VectorClock
}

type Payload struct {
	Key     string
	Val     string
	ValTime vectorclock.VectorClock
	Clock   vectorclock.VectorClock // current clock of the process
}

// Cache class
type Cache struct {
	data map[string]Value
}

func (c *Cache) Invalidate() {
	c.data = make(map[string]Value)
}

func (c *Cache) Insert(p *Payload) {
	c.data[p.Key] = Value{p.Val, p.ValTime}
}

func (c *Cache) Find(key *string) (Value, bool) {
	v, ok := c.data[*key]
	return v, ok
}

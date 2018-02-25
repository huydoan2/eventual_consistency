package cache

import (
	"../vectorclock"
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
	Data map[string]Value
}

// New initialize a new cache
func New() *Cache {
	c := new(Cache)
	c.Data = make(map[string]Value)
	return c
}

func (c *Cache) Invalidate() {
	c.Data = make(map[string]Value)
}

func (c *Cache) Insert(p *Payload) {

	c.Data[p.Key] = Value{p.Val, p.ValTime}
}

func (c *Cache) Find(key *string) (Value, bool) {
	v, ok := c.Data[*key]
	return v, ok
}

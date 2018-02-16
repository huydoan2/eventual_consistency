package cache

import "../vectorclock"

// key-value store cache
type Value struct {
	val   string
	clock vectorclock.VectorClock
}

type Payload struct {
	key   string
	val   Value                   // value and the clock associated with it
	clock vectorclock.VectorClock // current clock of the process
}

// Cache class
type Cache struct {
	data map[string]Value
}

func (c *Cache) Invalidate() {

}

func (c *Cache) Insert(p *Payload) {
	c.data[p.key] = Value{p.val, p.clock}
}

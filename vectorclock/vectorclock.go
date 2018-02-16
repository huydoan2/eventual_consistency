package vectorclock

import (
	"fmt"
	"math"
)

const MAXPROC = 10

type cmp int

const (
	LESS cmp = iota
	GREATER
	CONCURENT
)

type TimeStamp struct {
	time [MAXPROC]int64
}

func (t *TimeStamp) Compare(other *TimeStamp) cmp {
	var count = 0
	out := LESS
	for k, v := range t.time {
		if out == GREATER && other.time[k] < v {
			count++
			out = LESS
		} else if out == LESS && other.time[k] > v {
			count++
			out = GREATER
		}

		if count == 2 {
			out = CONCURENT
			break
		}
	}
	return out
}

func (t *TimeStamp) Increment(id int64) {
	if id > 9 {
		panic(fmt.Sprintf("VectorClock: id %d is out of range", id))
	}
	t.time[id]++
}

func (t *TimeStamp) Update(other *TimeStamp) {
	for k, _ := range t.time {
		t.time[k] = int64(math.Max(float64(t.time[k]), float64(other.time[k])))
	}
}

type VectorClock struct {
	Time TimeStamp
	Id   int64
}

func (t *VectorClock) Compare(other *VectorClock) cmp {
	out := t.Time.Compare(&other.Time)
	if out == CONCURENT {
		if t.Id > other.Id {
			return GREATER
		}
		return LESS
	}
	return out
}

func (t *VectorClock) Update(other *VectorClock) {
	t.Time.Update(&other.Time)
}

func (t *VectorClock) Increment(id int64) {
	if id > 9 {
		panic(fmt.Sprintf("VectorClock: id %d is out of range", id))
	}
	t.Time.Increment(id)
}
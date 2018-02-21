package vectorclock

import (
	"bytes"
	"fmt"
	"math"
)

const MAXPROC = 10

type Cmp int

const (
	LESS Cmp = iota
	GREATER
	CONCURENT
)

type TimeStamp struct {
	Time [MAXPROC]int64
}

func (t *TimeStamp) Compare(other *TimeStamp) Cmp {
	var count = 0
	out := LESS
	for k, v := range t.Time {
		if out == GREATER && other.Time[k] > v {
			count++
			out = LESS
		} else if out == LESS && other.Time[k] < v {
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
	t.Time[id]++
}

func (t *TimeStamp) Update(other *TimeStamp) {
	for k, _ := range t.Time {
		t.Time[k] = int64(math.Max(float64(t.Time[k]), float64(other.Time[k])))
	}
}

type VectorClock struct {
	Time TimeStamp
	Id   int64
}

func (t *VectorClock) Compare(other *VectorClock) Cmp {
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

func (t *VectorClock) ToString() string {
	var buffer bytes.Buffer
	buffer.WriteString("<<")

	i := 0
	for ; i < MAXPROC-1; i++ {
		buffer.WriteString(fmt.Sprintf("%d, ", t.Time.Time[i]))
	}

	buffer.WriteString(fmt.Sprintf("%d>, ", t.Time.Time[i]))
	buffer.WriteString(fmt.Sprintf("%d>", t.Id))

	return buffer.String()
}

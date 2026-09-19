package main

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

// lru is a fixed-size in-memory cache. When full, it drops the entry that
// was used longest ago. Recency is tracked with a linked list: every get or
// set moves the entry to the front, so the back is always the oldest.
// Entries also expire after ttl, so a changed record is never served stale
// for longer than that.
type lru struct {
	mu        sync.Mutex
	max       int
	ttl       time.Duration
	items     map[string]*list.Element
	order     *list.List // front = most recently used
	evictions atomic.Int64
	expired   atomic.Int64
}

type entry struct {
	key, val string
	exp      time.Time
}

func newLRU(max int, ttl time.Duration) *lru {
	return &lru{max: max, ttl: ttl, items: map[string]*list.Element{}, order: list.New()}
}

func (c *lru) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return "", false
	}
	e := el.Value.(*entry)
	if time.Now().After(e.exp) {
		c.order.Remove(el)
		delete(c.items, key)
		c.expired.Add(1)
		return "", false
	}
	c.order.MoveToFront(el)
	return e.val, true
}

func (c *lru) set(key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp := time.Now().Add(c.ttl)
	if el, ok := c.items[key]; ok {
		el.Value.(*entry).val = val
		el.Value.(*entry).exp = exp
		c.order.MoveToFront(el)
		return
	}
	c.items[key] = c.order.PushFront(&entry{key, val, exp})
	if c.order.Len() > c.max {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*entry).key)
		c.evictions.Add(1)
	}
}

func (c *lru) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// flight collapses concurrent lookups of the same key into one call: the
// first caller runs fn, everyone else arriving meanwhile waits for its result.
// This is what stops a cache miss on a hot key from stampeding the database.
type flight struct {
	mu    sync.Mutex
	calls map[string]*call
}

type call struct {
	wg  sync.WaitGroup
	val string
}

func newFlight() *flight { return &flight{calls: map[string]*call{}} }

// do returns fn's result and whether this caller was the one that ran it.
func (f *flight) do(key string, fn func() string) (val string, leader bool) {
	f.mu.Lock()
	if c, ok := f.calls[key]; ok {
		f.mu.Unlock()
		c.wg.Wait()
		return c.val, false
	}
	c := &call{}
	c.wg.Add(1)
	f.calls[key] = c
	f.mu.Unlock()

	c.val = fn()
	c.wg.Done()

	f.mu.Lock()
	delete(f.calls, key)
	f.mu.Unlock()
	return c.val, true
}

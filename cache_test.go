package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLRUEvictsOldest(t *testing.T) {
	c := newLRU(2, time.Minute)
	c.set("a", "1")
	c.set("b", "2")
	c.get("a")      // a is now most recent, b is oldest
	c.set("c", "3") // over capacity: b must go
	if _, ok := c.get("b"); ok {
		t.Fatal("b should have been evicted")
	}
	if v, ok := c.get("a"); !ok || v != "1" {
		t.Fatal("a should survive, it was used recently")
	}
	if c.len() != 2 || c.evictions.Load() != 1 {
		t.Fatalf("len=%d evictions=%d", c.len(), c.evictions.Load())
	}
}

func TestLRUExpires(t *testing.T) {
	c := newLRU(2, 10*time.Millisecond)
	c.set("a", "1")
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.get("a"); ok {
		t.Fatal("a should have expired")
	}
	if c.expired.Load() != 1 || c.len() != 0 {
		t.Fatalf("expired=%d len=%d", c.expired.Load(), c.len())
	}
}

func TestFlightRunsOnce(t *testing.T) {
	f := newFlight()
	var calls, leaders atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, leader := f.do("k", func() string {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return "v"
			})
			if v != "v" {
				t.Errorf("got %q", v)
			}
			if leader {
				leaders.Add(1)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 || leaders.Load() != 1 {
		t.Fatalf("calls=%d leaders=%d, want 1 and 1", calls.Load(), leaders.Load())
	}
}

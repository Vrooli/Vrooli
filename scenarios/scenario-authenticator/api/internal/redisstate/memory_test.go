package redisstate

import (
	"context"
	"testing"
	"time"
)

func TestMemorySetGetDelExists(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	if err := m.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	v, ok, _ := m.Get(ctx, "k")
	if !ok || v != "v" {
		t.Fatalf("get = %q,%v", v, ok)
	}
	if ex, _ := m.Exists(ctx, "k"); !ex {
		t.Fatal("exists false")
	}
	_ = m.Del(ctx, "k")
	if _, ok, _ := m.Get(ctx, "k"); ok {
		t.Fatal("key not deleted")
	}
}

func TestMemoryTTLExpiry(t *testing.T) {
	now := time.Unix(1000, 0)
	m := NewMemoryWithClock(func() time.Time { return now })
	ctx := context.Background()
	_ = m.Set(ctx, "k", "v", time.Minute)
	if _, ok, _ := m.Get(ctx, "k"); !ok {
		t.Fatal("should be present before expiry")
	}
	now = now.Add(2 * time.Minute)
	if _, ok, _ := m.Get(ctx, "k"); ok {
		t.Fatal("should be expired")
	}
	if ex, _ := m.Exists(ctx, "k"); ex {
		t.Fatal("expired key still exists")
	}
}

func TestMemorySets(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	_ = m.SAdd(ctx, "s", "a", "b", "c")
	members, _ := m.SMembers(ctx, "s")
	if len(members) != 3 {
		t.Fatalf("members = %v", members)
	}
	_ = m.SRem(ctx, "s", "b")
	members, _ = m.SMembers(ctx, "s")
	if len(members) != 2 {
		t.Fatalf("after srem = %v", members)
	}
}

func TestMemoryIncr(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	for i := int64(1); i <= 3; i++ {
		n, err := m.Incr(ctx, "c")
		if err != nil || n != i {
			t.Fatalf("incr %d = %d,%v", i, n, err)
		}
	}
}

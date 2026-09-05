package redisstate

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func TestNamespacedStoreIsolatesVariants(t *testing.T) {
	ctx := context.Background()
	base := NewMemory()
	liveNS, err := storage.ResolveNamespace(storage.NamespaceConfig{Root: "scenario-authenticator"})
	if err != nil {
		t.Fatalf("resolve live namespace: %v", err)
	}
	shadowNS, err := storage.ResolveNamespace(storage.NamespaceConfig{Root: "scenario-authenticator_shadow", Variant: "shadow"})
	if err != nil {
		t.Fatalf("resolve shadow namespace: %v", err)
	}
	live, err := NewNamespacedStore(base, liveNS, "auth")
	if err != nil {
		t.Fatalf("create live store: %v", err)
	}
	shadow, err := NewNamespacedStore(base, shadowNS, "auth")
	if err != nil {
		t.Fatalf("create shadow store: %v", err)
	}

	if err := live.Set(ctx, "session:same", "live", time.Minute); err != nil {
		t.Fatalf("write live key: %v", err)
	}
	if err := shadow.Set(ctx, "session:same", "shadow", time.Minute); err != nil {
		t.Fatalf("write shadow key: %v", err)
	}
	liveValue, liveFound, err := live.Get(ctx, "session:same")
	if err != nil || !liveFound || liveValue != "live" {
		t.Fatalf("live key = %q, found=%v, err=%v", liveValue, liveFound, err)
	}
	shadowValue, shadowFound, err := shadow.Get(ctx, "session:same")
	if err != nil || !shadowFound || shadowValue != "shadow" {
		t.Fatalf("shadow key = %q, found=%v, err=%v", shadowValue, shadowFound, err)
	}

	if err := live.SAdd(ctx, "usersessions:user", "session:same"); err != nil {
		t.Fatalf("write live set: %v", err)
	}
	members, err := shadow.SMembers(ctx, "usersessions:user")
	if err != nil {
		t.Fatalf("read shadow set: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("shadow set leaked live members: %v", members)
	}
}

func TestNewNamespacedStoreRejectsNilStore(t *testing.T) {
	ns, err := storage.ResolveNamespace(storage.NamespaceConfig{Root: "scenario-authenticator"})
	if err != nil {
		t.Fatalf("resolve namespace: %v", err)
	}
	if _, err := NewNamespacedStore(nil, ns, "auth"); err == nil {
		t.Fatal("expected nil store to be rejected")
	}
}

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

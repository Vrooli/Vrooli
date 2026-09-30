package models

import (
	"context"
	"testing"
)

func TestOpDefaultStoreSetGetClear(t *testing.T) {
	ctx := context.Background()
	st := NewOpDefaultStore(newStateDB(t))

	if got, err := st.Get(ctx, "upscale"); err != nil || got != "" {
		t.Fatalf("fresh default = %q, %v; want empty nil", got, err)
	}
	if err := st.Set(ctx, "upscale", "real-esrgan"); err != nil {
		t.Fatalf("set default: %v", err)
	}
	if got, err := st.Get(ctx, "upscale"); err != nil || got != "real-esrgan" {
		t.Fatalf("default = %q, %v; want real-esrgan nil", got, err)
	}

	if err := st.Set(ctx, "upscale", "realesr-general-x4v3"); err != nil {
		t.Fatalf("upsert default: %v", err)
	}
	all, err := st.Load(ctx)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if len(all) != 1 || all["upscale"] != "realesr-general-x4v3" {
		t.Fatalf("defaults after upsert = %+v", all)
	}

	if err := st.Set(ctx, "upscale", ""); err != nil {
		t.Fatalf("empty id should clear: %v", err)
	}
	if got, err := st.Get(ctx, "upscale"); err != nil || got != "" {
		t.Fatalf("default after clear = %q, %v; want empty nil", got, err)
	}
}

func TestOpDefaultStoreRejectsEmptyOperation(t *testing.T) {
	st := NewOpDefaultStore(newStateDB(t))
	if err := st.Set(context.Background(), "", "model"); err == nil {
		t.Fatal("expected empty operation to be rejected")
	}
}

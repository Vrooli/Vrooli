package aisearch

import (
	"testing"
)

func TestPayloadHash_StableAcrossNoOpRebuilds(t *testing.T) {
	r := CommandRecord{
		Origin: "demo", Group: "x", Name: "y", FullPath: "demo x y",
		Description: "list things", Flags: []string{"json"}, Tags: []string{"effect:read"},
		Binding: "Svc.M", Source: SourceManifest,
	}
	t1 := composeCommandEmbeddingText(r)
	p1 := buildCommandPayload(r, t1)
	t2 := composeCommandEmbeddingText(r)
	p2 := buildCommandPayload(r, t2)
	h1, _ := p1[payloadHashKey].(string)
	h2, _ := p2[payloadHashKey].(string)
	if h1 == "" {
		t.Fatalf("payload hash missing")
	}
	if h1 != h2 {
		t.Errorf("hash changed across no-op rebuild: %q vs %q", h1, h2)
	}
}

func TestPayloadHash_ChangesOnFieldEdit(t *testing.T) {
	a := CommandRecord{Origin: "demo", Name: "x", FullPath: "demo x", Source: SourceManifest}
	b := a
	b.Description = "now with a description"
	ha, _ := buildCommandPayload(a, composeCommandEmbeddingText(a))[payloadHashKey].(string)
	hb, _ := buildCommandPayload(b, composeCommandEmbeddingText(b))[payloadHashKey].(string)
	if ha == hb {
		t.Errorf("hash unchanged after field edit (%q)", ha)
	}
}

func TestPayloadToHit_FullProjection(t *testing.T) {
	r := CommandRecord{
		Origin: "demo", Group: "x", Name: "y", FullPath: "demo x y",
		Description: "d", Flags: []string{"json"}, Tags: []string{"effect:read"},
		Binding: "Svc.M", Source: SourceManifest,
	}
	p := buildCommandPayload(r, composeCommandEmbeddingText(r))
	hit := payloadToHit("pt-1", 0.83, p)
	if hit.ID != "pt-1" || hit.Origin != "demo" || hit.FullPath != "demo x y" {
		t.Errorf("bad identity: %+v", hit)
	}
	if hit.Score != 0.83 || hit.ScorePercent != 83 {
		t.Errorf("score = %v / %d", hit.Score, hit.ScorePercent)
	}
	if hit.Binding != "Svc.M" || hit.Source != SourceManifest {
		t.Errorf("missing binding/source: %+v", hit)
	}
}

func TestPointIDForCommand_StableAndUnique(t *testing.T) {
	a := PointIDForCommand("demo x list")
	b := PointIDForCommand("demo x list")
	c := PointIDForCommand("demo x show")
	if a != b {
		t.Errorf("not stable: %q vs %q", a, b)
	}
	if a == c {
		t.Errorf("collision: %q == %q", a, c)
	}
	if len(a) != 36 {
		t.Errorf("not UUID-shaped (len=%d): %q", len(a), a)
	}
}

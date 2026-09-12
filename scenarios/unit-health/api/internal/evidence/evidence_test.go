package evidence

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewKeyCanonicalizesAllDimensions(t *testing.T) {
	a, err := NewKey(KeyInput{SourceDigest: "s", Environment: map[string]string{"B": "2", "A": "1"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewKey(KeyInput{SourceDigest: "s", Environment: map[string]string{"A": "1", "B": "2"}})
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest != b.Digest || string(a.Canonical) != string(b.Canonical) {
		t.Fatalf("keys differ: %s/%s", a.Digest, b.Digest)
	}
}

func TestNewKeySeparatesToolchains(t *testing.T) {
	a, err := NewKey(KeyInput{SourceDigest: "same", ToolchainIdentity: "go@1.23"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewKey(KeyInput{SourceDigest: "same", ToolchainIdentity: "go@1.24"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest == b.Digest {
		t.Fatal("toolchain change reused the same evidence key")
	}
}

func TestNewKeySeparatesExecutionAndPolicyDimensions(t *testing.T) {
	base := KeyInput{
		SourceDigest: "source", ConfigDigest: "config", DependencyLockDigest: "lock",
		ToolchainIdentity: "toolchain", AdapterID: "adapter", AdapterVersion: "1.0.0",
		PolicyDigest: "policy-a", RunnerProfile: "bounded", OS: "linux", Architecture: "amd64",
		CoverageMode: "coverage", ArtifactSchema: "unit-health.response.v1",
	}
	variants := []KeyInput{base, base}
	variants[1].PolicyDigest = "policy-b"
	a, err := NewKey(variants[0])
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewKey(variants[1])
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest == b.Digest {
		t.Fatal("policy change reused the same evidence key")
	}
}

func TestNewKeySeparatesEveryExecutionInputDimension(t *testing.T) {
	base := KeyInput{
		SourceDigest: "source", ConfigDigest: "config", DependencyLockDigest: "lock",
		ToolchainIdentity: "toolchain", AdapterID: "adapter", AdapterVersion: "1.0.0",
		PolicyDigest: "policy", RunnerProfile: "profile", OS: "linux", Architecture: "amd64",
		Environment: map[string]string{"target_kind": "scenario"}, CoverageMode: "coverage", ArtifactSchema: "schema",
	}
	cases := []struct {
		name   string
		mutate func(*KeyInput)
	}{
		{"source", func(input *KeyInput) { input.SourceDigest = "changed" }},
		{"config", func(input *KeyInput) { input.ConfigDigest = "changed" }},
		{"lock", func(input *KeyInput) { input.DependencyLockDigest = "changed" }},
		{"toolchain", func(input *KeyInput) { input.ToolchainIdentity = "changed" }},
		{"adapter", func(input *KeyInput) { input.AdapterID = "changed" }},
		{"adapter-version", func(input *KeyInput) { input.AdapterVersion = "changed" }},
		{"policy", func(input *KeyInput) { input.PolicyDigest = "changed" }},
		{"profile", func(input *KeyInput) { input.RunnerProfile = "changed" }},
		{"platform", func(input *KeyInput) { input.OS = "darwin" }},
		{"architecture", func(input *KeyInput) { input.Architecture = "arm64" }},
		{"environment", func(input *KeyInput) { input.Environment = map[string]string{"target_kind": "package"} }},
		{"coverage", func(input *KeyInput) { input.CoverageMode = "test" }},
		{"artifact-schema", func(input *KeyInput) { input.ArtifactSchema = "changed" }},
	}
	baseKey, err := NewKey(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			variant := base
			tc.mutate(&variant)
			variantKey, err := NewKey(variant)
			if err != nil {
				t.Fatal(err)
			}
			if variantKey.Digest == baseKey.Digest {
				t.Fatalf("%s dimension did not change evidence key", tc.name)
			}
		})
	}
}

func TestStoreRejectsCorruptionAndEvictsByBudget(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, 250, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := NewKey(KeyInput{SourceDigest: "one"})
	now := time.Unix(100, 0)
	if err := store.Put(key, []byte("payload"), now); err != nil {
		t.Fatal(err)
	}
	record, err := store.Get(key, now.Add(time.Minute))
	if err != nil || string(record.Payload) != "payload" {
		t.Fatalf("get=%+v err=%v", record, err)
	}
	if err := os.WriteFile(store.path(key.Digest), []byte(`{"complete":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(key, now.Add(2*time.Minute)); !errors.Is(err, ErrMiss) {
		t.Fatalf("corrupt get err=%v", err)
	}
	matches, err := filepath.Glob(store.path(key.Digest) + ".corrupt-*")
	if err != nil || len(matches) != 1 {
		t.Fatalf("corrupt record was not quarantined: matches=%v err=%v", matches, err)
	}
}

func TestStoreRejectsStaleAndIncompleteRecords(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(100, 0)
	stale, _ := NewKey(KeyInput{SourceDigest: "stale"})
	if err := store.Put(stale, []byte("old"), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(stale, now); !errors.Is(err, ErrMiss) || !errors.Is(err, ErrStale) {
		t.Fatalf("stale record error=%v; want stale cache miss", err)
	}
	incomplete, _ := NewKey(KeyInput{SourceDigest: "incomplete"})
	if err := os.WriteFile(store.path(incomplete.Digest), []byte(`{"key_digest":"`+incomplete.Digest+`","complete":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(incomplete, now); !errors.Is(err, ErrMiss) || !errors.Is(err, ErrCorrupt) {
		t.Fatalf("incomplete record error=%v; want corrupt cache miss", err)
	}
}

func TestStoreEvictsOldestCompleteRecordsWithinByteBudget(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, 500, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := NewKey(KeyInput{SourceDigest: "first"})
	second, _ := NewKey(KeyInput{SourceDigest: "second"})
	now := time.Unix(100, 0)
	if err := store.Put(first, []byte(strings.Repeat("a", 180)), now); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(second, []byte(strings.Repeat("b", 180)), now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(first, now.Add(time.Minute)); !errors.Is(err, ErrMiss) {
		t.Fatalf("oldest record survived byte-budget eviction: %v", err)
	}
	if _, err := store.Get(second, now.Add(time.Minute)); err != nil {
		t.Fatalf("newest record was evicted unexpectedly: %v", err)
	}
}

func TestStoreEvictsSingleRecordThatExceedsByteBudget(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, 100, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := NewKey(KeyInput{SourceDigest: "oversized"})
	now := time.Unix(100, 0)
	if err := store.Put(key, []byte(strings.Repeat("x", 1<<10)), now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(key, now.Add(time.Minute)); !errors.Is(err, ErrMiss) {
		t.Fatalf("oversized record remained reusable after byte-budget eviction: %v", err)
	}
}

func TestStoreIgnoresInterruptedTemporaryWrites(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := NewKey(KeyInput{SourceDigest: "interrupted"})
	if err := os.WriteFile(filepath.Join(root, ".evidence-interrupted.tmp"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(key, time.Unix(100, 0)); !errors.Is(err, ErrMiss) {
		t.Fatalf("interrupted temporary write affected cache lookup: %v", err)
	}
	if err := store.Put(key, []byte("complete"), time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(key, time.Unix(101, 0)); err != nil {
		t.Fatalf("complete record could not be read with interrupted temp present: %v", err)
	}
}

func TestStoreSurvivesReconstructionAfterRestart(t *testing.T) {
	root := t.TempDir()
	key, _ := NewKey(KeyInput{SourceDigest: "restart"})
	now := time.Unix(100, 0)
	first, err := NewStore(root, 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Put(key, []byte("durable"), now); err != nil {
		t.Fatal(err)
	}

	second, err := NewStore(root, 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	record, err := second.Get(key, now.Add(time.Minute))
	if err != nil || string(record.Payload) != "durable" {
		t.Fatalf("reconstructed store get=%+v err=%v", record, err)
	}
}

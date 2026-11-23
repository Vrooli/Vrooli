package modelretention

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCanonicalNameFoldsTagAndCase(t *testing.T) {
	cases := map[string]string{
		"nomic-embed-text":                      "nomic-embed-text:latest",
		"Qwen3.5:4B":                            "qwen3.5:4b",
		"huihui_ai/granite4.1-abliterated:8b":   "huihui_ai/granite4.1-abliterated:8b",
		"laya@sha256:abc":                       "laya:latest",
		"hf.co/org/model":                       "hf.co/org/model:latest",
		"registry.ollama.ai/library/Qwen3.5:9B": "qwen3.5:9b",
		"library/qwen3.5:9b":                    "qwen3.5:9b",
		"registry.ollama.ai/huihui_ai/x:8b":     "huihui_ai/x:8b",
		"  ":                                    "",
	}
	for in, want := range cases {
		if got := CanonicalName(in); got != want {
			t.Errorf("CanonicalName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFenceRefusesEvictionWhileARequestIsInFlightAndRejectsRequestsDuringEviction(t *testing.T) {
	fence := Fence{Dir: t.TempDir()}
	releaseRequest, err := fence.EnterRequest("llama3.2:1b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fence.AcquireEviction("llama3.2:1b"); !errors.Is(err, ErrModelInUse) {
		t.Fatalf("eviction during a request: got %v, want ErrModelInUse", err)
	}
	// A request for another model does not block this eviction.
	other, err := fence.EnterRequest("qwen3:4b")
	if err != nil {
		t.Fatal(err)
	}
	releaseRequest()
	releaseEviction, err := fence.AcquireEviction("LLAMA3.2:1b")
	if err != nil {
		t.Fatalf("eviction after the request ended: %v", err)
	}
	if _, err := fence.EnterRequest("llama3.2:1b"); !errors.Is(err, ErrModelFenced) {
		t.Fatalf("request during eviction: got %v, want ErrModelFenced", err)
	}
	releaseEviction()
	again, err := fence.EnterRequest("llama3.2:1b")
	if err != nil {
		t.Fatalf("request after eviction released: %v", err)
	}
	again()
	other()
}

func TestFenceTreatsAnUnidentifiedRequestAsInFlightForEveryModel(t *testing.T) {
	fence := Fence{Dir: t.TempDir()}
	release, err := fence.EnterRequest("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fence.AcquireEviction("qwen3:4b"); !errors.Is(err, ErrModelInUse) {
		t.Fatalf("got %v, want ErrModelInUse", err)
	}
	release()
	evict, err := fence.AcquireEviction("qwen3:4b")
	if err != nil {
		t.Fatal(err)
	}
	evict()
}

func TestRecorderKeepsTheLatestRequestAndStartsTracking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "model-usage.json")
	clock := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	rec := NewRecorder("ollama", path, func() time.Time { return clock })
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	doc, err := ReadUsage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.TrackingSince.Equal(clock) || len(doc.Models) != 0 {
		t.Fatalf("first flush should start tracking with no models: %+v", doc)
	}
	clock = clock.Add(time.Hour)
	rec.Record("nomic-embed-text")
	// A second recorder with an older observation must not move it backwards.
	older := NewRecorder("ollama", path, func() time.Time { return clock.Add(-30 * time.Minute) })
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	older.Record("nomic-embed-text:latest")
	if err := older.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	doc, err = ReadUsage(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := doc.Models["nomic-embed-text:latest"]
	if !entry.LastRequestAt.Equal(clock) || entry.Requests != 2 {
		t.Fatalf("entry = %+v, want last %s and 2 requests", entry, clock)
	}
	if got := doc.LastActivity("qwen3:4b"); !got.Equal(doc.TrackingSince) {
		t.Fatalf("an unrequested model's last activity is tracking start, got %s", got)
	}
}

func TestRecorderSplitsRequestsByCallerAcrossFlushes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model-usage.json")
	clock := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	rec := NewRecorder("ollama", path, func() time.Time { return clock })
	rec.Record("qwen3.5:9b")
	rec.RecordCaller("qwen3.5:9b", "scenario:ai-gateway-start-api/ai-gateway-api", clock)
	rec.Record("qwen3.5:9b")
	rec.RecordCaller("qwen3.5:9b", "", clock.Add(time.Second))
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	later := clock.Add(time.Minute)
	rec.Record("qwen3.5:9b")
	rec.RecordCaller("QWEN3.5:9b", "scenario:ai-gateway-start-api/ai-gateway-api", later)
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	doc, err := ReadUsage(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := doc.Models["qwen3.5:9b"]
	gateway := entry.Callers["scenario:ai-gateway-start-api/ai-gateway-api"]
	unknown := entry.Callers[UnknownCaller]
	if entry.Requests != 3 || gateway.Requests != 2 || !gateway.LastRequestAt.Equal(later) || unknown.Requests != 1 {
		t.Fatalf("entry = %+v, want 3 requests split 2 gateway (last %s) and 1 unknown", entry, later)
	}
}

func TestCallerTableKeepsTheMostRecentCallers(t *testing.T) {
	var entry UsageEntry
	start := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	for i := 0; i <= maxCallersPerModel; i++ {
		entry.addCaller(fmt.Sprintf("process:caller-%d", i), CallerUsage{LastRequestAt: start.Add(time.Duration(i) * time.Second), Requests: 1})
	}
	if len(entry.Callers) != maxCallersPerModel {
		t.Fatalf("callers = %d, want the cap %d", len(entry.Callers), maxCallersPerModel)
	}
	if _, kept := entry.Callers["process:caller-0"]; kept {
		t.Fatal("the least recently seen caller should be dropped first")
	}
}

func TestReadUsageOfAMissingFileMeansUnknown(t *testing.T) {
	doc, err := ReadUsage(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !doc.TrackingSince.IsZero() || !doc.LastActivity("x").IsZero() {
		t.Fatalf("missing usage must read as unknown: %+v", doc)
	}
}

func TestRequestModelSeparatesUseFromUnload(t *testing.T) {
	cases := []struct {
		body  string
		model string
		use   bool
	}{
		{`{"model":"qwen3:4b","prompt":"hi"}`, "qwen3:4b", true},
		{`{"model":"qwen3:4b","messages":[{"role":"user","content":"x"}]}`, "qwen3:4b", true},
		{`{"model":"nomic-embed-text","input":["a"]}`, "nomic-embed-text", true},
		{`{"model":"qwen3:4b","keep_alive":0}`, "qwen3:4b", false},
		{`{"model":"laya","keep_alive":0}`, "laya", false},
		{`{"model":"laya","state":"x","questions":[{}]}`, "laya", true},
		{`{"name":"llava:7b"}`, "llava:7b", true},
		{`{"prompt":"no model"}`, "", false},
		{`not json`, "", false},
	}
	for _, tc := range cases {
		model, use := RequestModel([]byte(tc.body))
		if model != tc.model || use != tc.use {
			t.Errorf("RequestModel(%s) = (%q, %t), want (%q, %t)", tc.body, model, use, tc.model, tc.use)
		}
	}
}

func TestReadFootprintCountsSharedBlobsOnce(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		path := filepath.Join(root, "manifests", rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("registry.ollama.ai/library/qwen3-vl/4b", `{"config":{"digest":"sha256:c1","size":10},"layers":[{"digest":"sha256:w4","size":4000},{"digest":"sha256:proj","size":500}]}`)
	write("registry.ollama.ai/library/qwen3-vl/2b", `{"config":{"digest":"sha256:c2","size":10},"layers":[{"digest":"sha256:w2","size":2000},{"digest":"sha256:proj","size":500}]}`)
	write("registry.ollama.ai/huihui_ai/granite/8b", `{"config":{"digest":"sha256:c3","size":10},"layers":[{"digest":"sha256:g","size":8000}]}`)
	fp, err := ReadFootprint(root, "registry.ollama.ai")
	if err != nil {
		t.Fatal(err)
	}
	if got := fp.Models["qwen3-vl:4b"]; got.Bytes != 4510 || got.UniqueBytes != 4010 {
		t.Fatalf("qwen3-vl:4b = %+v", got)
	}
	if got := fp.Models["huihui_ai/granite:8b"]; got.UniqueBytes != 8010 {
		t.Fatalf("namespaced model = %+v", got)
	}
	if fp.StoreBytes != 10+4000+500+10+2000+10+8000 {
		t.Fatalf("store bytes = %d", fp.StoreBytes)
	}
}

func TestReceiptsReturnTheLatestForAModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evictions.jsonl")
	ctx := context.Background()
	for _, r := range []Receipt{
		{Model: "llava:7b", Status: ReceiptEvicting, Reacquire: []string{"resource-ollama", "models", "reacquire", "llava:7b"}},
		{Model: "qwen3:4b", Status: ReceiptEvicting},
		{Model: "llava:7b", Status: ReceiptEvicted, Bytes: 4733363377},
	} {
		if err := AppendReceipt(ctx, path, r); err != nil {
			t.Fatal(err)
		}
	}
	got, ok, err := LatestReceipt(path, "LLAVA:7b")
	if err != nil || !ok || got.Status != ReceiptEvicted || got.Bytes != 4733363377 {
		t.Fatalf("latest = %+v ok=%t err=%v", got, ok, err)
	}
	if _, ok, _ := LatestReceipt(path, "absent:1b"); ok {
		t.Fatal("absent model must have no receipt")
	}
}

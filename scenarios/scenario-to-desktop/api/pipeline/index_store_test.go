package pipeline

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type indexStoreLogger struct{ infos, errors int }

func (l *indexStoreLogger) Info(string, ...interface{})  { l.infos++ }
func (l *indexStoreLogger) Warn(string, ...interface{})  {}
func (l *indexStoreLogger) Error(string, ...interface{}) { l.errors++ }
func (l *indexStoreLogger) Debug(string, ...interface{}) {}

func TestScenarioIndexStorePersistsArchivesAndReloadsIndexes(t *testing.T) {
	dir := t.TempDir()
	clock := int64(100)
	store, err := NewScenarioIndexStore(dir, WithIndexStoreTimeFunc(func() int64 { return clock }))
	if err != nil {
		t.Fatal(err)
	}
	if index := store.GetOrCreate("alpha"); index.UpdatedAt != 100 || index.MaxHistorySize != DefaultMaxHistorySize {
		t.Fatalf("new index = %#v", index)
	}
	clock = 200
	if err := store.SetActivePipeline("alpha", "pipeline-1"); err != nil {
		t.Fatal(err)
	}
	archived, err := store.ArchiveActive("alpha")
	if err != nil || archived != "pipeline-1" || store.Get("alpha").ActivePipelineID != "" || len(store.GetHistory("alpha", 1)) != 1 {
		t.Fatalf("ArchiveActive() = %q, %v; index=%#v", archived, err, store.Get("alpha"))
	}
	if archived, err := store.ArchiveActive("missing"); err != nil || archived != "" {
		t.Fatalf("empty archive = %q, %v", archived, err)
	}
	if err := store.SetActivePipeline("beta", "pipeline-2"); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearActive("beta"); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearActive("unknown"); err != nil {
		t.Fatal(err)
	}
	names := store.List()
	sort.Strings(names)
	if got := names; len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("List() = %#v", got)
	}

	reloaded, err := NewScenarioIndexStore(dir)
	if err != nil || reloaded.Get("alpha") == nil || reloaded.GetHistory("alpha", 0)[0] != "pipeline-1" {
		t.Fatalf("reloaded index = %#v, %v", reloaded.Get("alpha"), err)
	}
}

func TestScenarioIndexStoreSkipsCorruptFilesAndReportsLoadDiagnostics(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "valid.json"), []byte(`{"scenario_name":"valid","history":["old"],"max_history_size":5}`), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := &indexStoreLogger{}
	store, err := NewScenarioIndexStore(dir, WithIndexStoreLogger(logger))
	if err != nil || store.Get("valid") == nil || store.Get("broken") != nil || logger.errors != 1 || logger.infos != 1 {
		t.Fatalf("load result valid=%#v errors=%d infos=%d err=%v", store.Get("valid"), logger.errors, logger.infos, err)
	}
	if _, err := store.readFromFile(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("expected missing index read error")
	}
}

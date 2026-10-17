package ai

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLConfigStoreLifecycle(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/ai.db")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE ai_provider_configs (name TEXT PRIMARY KEY, enabled INTEGER, priority INTEGER, timeout_sec INTEGER, max_retries INTEGER); INSERT INTO ai_provider_configs VALUES ('ollama', 1, 1, 7, 2), ('openrouter', 0, 2, 11, 3);`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	store := NewSQLConfigStore(ctx, db)
	configs := store.GetConfigs(ctx)
	if len(configs) != 2 || configs[0].Name != "ollama" || configs[0].TimeoutSec != 7 {
		t.Fatalf("configs = %+v", configs)
	}
	if !store.UpdateConfig(ctx, "ollama", false, 3, 15, 4) || store.UpdateConfig(ctx, "missing", true, 1, 1, 1) {
		t.Fatal("unexpected update results")
	}
	if store.IsEnabled(ctx, "ollama") || store.IsEnabled(ctx, "missing") {
		t.Fatal("enabled state was not updated")
	}
	if store.GetProviderTimeout(ctx, "missing") != DefaultProviderTimeout {
		t.Fatal("missing provider did not use default timeout")
	}
	store.RecordSuccess(ctx, "ollama", 12*time.Millisecond)
	store.RecordError(ctx, "openrouter")
	store.RecordSuccess(ctx, "missing", time.Second)
	health := store.GetHealth(ctx)
	if len(health) != 2 {
		t.Fatalf("health = %+v", health)
	}
}

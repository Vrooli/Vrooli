package maintenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupUserStorageMigratesResourcesAndRemovesLegacyPaths(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	legacyRoot := filepath.Join(home, ".vrooli")

	writeFile := func(rel, content string) {
		path := filepath.Join(legacyRoot, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeDir := func(rel string) {
		path := filepath.Join(legacyRoot, rel)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}

	writeFile("browserless/functions.json", `{}`)
	writeFile("redis/config/redis.conf", "save 60 1\n")
	writeFile("redis/data/dump.rdb", "rdb")
	writeFile("redis/logs/redis.log", "ok\n")
	writeFile("redis/backups/one.rdb", "backup")
	writeFile("twilio/phone-numbers.json", `{}`)
	writeFile("twilio-credentials.json", `{}`)
	writeDir("comment-system")
	writeFile("resources.local.json.backup.2025-07-26T13-42-28-502Z", `{}`)

	origRunning := resourceRunningFn
	origAction := runResourceActionFn
	t.Cleanup(func() {
		resourceRunningFn = origRunning
		runResourceActionFn = origAction
	})

	actions := make([]string, 0, 4)
	resourceRunningFn = func(root, home, name string) (bool, error) {
		return name == "redis", nil
	}
	runResourceActionFn = func(root, home, name, action string) error {
		actions = append(actions, name+":"+action)
		return nil
	}

	controller := NewController(root, home)
	report, err := controller.CleanupUserStorage()
	if err != nil {
		t.Fatalf("CleanupUserStorage: %v", err)
	}

	if got, want := strings.Join(actions, ","), "redis:stop,redis:start"; got != want {
		t.Fatalf("resource actions = %q, want %q", got, want)
	}

	assertExists(t, filepath.Join(home, ".local", "share", "vrooli", "resources", "browserless", "functions.json"))
	assertExists(t, filepath.Join(home, ".config", "vrooli", "resources", "redis", "redis.conf"))
	assertExists(t, filepath.Join(home, ".local", "share", "vrooli", "resources", "redis", "dump.rdb"))
	assertExists(t, filepath.Join(home, ".local", "state", "logs", "vrooli", "resources", "redis", "redis.log"))
	assertExists(t, filepath.Join(home, ".local", "state", "vrooli", "resources", "redis", "backups", "one.rdb"))
	assertExists(t, filepath.Join(home, ".config", "vrooli", "resources", "twilio", "phone-numbers.json"))
	assertExists(t, filepath.Join(home, ".config", "vrooli", "resources", "twilio", "credentials.json"))

	assertNotExists(t, filepath.Join(legacyRoot, "browserless"))
	assertNotExists(t, filepath.Join(legacyRoot, "redis"))
	assertNotExists(t, filepath.Join(legacyRoot, "twilio"))
	assertNotExists(t, filepath.Join(legacyRoot, "twilio-credentials.json"))
	assertNotExists(t, filepath.Join(legacyRoot, "comment-system"))
	assertNotExists(t, filepath.Join(legacyRoot, "resources.local.json.backup.2025-07-26T13-42-28-502Z"))

	if len(report.Actions) == 0 {
		t.Fatal("expected report actions")
	}
}

func TestCleanupUserStorageIsIdempotent(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	origRunning := resourceRunningFn
	origAction := runResourceActionFn
	t.Cleanup(func() {
		resourceRunningFn = origRunning
		runResourceActionFn = origAction
	})
	resourceRunningFn = func(root, home, name string) (bool, error) { return false, nil }
	runResourceActionFn = func(root, home, name, action string) error { return nil }

	controller := NewController(root, home)
	report, err := controller.CleanupUserStorage()
	if err != nil {
		t.Fatalf("CleanupUserStorage: %v", err)
	}
	if len(report.Actions) != 0 {
		t.Fatalf("actions = %#v, want none", report.Actions)
	}
}

func TestCleanupUserStorageReportsConflictsWhenCanonicalDataDiffers(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	legacyRoot := filepath.Join(home, ".vrooli")

	if err := os.MkdirAll(filepath.Join(legacyRoot, "twilio"), 0o755); err != nil {
		t.Fatalf("mkdir legacy twilio: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "twilio-credentials.json"), []byte(`{"sid":"legacy"}`), 0o600); err != nil {
		t.Fatalf("write legacy credentials: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "twilio", "phone-numbers.json"), []byte(`{"legacy":true}`), 0o644); err != nil {
		t.Fatalf("write legacy phone numbers: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(home, ".config", "vrooli", "resources", "twilio"), 0o755); err != nil {
		t.Fatalf("mkdir canonical twilio config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "vrooli", "resources", "twilio", "credentials.json"), []byte(`{"sid":"canonical"}`), 0o644); err != nil {
		t.Fatalf("write canonical credentials: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "vrooli", "resources", "twilio", "phone-numbers.json"), []byte(`{"canonical":true}`), 0o644); err != nil {
		t.Fatalf("write canonical phone numbers: %v", err)
	}

	origRunning := resourceRunningFn
	origAction := runResourceActionFn
	t.Cleanup(func() {
		resourceRunningFn = origRunning
		runResourceActionFn = origAction
	})
	resourceRunningFn = func(root, home, name string) (bool, error) { return false, nil }
	runResourceActionFn = func(root, home, name, action string) error { return nil }

	controller := NewController(root, home)
	report, err := controller.CleanupUserStorage()
	if err != nil {
		t.Fatalf("CleanupUserStorage: %v", err)
	}

	conflicts := 0
	for _, action := range report.Actions {
		if action.Kind == "conflict" {
			conflicts++
		}
	}
	if conflicts < 2 {
		t.Fatalf("expected twilio conflicts in report, got %#v", report.Actions)
	}

	assertExists(t, filepath.Join(legacyRoot, "twilio-credentials.json"))
	assertExists(t, filepath.Join(legacyRoot, "twilio", "phone-numbers.json"))
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to not exist: %v", path, err)
	}
}

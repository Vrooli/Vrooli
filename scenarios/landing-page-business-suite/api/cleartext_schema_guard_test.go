package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCredentialSchemaGuardRejectsCleartextProviderSecrets(t *testing.T) {
	column := regexp.MustCompile(`(?i)\b(secret_key|secret_access_key|session_token|webhook_secret)\b`)
	// These existing schemas use the terms as identifiers or session metadata,
	// not as paid-path provider credential values. Keep the exceptions explicit
	// so a new payment/storage secret column cannot return unnoticed.
	allowed := map[string]bool{
		filepath.Clean("../../file-tools/api/internal/files/schema.sql"):                true,
		filepath.Clean("../../data-tools/api/internal/data/schema.sql"):                 true,
		filepath.Clean("../../secrets-manager/api/internal/secrets/schema.sql"):         true,
		filepath.Clean("../../secrets-manager/api/internal/secrets/desktop_schema.sql"): true,
	}
	err := filepath.Walk("../../", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".sql") || allowed[filepath.Clean(path)] {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for lineNumber, line := range strings.Split(string(contents), "\n") {
			if column.MatchString(line) {
				t.Errorf("%s:%d contains a cleartext credential column: %s", path, lineNumber+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

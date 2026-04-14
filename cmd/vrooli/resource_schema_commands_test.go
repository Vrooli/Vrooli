package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestRunResourceSchemaValidateJSON(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, filepath.Join("resources", "redis", repocontractmeta.ResourceManifestFilename), `{
  "$schema": "../../.vrooli/schemas/resource.schema.json",
  "name": "redis",
  "display_name": "Redis",
  "description": "Cache",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "full",
  "runtime": {
    "image": "redis:7-alpine"
  }
}`)
	writeTestFile(t, root, filepath.Join("scenarios", "alpha", repocontractmeta.ServiceManifestPathname), `{
  "service": {"name": "alpha"},
  "dependencies": {
    "resources": {
      "redis": {"enabled": true}
    }
  }
}`)
	if _, err := resources.SyncSchemaArtifacts(root); err != nil {
		t.Fatalf("SyncSchemaArtifacts: %v", err)
	}

	app := newTestApp(root)
	var stdout bytes.Buffer
	code := app.Run([]string{"resource", "schema", "validate", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%s", code, stdout.String())
	}
	var payload struct {
		Success bool `json:"success"`
		Report  struct {
			Passed bool `json:"passed"`
		} `json:"report"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !payload.Success || !payload.Report.Passed {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestRunResourceSchemaSyncReturnsSilentNonZeroForMissingResources(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, filepath.Join("resources", "redis", repocontractmeta.ResourceManifestFilename), `{
  "$schema": "../../.vrooli/schemas/resource.schema.json",
  "name": "redis",
  "display_name": "Redis",
  "description": "Cache",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "full",
  "runtime": {
    "image": "redis:7-alpine"
  }
}`)
	writeTestFile(t, root, filepath.Join("scenarios", "alpha", repocontractmeta.ServiceManifestPathname), `{
  "service": {"name": "alpha"},
  "dependencies": {
    "resources": {
      "n8n": {"enabled": true}
    }
  }
}`)

	app := newTestApp(root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"resource", "schema", "sync", "--json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected silent stderr, got %q", stderr.String())
	}
	var payload struct {
		Success bool `json:"success"`
		Report  struct {
			Passed            bool `json:"passed"`
			MissingReferences []struct {
				Resource string `json:"resource"`
			} `json:"missing_references"`
		} `json:"report"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload.Success || payload.Report.Passed {
		t.Fatalf("payload = %+v", payload)
	}
	if len(payload.Report.MissingReferences) != 1 || payload.Report.MissingReferences[0].Resource != "n8n" {
		t.Fatalf("payload = %+v", payload)
	}
}

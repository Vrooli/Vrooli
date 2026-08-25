package binaryfetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AcquisitionSchemaPath is the committed generated fragment consumed by tool
// and resource schemas.
const AcquisitionSchemaPath = ".vrooli/schemas/acquisition.schema.json"

// AcquisitionSchema returns the JSON Schema generated for Acquisition. Keep
// this function next to the owning Go types: consumers do not maintain a
// second hand-written field list.
func AcquisitionSchema() map[string]any {
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "acquisition.schema.json",
		"title":                "Declared artifact acquisition",
		"type":                 "object",
		"required":             []string{"kind", "targets"},
		"additionalProperties": false,
		"properties": map[string]any{
			"kind":    map[string]any{"type": "string", "enum": []string{"url", "oci-image", "none", "composed"}},
			"license": map[string]any{"type": "string"},
			"targets": map[string]any{
				"type": "array", "minItems": 1,
				"items": map[string]any{
					"type": "object", "additionalProperties": false,
					"properties": map[string]any{
						"when":            map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
						"kind":            map[string]any{"type": "string", "enum": []string{"url", "oci-image", "none", "composed"}},
						"url":             map[string]any{"type": "string", "format": "uri"},
						"image":           map[string]any{"type": "string"},
						"sha256":          map[string]any{"type": "string", "pattern": "^[a-fA-F0-9]{64}$"},
						"artifact_sha256": map[string]any{"type": "string", "pattern": "^[a-fA-F0-9]{64}$"},
						"archive":         map[string]any{"type": "string", "enum": []string{"tar.gz", "tar.bz2", "tar.zst", "zip", "none"}},
						"layout":          map[string]any{"type": "string", "enum": []string{"file", "dir"}},
						"bin_path":        map[string]any{"type": "string"},
						"mode":            map[string]any{"type": "string", "pattern": "^0[0-7]{3}$"},
						"executable":      map[string]any{"type": "string", "pattern": "^[A-Za-z0-9._-]+$"},
						"runtime_env":     map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
						"unsupported":     map[string]any{"type": "string", "minLength": 1},
						"compose": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object", "additionalProperties": false,
								"required": []string{"role", "kind", "dest"},
								"properties": map[string]any{
									"role":     map[string]any{"type": "string"},
									"kind":     map[string]any{"type": "string", "enum": []string{"url", "python-wheels", "local"}},
									"dest":     map[string]any{"type": "string"},
									"source":   map[string]any{"type": "string"},
									"url":      map[string]any{"type": "string", "format": "uri"},
									"sha256":   map[string]any{"type": "string", "pattern": "^[a-fA-F0-9]{64}$"},
									"archive":  map[string]any{"enum": []string{"tar.gz", "tar.bz2", "tar.zst", "zip"}, "type": "string"},
									"bin_path": map[string]any{"type": "string"},
									"mode":     map[string]any{"type": "string", "pattern": "^0[0-7]{3}$"},
									"lockfile": map[string]any{"type": "string"},
								},
							},
						},
					},
				},
			},
			"provenance": map[string]any{
				"type": "object", "additionalProperties": false,
				"required": []string{"kind"},
				"properties": map[string]any{
					"kind":                   map[string]any{"type": "string", "enum": []string{"none", "gpg-checksums"}},
					"key_url":                map[string]any{"type": "string", "format": "uri"},
					"checksum_manifest_url":  map[string]any{"type": "string", "format": "uri"},
					"checksum_signature_url": map[string]any{"type": "string", "format": "uri"},
					"fingerprint":            map[string]any{"type": "string"},
				},
			},
		},
	}
}

// RenderAcquisitionSchema is deterministic so drift is detected by a byte
// comparison rather than by a tolerant parser.
func RenderAcquisitionSchema() ([]byte, error) {
	data, err := json.MarshalIndent(AcquisitionSchema(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render acquisition schema: %w", err)
	}
	return append(data, '\n'), nil
}

// SyncAcquisitionSchema writes the generated fragment beneath root.
func SyncAcquisitionSchema(root string) error {
	data, err := RenderAcquisitionSchema()
	if err != nil {
		return err
	}
	path := filepath.Join(root, filepath.FromSlash(AcquisitionSchemaPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create acquisition schema directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write acquisition schema: %w", err)
	}
	return nil
}

// ValidateAcquisitionSchema fails when the committed generated fragment does
// not exactly match the owning Go contract.
func ValidateAcquisitionSchema(root string) error {
	want, err := RenderAcquisitionSchema()
	if err != nil {
		return err
	}
	path := filepath.Join(root, filepath.FromSlash(AcquisitionSchemaPath))
	got, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read acquisition schema %s: %w", path, err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("acquisition schema is stale at %s; regenerate it with binaryfetch.SyncAcquisitionSchema", path)
	}
	return nil
}

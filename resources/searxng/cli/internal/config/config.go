// Package config owns the SearXNG settings lifecycle. It preserves upstream
// YAML outside Vrooli's small, explicit policy overlay.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const settingsName = "settings.yml"

// Report intentionally contains no secret values and is safe to print.
type Report struct {
	Path       string `json:"path"`
	BackupPath string `json:"backup_path,omitempty"`
	Created    bool   `json:"created"`
	Secret     string `json:"secret"`
}

// ConfigDir returns the manifest storage location, with a deterministic
// per-user fallback for direct CLI use.
func ConfigDir(override string) (string, error) {
	if value := strings.TrimSpace(override); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(os.Getenv("RESOURCE_CONFIG_DIR")); value != "" {
		return value, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "vrooli", "resources", "searxng"), nil
}

func SettingsPath(dir string) string { return filepath.Join(dir, settingsName) }

// Load returns either the operator's full YAML document or a fresh document.
func Load(dir string) (*yaml.Node, bool, error) {
	data, err := os.ReadFile(SettingsPath(dir))
	if os.IsNotExist(err) {
		return freshDocument(), false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read settings: %w", err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, false, fmt.Errorf("parse settings: %w", err)
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("settings must contain one YAML mapping document")
	}
	return &document, true, nil
}

// Apply imports an existing document, applies only Vrooli-owned fields, and
// atomically writes it. An existing file is backed up before replacement.
func Apply(dir, baseURL, instanceName, secret string) (Report, error) {
	document, existed, err := Load(dir)
	if err != nil {
		return Report{}, err
	}
	hadSecret := strings.TrimSpace(findScalar(document, "server", "secret_key")) != ""
	providedSecret := strings.TrimSpace(secret) != ""
	if err := applyOwned(document, baseURL, instanceName, secret); err != nil {
		return Report{}, err
	}
	if err := Validate(document); err != nil {
		return Report{}, err
	}
	data, err := yaml.Marshal(document)
	if err != nil {
		return Report{}, fmt.Errorf("encode settings: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Report{}, fmt.Errorf("create config directory: %w", err)
	}
	path := SettingsPath(dir)
	report := Report{Path: path, Created: !existed, Secret: "preserved"}
	if !hadSecret {
		report.Secret = "generated"
		if providedSecret {
			report.Secret = "provided"
		}
	}
	if existed {
		backup := path + ".backup." + time.Now().UTC().Format("20060102T150405Z")
		current, readErr := os.ReadFile(path)
		if readErr != nil {
			return Report{}, fmt.Errorf("read settings for backup: %w", readErr)
		}
		if err := os.WriteFile(backup, current, 0o600); err != nil {
			return Report{}, fmt.Errorf("back up settings: %w", err)
		}
		report.BackupPath = backup
	}
	if err := atomicWrite(path, data); err != nil {
		return Report{}, err
	}
	return report, nil
}

func Validate(document *yaml.Node) error {
	if document == nil || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("settings must contain one YAML mapping document")
	}
	formats := findSequence(document, "search", "formats")
	if len(formats) > 0 && !contains(formats, "json") {
		return fmt.Errorf("search.formats must include json for the declared SearXNG API contract")
	}
	if secret := strings.TrimSpace(findScalar(document, "server", "secret_key")); secret == "" || strings.Contains(secret, "${") {
		return fmt.Errorf("server.secret_key must be a materialized non-empty secret")
	}
	return nil
}

func RedactedSummary(document *yaml.Node) map[string]any {
	return map[string]any{
		"use_default_settings": findRootScalar(document, "use_default_settings"),
		"instance_name":        findScalar(document, "general", "instance_name"),
		"base_url":             findScalar(document, "server", "base_url"),
		"secret_key":           "[redacted]",
		"formats":              findSequence(document, "search", "formats"),
	}
}

func freshDocument() *yaml.Node {
	return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
}

func applyOwned(document *yaml.Node, baseURL, instanceName, requestedSecret string) error {
	root := document.Content[0]
	setScalar(root, "use_default_settings", "true", "!!bool")
	general := ensureMap(root, "general")
	if strings.TrimSpace(instanceName) == "" {
		instanceName = "Vrooli SearXNG"
	}
	setScalar(general, "instance_name", instanceName, "!!str")
	setScalar(general, "enable_metrics", "true", "!!bool")
	search := ensureMap(root, "search")
	if formats := findSequenceInMap(search, "formats"); len(formats) > 0 && !contains(formats, "json") {
		return fmt.Errorf("search.formats must include json for the declared SearXNG API contract")
	}
	ensureFormats(search)
	server := ensureMap(root, "server")
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://localhost:8280"
	}
	setScalar(server, "base_url", baseURL, "!!str")
	setScalar(server, "limiter", "false", "!!bool")
	if strings.TrimSpace(findScalar(document, "server", "secret_key")) == "" {
		if strings.TrimSpace(requestedSecret) == "" {
			var err error
			requestedSecret, err = randomSecret()
			if err != nil {
				return err
			}
		}
		setScalar(server, "secret_key", requestedSecret, "!!str")
	}
	return nil
}

func randomSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate session secret: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func atomicWrite(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return fmt.Errorf("create temporary settings: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect temporary settings: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary settings: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary settings: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace settings atomically: %w", err)
	}
	return nil
}

func ensureMap(root *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key && root.Content[i+1].Kind == yaml.MappingNode {
			return root.Content[i+1]
		}
	}
	value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
	return value
}

func setScalar(root *yaml.Node, key, value, tag string) {
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
			return
		}
	}
	root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value})
}

func ensureFormats(search *yaml.Node) {
	values := findSequenceInMap(search, "formats")
	if contains(values, "json") && contains(values, "html") {
		return
	}
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, value := range []string{"html", "json"} {
		sequence.Content = append(sequence.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
	}
	for i := 0; i < len(search.Content); i += 2 {
		if search.Content[i].Value == "formats" {
			search.Content[i+1] = sequence
			return
		}
	}
	search.Content = append(search.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "formats"}, sequence)
}

func findRootScalar(document *yaml.Node, key string) string {
	if document == nil || len(document.Content) == 0 {
		return ""
	}
	for i := 0; i < len(document.Content[0].Content); i += 2 {
		if document.Content[0].Content[i].Value == key {
			return document.Content[0].Content[i+1].Value
		}
	}
	return ""
}

func findScalar(document *yaml.Node, section, key string) string {
	if document == nil || len(document.Content) == 0 {
		return ""
	}
	for i := 0; i < len(document.Content[0].Content); i += 2 {
		if document.Content[0].Content[i].Value == section {
			return findScalarInMap(document.Content[0].Content[i+1], key)
		}
	}
	return ""
}

func findScalarInMap(root *yaml.Node, key string) string {
	if root == nil || root.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1].Value
		}
	}
	return ""
}

func findSequence(document *yaml.Node, section, key string) []string {
	if document == nil || len(document.Content) == 0 {
		return nil
	}
	for i := 0; i < len(document.Content[0].Content); i += 2 {
		if document.Content[0].Content[i].Value == section {
			return findSequenceInMap(document.Content[0].Content[i+1], key)
		}
	}
	return nil
}

func findSequenceInMap(root *yaml.Node, key string) []string {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key && root.Content[i+1].Kind == yaml.SequenceNode {
			out := make([]string, 0, len(root.Content[i+1].Content))
			for _, n := range root.Content[i+1].Content {
				out = append(out, n.Value)
			}
			return out
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

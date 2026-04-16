package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	resourceenv "resource-gemini/cli/internal/env"
)

var ErrContentNotFound = errors.New("gemini content not found")

const (
	ContentKindPrompt   = "prompt"
	ContentKindTemplate = "template"
	ContentKindFunction = "function"
)

// ContentStore owns repo-external Gemini prompt/template/function storage.
type ContentStore struct {
	Runtime resourceenv.Runtime
}

// NewContentStore returns a content store rooted in the provided runtime.
func NewContentStore(runtime resourceenv.Runtime) ContentStore {
	return ContentStore{Runtime: runtime}
}

// EnsureInitialized creates the content storage directories.
func (s ContentStore) EnsureInitialized() error {
	return s.Runtime.EnsureDirectories()
}

// Add stores a named Gemini content asset.
func (s ContentStore) Add(kind, name string, payload []byte) error {
	path, err := s.pathFor(kind, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

// Get returns a named Gemini content asset.
func (s ContentStore) Get(kind, name string) ([]byte, error) {
	path, err := s.pathFor(kind, name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}
	return data, nil
}

// Remove deletes a named Gemini content asset.
func (s ContentStore) Remove(kind, name string) error {
	path, err := s.pathFor(kind, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrContentNotFound
		}
		return err
	}
	return nil
}

// List enumerates stored content names by kind, or across all kinds when kind is empty or "all".
func (s ContentStore) List(kind string) ([]string, error) {
	kinds := []string{ContentKindPrompt, ContentKindTemplate, ContentKindFunction}
	if normalized := normalizeKind(kind); normalized != "" && normalized != "all" {
		kinds = []string{normalized}
	}

	var out []string
	for _, current := range kinds {
		dir, suffix, err := s.dirAndSuffix(current)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, suffix) {
				continue
			}
			out = append(out, current+":"+strings.TrimSuffix(name, suffix))
		}
	}
	sort.Strings(out)
	return out, nil
}

// GenerateContentEndpoint returns the Gemini generate-content URL for a model.
func GenerateContentEndpoint(apiBaseURL, model string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	model = strings.Trim(strings.TrimSpace(model), "/")
	if base == "" || model == "" {
		return ""
	}
	if strings.HasPrefix(model, "models/") {
		return base + "/" + model + ":generateContent"
	}
	return base + "/models/" + model + ":generateContent"
}

// ModelsEndpoint returns the Gemini model-listing URL.
func ModelsEndpoint(apiBaseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/models"
}

func (s ContentStore) pathFor(kind, name string) (string, error) {
	dir, suffix, err := s.dirAndSuffix(kind)
	if err != nil {
		return "", err
	}
	name = sanitizeName(name)
	if name == "" {
		return "", fmt.Errorf("content name is required")
	}
	return filepath.Join(dir, name+suffix), nil
}

func (s ContentStore) dirAndSuffix(kind string) (string, string, error) {
	switch normalizeKind(kind) {
	case ContentKindPrompt:
		return s.Runtime.PromptsDir, ".txt", nil
	case ContentKindTemplate:
		return s.Runtime.TemplatesDir, ".json", nil
	case ContentKindFunction:
		return s.Runtime.FunctionsDir, ".json", nil
	default:
		return "", "", fmt.Errorf("unknown content kind: %s", kind)
	}
}

func normalizeKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "", "all":
		return strings.TrimSpace(strings.ToLower(kind))
	case "prompt", "prompts":
		return ContentKindPrompt
	case "template", "templates":
		return ContentKindTemplate
	case "function", "functions":
		return ContentKindFunction
	default:
		return strings.TrimSpace(strings.ToLower(kind))
	}
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "..", "")
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return strings.TrimSpace(name)
}

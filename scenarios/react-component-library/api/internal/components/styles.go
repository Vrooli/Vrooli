package components

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DesignStyle struct {
	ID       string
	Name     string
	Tags     []string
	Supports []string
}

func (s *service) ListDesignStyles(ctx context.Context) ([]DesignStyle, error) {
	root, err := defaultDesignRoot()
	if err != nil {
		return nil, err
	}
	return LoadDesignStyles(ctx, root)
}

func (s *service) ValidateDesignStyle(ctx context.Context, id string) error {
	styles, err := s.ListDesignStyles(ctx)
	if err != nil {
		return err
	}
	want := strings.TrimSpace(id)
	for _, style := range styles {
		if style.ID == want {
			return nil
		}
	}
	return ErrInvalidHeader{SourcePath: "component.json", Field: "designStyles", Reason: fmt.Sprintf("unknown style id %q", want)}
}

func LoadDesignStyles(_ context.Context, root string) ([]DesignStyle, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read design styles: %w", err)
	}
	var out []DesignStyle
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name(), "metadata.json"))
		if err != nil {
			return nil, fmt.Errorf("read design metadata %q: %w", entry.Name(), err)
		}
		var mf struct {
			ID       string   `json:"id"`
			Name     string   `json:"name"`
			Tags     []string `json:"tags"`
			Adapters map[string]struct {
				Supports []string `json:"supports"`
			} `json:"adapters"`
		}
		if err := json.Unmarshal(raw, &mf); err != nil {
			return nil, fmt.Errorf("parse design metadata %q: %w", entry.Name(), err)
		}
		style := DesignStyle{
			ID:   strings.TrimSpace(mf.ID),
			Name: strings.TrimSpace(mf.Name),
			Tags: append([]string(nil), mf.Tags...),
		}
		if style.ID == "" {
			return nil, fmt.Errorf("design metadata %q: id required", entry.Name())
		}
		for _, adapter := range mf.Adapters {
			style.Supports = append(style.Supports, adapter.Supports...)
		}
		sort.Strings(style.Supports)
		out = append(out, style)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func defaultDesignRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve design root: get working directory: %w", err)
	}
	for depth, dir := 0, filepath.Clean(cwd); depth < 12; depth++ {
		candidate := filepath.Join(dir, "templates", "design")
		marker := filepath.Join(candidate, "vrooli-default", "metadata.json")
		raw, err := os.ReadFile(marker)
		if err == nil {
			var metadata struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &metadata); err != nil {
				return "", fmt.Errorf("parse default design marker %q: %w", marker, err)
			}
			metadataID := strings.TrimSpace(metadata.ID)
			if metadataID == "" {
				return "", fmt.Errorf("default design marker %q: id required", marker)
			}
			if metadataID == "vrooli-default" {
				return candidate, nil
			}
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("read default design marker %q: %w", marker, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("resolve design root: canonical templates/design with vrooli-default/metadata.json not found within 12 parent directories of %q", cwd)
}

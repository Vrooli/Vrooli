package components

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve design root: runtime caller unavailable")
	}
	candidates := []string{
		filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "templates", "design"),
		filepath.Join("..", "..", "templates", "design"),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "templates", "design"),
			filepath.Join(cwd, "..", "..", "templates", "design"),
			filepath.Join(cwd, "..", "..", "..", "templates", "design"),
		)
	}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			absolute, err := filepath.Abs(candidate)
			if err == nil {
				return absolute, nil
			}
		}
	}
	return filepath.Abs(filepath.Clean(candidates[0]))
}

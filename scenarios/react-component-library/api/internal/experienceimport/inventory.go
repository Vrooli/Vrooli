package experienceimport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sketch "react-component-library/internal/sketch"
)

// InventoryItem describes a composition-bearing experience page without
// changing either its source file or workspace storage. ScenarioRoot is the
// directory containing scenario directories (normally <repo>/scenarios).
type InventoryItem struct {
	Scenario       string `json:"scenario"`
	Page           string `json:"page"`
	ContentHash    string `json:"contentHash"`
	Regions        int    `json:"regions"`
	Placements     int    `json:"placements"`
	CanConvert     bool   `json:"canConvert"`
	ConverterError string `json:"converterError,omitempty"`
}

// InventoryCompositions discovers experience pages that contain composition
// data. Intent-only sketches (viewport, intent, notes, or semantic regions)
// are excluded. Symlinks in the discovered scenario/page path are rejected so
// an inventory cannot escape ScenarioRoot or silently omit external data.
func InventoryCompositions(scenarioRoot string) ([]InventoryItem, error) {
	root, err := filepath.Abs(scenarioRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario root symlinks: %w", err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat scenario root: %w", err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("scenario root %q is not a directory", root)
	}

	scenarios, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read scenario root: %w", err)
	}
	var items []InventoryItem
	for _, scenarioEntry := range scenarios {
		if scenarioEntry.Type()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing symlink scenario path %q", filepath.Join(root, scenarioEntry.Name()))
		}
		if !scenarioEntry.IsDir() || !safeIdentity(scenarioEntry.Name()) {
			continue
		}
		scenarioDir := filepath.Join(root, scenarioEntry.Name())
		if err := ensureContainedDirectory(root, scenarioDir); err != nil {
			return nil, err
		}
		pagesDir := filepath.Join(scenarioDir, "experience", "pages")
		if err := ensureContainedDirectory(root, pagesDir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		pages, err := os.ReadDir(pagesDir)
		if err != nil {
			return nil, fmt.Errorf("read pages for scenario %q: %w", scenarioEntry.Name(), err)
		}
		for _, pageEntry := range pages {
			if pageEntry.IsDir() || filepath.Ext(pageEntry.Name()) != ".json" {
				continue
			}
			page := strings.TrimSuffix(pageEntry.Name(), ".json")
			if !safeIdentity(page) {
				continue
			}
			path := filepath.Join(pagesDir, pageEntry.Name())
			info, err := os.Lstat(path)
			if err != nil {
				return nil, fmt.Errorf("inspect %s: %w", path, err)
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				return nil, fmt.Errorf("refusing symlink experience page %q", path)
			}
			if !info.Mode().IsRegular() {
				continue
			}
			if err := ensureContainedFile(root, path); err != nil {
				return nil, err
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", path, err)
			}
			item, composition, err := inspectPage(scenarioEntry.Name(), page, source)
			if err != nil {
				return nil, fmt.Errorf("inspect %s: %w", path, err)
			}
			if composition {
				items = append(items, item)
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Scenario == items[j].Scenario {
			return items[i].Page < items[j].Page
		}
		return items[i].Scenario < items[j].Scenario
	})
	return items, nil
}

func inspectPage(scenario, page string, source []byte) (InventoryItem, bool, error) {
	var envelope struct {
		Page struct {
			ID string `json:"id"`
		} `json:"page"`
		Sketch json.RawMessage `json:"sketch"`
	}
	if err := json.Unmarshal(source, &envelope); err != nil {
		return InventoryItem{}, false, err
	}
	if len(envelope.Sketch) == 0 || string(envelope.Sketch) == "null" {
		return InventoryItem{}, false, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Sketch, &fields); err != nil {
		return InventoryItem{}, false, fmt.Errorf("decode sketch object: %w", err)
	}
	if fields == nil {
		return InventoryItem{}, false, nil
	}
	var doc sketch.Document
	if err := json.Unmarshal(envelope.Sketch, &doc); err != nil {
		return InventoryItem{}, false, fmt.Errorf("decode sketch fields: %w", err)
	}
	composition := doc.Template != nil || doc.Render != nil || len(doc.Placements) > 0 || len(doc.Unplaced) > 0
	if !composition {
		return InventoryItem{}, false, nil
	}
	digest := sha256.Sum256(source)
	item := InventoryItem{Scenario: scenario, Page: page, ContentHash: hex.EncodeToString(digest[:]), Regions: len(doc.Regions), Placements: len(doc.Placements)}
	if envelope.Page.ID != page {
		item.ConverterError = fmt.Sprintf("page identity mismatch: file is %q, document declares %q", page, envelope.Page.ID)
		return item, true, nil
	}
	_, err := ConvertPage(scenario, page, source)
	if err != nil {
		item.ConverterError = err.Error()
		return item, true, nil
	}
	item.CanConvert = true
	return item, true, nil
}

func ensureContainedDirectory(root, path string) error {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect inventory path %q: %w", current, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in inventory path %q", current)
		}
		if current == root {
			break
		}
		if !within(root, current) || filepath.Dir(current) == current {
			return fmt.Errorf("inventory path %q escapes scenario root %q", path, root)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect inventory directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("inventory path %q is not a directory", path)
	}
	return nil
}

func ensureContainedFile(root, path string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve inventory file %q: %w", path, err)
	}
	if !within(root, resolved) {
		return fmt.Errorf("inventory file %q escapes scenario root %q", path, root)
	}
	return nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

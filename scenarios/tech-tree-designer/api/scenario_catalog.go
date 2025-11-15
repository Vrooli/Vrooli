package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type ScenarioCatalogEntry struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	RelativePath string    `json:"relative_path"`
	Tags         []string  `json:"tags"`
	Dependencies []string  `json:"dependencies"`
	Resources    []string  `json:"resources"`
	Hidden       bool      `json:"hidden"`
	LastModified time.Time `json:"last_modified"`
}

type ScenarioDependencyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type scenarioVisibilityConfig struct {
	Hidden []string `json:"hidden"`
}

type ScenarioCatalogManager struct {
	mu             sync.RWMutex
	repoRoot       string
	visibilityPath string
	entries        map[string]ScenarioCatalogEntry
	edges          []ScenarioDependencyEdge
	lastSynced     time.Time
	hiddenSet      map[string]struct{}
}

type serviceFile struct {
	Service struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"service"`
	Dependencies struct {
		Scenarios map[string]json.RawMessage `json:"scenarios"`
		Resources map[string]json.RawMessage `json:"resources"`
	} `json:"dependencies"`
}

func NewScenarioCatalogManager(repoRoot, visibilityPath string) (*ScenarioCatalogManager, error) {
	manager := &ScenarioCatalogManager{
		repoRoot:       repoRoot,
		visibilityPath: visibilityPath,
		entries:        make(map[string]ScenarioCatalogEntry),
		hiddenSet:      make(map[string]struct{}),
	}

	if err := manager.ensureVisibilityFile(); err != nil {
		return nil, err
	}

	if err := manager.loadVisibilityConfig(); err != nil {
		return nil, err
	}

	if err := manager.Refresh(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *ScenarioCatalogManager) ensureVisibilityFile() error {
	if m.visibilityPath == "" {
		return errors.New("visibility path not configured")
	}

	if _, err := os.Stat(m.visibilityPath); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(m.visibilityPath), 0o755); err != nil {
			return err
		}
		cfg := scenarioVisibilityConfig{Hidden: []string{}}
		data, marshalErr := json.MarshalIndent(cfg, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		if writeErr := os.WriteFile(m.visibilityPath, data, 0o644); writeErr != nil {
			return writeErr
		}
		return nil
	}

	return nil
}

func (m *ScenarioCatalogManager) loadVisibilityConfig() error {
	raw, err := os.ReadFile(m.visibilityPath)
	if err != nil {
		return err
	}

	var cfg scenarioVisibilityConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	hidden := make(map[string]struct{}, len(cfg.Hidden))
	for _, name := range cfg.Hidden {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		hidden[strings.ToLower(trimmed)] = struct{}{}
	}

	m.hiddenSet = hidden
	return nil
}

func (m *ScenarioCatalogManager) persistVisibilityConfigLocked() error {
	cfg := scenarioVisibilityConfig{Hidden: make([]string, 0, len(m.hiddenSet))}
	for name := range m.hiddenSet {
		cfg.Hidden = append(cfg.Hidden, name)
	}
	sort.Strings(cfg.Hidden)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmp := m.visibilityPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, m.visibilityPath)
}

func (m *ScenarioCatalogManager) Refresh() error {
	entries, edges, err := m.scanScenarios()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = entries
	m.edges = edges
	m.lastSynced = time.Now().UTC()
	return nil
}

func (m *ScenarioCatalogManager) scanScenarios() (map[string]ScenarioCatalogEntry, []ScenarioDependencyEdge, error) {
	scenariosDir := filepath.Join(m.repoRoot, "scenarios")
	log.Printf("DEBUG: scanScenarios - repoRoot=%s, scenariosDir=%s", m.repoRoot, scenariosDir)

	stat, err := os.Stat(scenariosDir)
	if err != nil {
		log.Printf("ERROR: scanScenarios - failed to stat scenarios dir: %v", err)
		return nil, nil, err
	}
	if !stat.IsDir() {
		return nil, nil, fmt.Errorf("scenarios directory missing at %s", scenariosDir)
	}

	entries := make(map[string]ScenarioCatalogEntry)
	edges := make([]ScenarioDependencyEdge, 0)

	log.Printf("DEBUG: scanScenarios - starting WalkDir on %s", scenariosDir)

	err = filepath.WalkDir(scenariosDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Printf("DEBUG: scanScenarios - walkErr for path=%s: %v", path, walkErr)
			return walkErr
		}

		if !d.IsDir() {
			return nil
		}

		servicePath := filepath.Join(path, ".vrooli", "service.json")
		if _, statErr := os.Stat(servicePath); errors.Is(statErr, os.ErrNotExist) {
			// Only log for immediate subdirectories of scenarios (depth 1)
			if filepath.Dir(path) == scenariosDir {
				log.Printf("DEBUG: scanScenarios - no service.json at %s", servicePath)
			}
			return nil
		}

		log.Printf("DEBUG: scanScenarios - found service.json at %s", servicePath)

		fileBytes, readErr := os.ReadFile(servicePath)
		if readErr != nil {
			return readErr
		}

		var payload serviceFile
		if err := json.Unmarshal(fileBytes, &payload); err != nil {
			return err
		}

		name := strings.TrimSpace(payload.Service.Name)
		if name == "" {
			name = filepath.Base(path)
		}
		display := strings.TrimSpace(payload.Service.DisplayName)
		if display == "" {
			display = name
		}
		description := strings.TrimSpace(payload.Service.Description)

		deps := make([]string, 0, len(payload.Dependencies.Scenarios))
		for dep := range payload.Dependencies.Scenarios {
			deps = append(deps, dep)
		}
		sort.Strings(deps)

		resources := make([]string, 0, len(payload.Dependencies.Resources))
		for res := range payload.Dependencies.Resources {
			resources = append(resources, res)
		}
		sort.Strings(resources)

		relPath, err := filepath.Rel(m.repoRoot, path)
		if err != nil {
			relPath = path
		}

		info, infoErr := os.Stat(servicePath)
		var modTime time.Time
		if infoErr == nil {
			modTime = info.ModTime().UTC()
		}

		entry := ScenarioCatalogEntry{
			Name:         name,
			DisplayName:  display,
			Description:  description,
			RelativePath: filepath.ToSlash(relPath),
			Tags:         append([]string{}, payload.Service.Tags...),
			Dependencies: deps,
			Resources:    resources,
			LastModified: modTime,
			Hidden:       m.isHidden(name),
		}

		entries[strings.ToLower(name)] = entry
		log.Printf("DEBUG: scanScenarios - added scenario: name=%s, displayName=%s, deps=%d, resources=%d",
			name, display, len(deps), len(resources))

		for _, dep := range deps {
			edges = append(edges, ScenarioDependencyEdge{
				From: name,
				To:   dep,
				Type: "requires",
			})
		}

		return filepath.SkipDir
	})

	if err != nil {
		log.Printf("ERROR: scanScenarios - WalkDir returned error: %v", err)
		return nil, nil, err
	}

	log.Printf("DEBUG: scanScenarios - complete: found %d scenarios, %d edges", len(entries), len(edges))
	return entries, edges, nil
}

func (m *ScenarioCatalogManager) isHidden(name string) bool {
	_, ok := m.hiddenSet[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func (m *ScenarioCatalogManager) Snapshot() ([]ScenarioCatalogEntry, []ScenarioDependencyEdge, []string, time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]ScenarioCatalogEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		list = append(list, entry)
	}
	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].DisplayName) < strings.ToLower(list[j].DisplayName)
	})

	// Ensure edges is never nil (JSON will encode nil as null, not [])
	edges := make([]ScenarioDependencyEdge, 0, len(m.edges))
	edges = append(edges, m.edges...)

	hidden := make([]string, 0, len(m.hiddenSet))
	for key := range m.hiddenSet {
		hidden = append(hidden, key)
	}
	sort.Strings(hidden)

	return list, edges, hidden, m.lastSynced
}

func (m *ScenarioCatalogManager) UpdateVisibility(scenario string, hidden bool) error {
	trimmed := strings.ToLower(strings.TrimSpace(scenario))
	if trimmed == "" {
		return errors.New("scenario name required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if hidden {
		m.hiddenSet[trimmed] = struct{}{}
	} else {
		delete(m.hiddenSet, trimmed)
	}

	if err := m.persistVisibilityConfigLocked(); err != nil {
		return err
	}

	if entry, ok := m.entries[trimmed]; ok {
		entry.Hidden = hidden
		m.entries[trimmed] = entry
	}

	return nil
}

func (m *ScenarioCatalogManager) StartBackgroundRefresh(interval time.Duration) {
	if interval < time.Hour {
		interval = 24 * time.Hour
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := m.Refresh(); err != nil {
				log.Printf("scenario catalog refresh failed: %v", err)
			}
		}
	}()
}

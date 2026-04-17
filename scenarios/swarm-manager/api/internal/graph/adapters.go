package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/initiatives"
)

// captureAdapter reads capture entries from the filesystem for graph projections.
type captureAdapter struct {
	rootDir string
}

// NewCaptureAdapter creates a CaptureLister that reads captures from the given root directory.
func NewCaptureAdapter(rootDir string) *captureAdapter {
	return &captureAdapter{rootDir: rootDir}
}

func (a *captureAdapter) ListCaptures() ([]CaptureEntry, error) {
	capturesRoot := filepath.Join(a.rootDir, "captures")
	entries, err := os.ReadDir(capturesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []CaptureEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		capPath := filepath.Join(capturesRoot, entry.Name(), "capture.json")
		data, err := os.ReadFile(capPath)
		if err != nil {
			continue
		}
		var raw struct {
			ID     string `json:"id"`
			Text   string `json:"text"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}

		ce := CaptureEntry{
			ID:     raw.ID,
			Text:   raw.Text,
			Status: raw.Status,
		}

		// Load classification if it exists.
		classPath := filepath.Join(capturesRoot, entry.Name(), "classification.json")
		classData, err := os.ReadFile(classPath)
		if err == nil {
			var cls struct {
				Items []struct {
					Kind  string `json:"kind"`
					Title string `json:"title"`
				} `json:"items"`
			}
			if json.Unmarshal(classData, &cls) == nil {
				for _, item := range cls.Items {
					ce.Items = append(ce.Items, CaptureClassificationItem{
						Kind:  item.Kind,
						Title: item.Title,
					})
				}
			}
		}

		result = append(result, ce)
	}
	return result, nil
}

// initiativeAdapter bridges the initiatives store to InitiativeLister.
type initiativeAdapter struct {
	store *initiatives.Store
}

// NewInitiativeAdapter creates an InitiativeLister backed by the given initiatives.Store.
func NewInitiativeAdapter(store *initiatives.Store) *initiativeAdapter {
	return &initiativeAdapter{store: store}
}

func (a *initiativeAdapter) List() ([]InitiativeEntry, error) {
	items, err := a.store.LoadAll()
	if err != nil {
		return nil, err
	}

	result := make([]InitiativeEntry, 0, len(items))
	for _, item := range items {
		result = append(result, InitiativeEntry{
			Name:       item.Name,
			Title:      item.Title,
			Status:     item.Status,
			Items:      append([]string(nil), item.Items...),
			ArchivedAt: item.ArchivedAt,
		})
	}
	return result, nil
}

// executionAdapter bridges execution.Service to ExecutionLister.
type executionAdapter struct {
	svc *execution.Service
}

// NewExecutionAdapter creates an ExecutionLister backed by the given execution.Service.
func NewExecutionAdapter(svc *execution.Service) *executionAdapter {
	return &executionAdapter{svc: svc}
}

func (a *executionAdapter) List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error) {
	return a.svc.List(ctx, filters)
}

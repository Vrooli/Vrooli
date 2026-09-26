package autofiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/storage"
)

type Dismissal struct {
	FindingID string `json:"finding_id"`
	ItemRef   string `json:"item_ref,omitempty"`
	Reason    string `json:"reason,omitempty"`
	At        string `json:"at"`
}

type DismissalStore struct {
	path string
}

func NewDismissalStore(dataRoot string) *DismissalStore {
	return &DismissalStore{path: filepath.Join(dataRoot, "autofiler", "dismissed_findings.json")}
}

func NewDismissalStorePath(path string) *DismissalStore {
	return &DismissalStore{path: path}
}

func (s *DismissalStore) IsDismissed(findingID string) (bool, error) {
	dismissals, err := s.LoadAll()
	if err != nil {
		return false, err
	}
	_, ok := dismissals[strings.TrimSpace(findingID)]
	return ok, nil
}

func (s *DismissalStore) Remember(findingID, itemRef, reason string) error {
	findingID = strings.TrimSpace(findingID)
	if findingID == "" {
		return fmt.Errorf("finding ID is required")
	}
	dismissals, err := s.LoadAll()
	if err != nil {
		return err
	}
	if existing, ok := dismissals[findingID]; ok {
		if strings.TrimSpace(existing.ItemRef) == "" {
			existing.ItemRef = strings.TrimSpace(itemRef)
		}
		if strings.TrimSpace(existing.Reason) == "" {
			existing.Reason = strings.TrimSpace(reason)
		}
		dismissals[findingID] = existing
		return s.saveAll(dismissals)
	}
	dismissals[findingID] = Dismissal{
		FindingID: findingID,
		ItemRef:   strings.TrimSpace(itemRef),
		Reason:    strings.TrimSpace(reason),
		At:        time.Now().UTC().Format(time.RFC3339),
	}
	return s.saveAll(dismissals)
}

func (s *DismissalStore) LoadAll() (map[string]Dismissal, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Dismissal{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]Dismissal{}, nil
	}
	var dismissals map[string]Dismissal
	if err := json.Unmarshal(data, &dismissals); err != nil {
		return nil, fmt.Errorf("load auto-filer dismissals: %w", err)
	}
	if dismissals == nil {
		dismissals = map[string]Dismissal{}
	}
	return dismissals, nil
}

func (s *DismissalStore) Count() (int, error) {
	dismissals, err := s.LoadAll()
	if err != nil {
		return 0, err
	}
	return len(dismissals), nil
}

func (s *DismissalStore) List() ([]Dismissal, error) {
	dismissals, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	out := make([]Dismissal, 0, len(dismissals))
	for _, dismissal := range dismissals {
		out = append(out, dismissal)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].At != out[j].At {
			return out[i].At < out[j].At
		}
		return out[i].FindingID < out[j].FindingID
	})
	return out, nil
}

func (s *DismissalStore) saveAll(dismissals map[string]Dismissal) error {
	return storage.WriteJSONAtomic(s.path, dismissals)
}

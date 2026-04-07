package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/storage"
)

// backlogItem represents a backlog entry loaded from spec.json.
type backlogItem struct {
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Status             string   `json:"status"`
	Priority           int      `json:"priority"`
	Tags               []string `json:"tags"`
	Created            string   `json:"created"`
	Updated            string   `json:"updated"`
	Kind               string   `json:"kind"`
	SourceScenarioName string   `json:"sourceScenarioName,omitempty"`
	AcceptanceAllow    []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny     []string `json:"acceptance_deny,omitempty"`
	ArchivedAt         *string  `json:"archived_at,omitempty"`
}

func (s *Service) loadBacklogItem(kind, name string) (backlogItem, error) {
	specPath := filepath.Join(s.itemDir(kind, name), "spec.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return backlogItem{}, err
	}
	var item backlogItem
	if err := json.Unmarshal(data, &item); err != nil {
		return backlogItem{}, err
	}
	item.Name = strings.TrimSpace(name)
	item.Kind = strings.ToLower(strings.TrimSpace(kind))
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, nil
}

func (s *Service) loadBacklogItemByRecord(record *Record) (backlogItem, error) {
	return s.loadBacklogItem(record.BacklogKind, record.BacklogName)
}

func (s *Service) updateBacklogStatus(item backlogItem, status string) error {
	item.Status = status
	item.Updated = nowRFC3339()
	specPath := filepath.Join(s.itemDir(item.Kind, item.Name), "spec.json")
	merged := map[string]any{}
	if existing, err := os.ReadFile(specPath); err == nil {
		_ = json.Unmarshal(existing, &merged)
	} else if !os.IsNotExist(err) {
		return err
	}

	merged["name"] = item.Name
	merged["title"] = item.Title
	merged["description"] = item.Description
	merged["status"] = item.Status
	merged["priority"] = item.Priority
	merged["tags"] = item.Tags
	merged["created"] = item.Created
	merged["updated"] = item.Updated
	merged["kind"] = item.Kind
	delete(merged, "research_target")
	return storage.WriteJSONAtomic(specPath, merged)
}

func (s *Service) restoreBacklogStatusForRecord(record Record) error {
	item, err := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if err != nil {
		return apierr.Conflict("execution canceled but backlog status restore failed; fix the backlog item status and retry")
	}
	if err := s.updateBacklogStatus(item, restoreBacklogStatus(record)); err != nil {
		return apierr.Conflict("execution canceled but backlog status restore failed; fix the backlog item status and retry")
	}
	return nil
}

func restoreBacklogStatus(record Record) string {
	previous := strings.ToLower(strings.TrimSpace(record.PreviousStatus))
	switch previous {
	case backlogStatusBacklog, backlogStatusResearching, backlogStatusReady:
		return previous
	}
	return backlogStatusReady
}

func (s *Service) kindDir(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "idea":
		return filepath.Join(s.rootDir, "ideas")
	case "research":
		return filepath.Join(s.rootDir, "research")
	case "fix":
		return filepath.Join(s.rootDir, "fix")
	case "execute":
		return filepath.Join(s.rootDir, "execute")
	case "chore":
		return filepath.Join(s.rootDir, "chore")
	default:
		return filepath.Join(s.rootDir, "ideas")
	}
}

func (s *Service) itemDir(kind, name string) string {
	return filepath.Join(s.kindDir(kind), strings.TrimSpace(name))
}

func (s *Service) scenariosRootDir() string {
	return filepath.Dir(s.rootDir)
}

// isQueueableStatus checks whether the item's status allows queuing.
// Archived items (identified by ArchivedAt) are handled separately by the caller.
func isQueueableStatus(kind, status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case backlogStatusBacklog, backlogStatusResearching, backlogStatusReady:
		return true
	default:
		return false
	}
}

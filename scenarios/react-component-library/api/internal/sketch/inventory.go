package sketch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DesignScenario struct {
	Scenario  string
	PageCount int
	Issue     string
}
type DesignPage struct {
	Routes                                         []string
	Page, Title, Route, Status, ContentHash, Issue string
	RegionCount                                    int
	Registered                                     bool
}
type DesignInventory struct {
	Scenarios []DesignScenario
	Pages     []DesignPage
	Issues    []string
}
type WorkspaceReader interface {
	ListDesignPages(context.Context, string) (DesignInventory, error)
}

// ListDesignPages includes registered missing pages and unregistered files. A
// broken contract must remain visible rather than silently shrinking the fleet.
func (s *Store) ListDesignPages(ctx context.Context, scenario string) (DesignInventory, error) {
	out := DesignInventory{}
	if scenario != "" {
		if !safeSegment.MatchString(scenario) {
			return out, fmt.Errorf("invalid scenario identity")
		}
		return s.pageInventory(ctx, scenario)
	}
	entries, err := os.ReadDir(filepath.Join(s.repoRoot, "scenarios"))
	if err != nil {
		return out, err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if !entry.IsDir() || !safeSegment.MatchString(entry.Name()) {
			continue
		}
		item := DesignScenario{Scenario: entry.Name()}
		root, err := s.openExperience(entry.Name())
		if err != nil {
			if !os.IsNotExist(err) {
				item.Issue = err.Error()
			}
			out.Scenarios = append(out.Scenarios, item)
			continue
		}
		dir, err := root.Open("pages")
		if err == nil {
			pages, readErr := dir.ReadDir(-1)
			dir.Close()
			err = readErr
			for _, page := range pages {
				if strings.HasSuffix(page.Name(), ".json") && !page.IsDir() {
					item.PageCount++
				}
			}
		}
		root.Close()
		if err != nil && !os.IsNotExist(err) {
			item.Issue = err.Error()
		}
		out.Scenarios = append(out.Scenarios, item)
	}
	return out, nil
}

func (s *Store) pageInventory(ctx context.Context, scenario string) (DesignInventory, error) {
	out := DesignInventory{}
	root, err := s.openExperience(scenario)
	if err != nil {
		return out, err
	}
	defer root.Close()
	pages := map[string]*DesignPage{}
	indexBytes, err := root.ReadFile("index.json")
	if err != nil {
		out.Issues = append(out.Issues, "Experience index: "+err.Error())
	} else {
		var index struct {
			Pages []struct{ ID, Path, Title string }
		}
		if err := json.Unmarshal(indexBytes, &index); err != nil {
			out.Issues = append(out.Issues, "Experience index: "+err.Error())
		} else {
			for _, entry := range index.Pages {
				if !safeSegment.MatchString(entry.ID) {
					out.Issues = append(out.Issues, "Invalid page identity in index: "+entry.ID)
					continue
				}
				if old := pages[entry.ID]; old != nil {
					old.Issue = "Duplicate page registration"
					continue
				}
				item := &DesignPage{Page: entry.ID, Title: entry.Title, Registered: true, Status: "missing"}
				if entry.Path != "pages/"+entry.ID+".json" {
					item.Issue = "Registered path differs from canonical page path"
				}
				pages[entry.ID] = item
			}
		}
	}
	dir, err := root.Open("pages")
	if err != nil {
		out.Issues = append(out.Issues, "Page directory: "+err.Error())
	} else {
		entries, readErr := dir.ReadDir(-1)
		dir.Close()
		if readErr != nil {
			return out, readErr
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				id := strings.TrimSuffix(entry.Name(), ".json")
				if !safeSegment.MatchString(id) {
					out.Issues = append(out.Issues, "Invalid page filename: "+entry.Name())
					continue
				}
				if pages[id] == nil {
					pages[id] = &DesignPage{Page: id, Title: id}
				}
			}
		}
	}
	for id, item := range pages {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		pageRoot, name, err := s.openPages(scenario, id)
		if err != nil {
			item.Status = "missing"
			item.Issue = joinIssue(item.Issue, err.Error())
			out.Pages = append(out.Pages, *item)
			continue
		}
		raw, err := pageRoot.ReadFile(name)
		pageRoot.Close()
		if err != nil {
			item.Status = "unresolved"
			item.Issue = joinIssue(item.Issue, err.Error())
			out.Pages = append(out.Pages, *item)
			continue
		}
		snapshot, err := decodeSnapshot(raw)
		if err != nil {
			item.Status = "invalid"
			item.Issue = joinIssue(item.Issue, err.Error())
			out.Pages = append(out.Pages, *item)
			continue
		}
		var doc struct {
			Page struct {
				Title, Route string
				Routes       []string
			}
			Regions []struct{ ID string }
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			item.Status = "invalid"
			item.Issue = joinIssue(item.Issue, err.Error())
			out.Pages = append(out.Pages, *item)
			continue
		}
		if doc.Page.Title != "" {
			item.Title = doc.Page.Title
		}
		item.Route = doc.Page.Route
		item.Routes = doc.Page.Routes
		if len(item.Routes) > 0 {
			item.Route = item.Routes[0]
		}
		item.ContentHash = snapshot.ContentHash
		regions := map[string]bool{}
		for _, region := range doc.Regions {
			regions[region.ID] = true
		}
		for _, region := range snapshot.Document.Regions {
			regions[region.ID] = true
		}
		delete(regions, "")
		item.RegionCount = len(regions)
		item.Status = "contract-only"
		if item.RegionCount > 0 {
			item.Status = "region-declared"
		}
		out.Pages = append(out.Pages, *item)
	}
	sort.Slice(out.Pages, func(i, j int) bool { return out.Pages[i].Page < out.Pages[j].Page })
	return out, nil
}
func joinIssue(before, next string) string {
	if before == "" {
		return next
	}
	return before + "; " + next
}

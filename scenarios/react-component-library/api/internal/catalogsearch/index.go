// Package catalogsearch builds the searchable projection of authored catalog declarations.
package catalogsearch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var layerOrder = []string{"page-template", "pattern", "navigation", "component", "primitive", "foundation", "runtime-hook", "runtime-service", "adapter", "generator", "fixture"}

type Document struct {
	CatalogID, Name, Kind, Slot, Domain, Surface, Description, DeclarationPath string
	RequiredCapabilities, Regions []string
	Implemented bool
	Layer int
}

type Result struct { Document; Score float64 }

type Index struct {
	mu sync.RWMutex
	docs []Document
	indexedAt time.Time
}

func New() *Index { return &Index{} }

func (i *Index) Reindex(scenarioRoot string) error {
	paths, err := filepath.Glob(filepath.Join(scenarioRoot, "catalog", "assets", "*", "*.json")); if err != nil { return err }
	sort.Strings(paths)
	docs := make([]Document, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path); if err != nil { return fmt.Errorf("read %s: %w", path, err) }
		var declaration struct {
			Kind string `json:"kind"`
			Asset struct { ID, Name, Kind, Slot, Domain, Surface, Description string } `json:"asset"`
			RequiredCapabilities []string `json:"requiredCapabilities"`
			Regions []struct{ ID string `json:"id"` } `json:"regions"`
		}
		if err := json.Unmarshal(raw, &declaration); err != nil { return fmt.Errorf("parse %s: %w", path, err) }
		if declaration.Kind != "catalog-asset" { continue }
		doc := Document{CatalogID:declaration.Asset.ID, Name:declaration.Asset.Name, Kind:declaration.Asset.Kind, Slot:declaration.Asset.Slot, Domain:declaration.Asset.Domain, Surface:declaration.Asset.Surface, Description:declaration.Asset.Description, DeclarationPath:path, RequiredCapabilities:declaration.RequiredCapabilities, Layer:LayerRank(declaration.Asset.Kind)}
		for _, region := range declaration.Regions { doc.Regions = append(doc.Regions, region.ID) }
		doc.Implemented = implementationExists(scenarioRoot, doc.CatalogID)
		docs = append(docs, doc)
	}
	i.mu.Lock(); i.docs = docs; i.indexedAt = time.Now().UTC(); i.mu.Unlock()
	return nil
}

func LayerRank(kind string) int { for index, value := range layerOrder { if value == kind { return index } }; return len(layerOrder) }

func (i *Index) Search(query string, limit int, kind, domain, accepts string) []Result {
	i.mu.RLock(); defer i.mu.RUnlock()
	terms := tokenize(query); results := make([]Result, 0)
	for _, doc := range i.docs {
		if kind != "" && doc.Kind != kind || domain != "" && doc.Domain != domain || accepts != "" && doc.Domain != accepts && doc.Kind != accepts { continue }
		haystack := strings.ToLower(strings.Join(append([]string{doc.CatalogID, doc.Name, doc.Kind, doc.Domain, doc.Surface, doc.Description}, append(doc.RequiredCapabilities, doc.Regions...)...), " "))
		score := 0.0
		for _, term := range terms { if strings.Contains(haystack, term) { score += 1 } }
		if score == 0 { continue }
		results = append(results, Result{Document:doc, Score:score/float64(len(terms))})
	}
	sort.SliceStable(results, func(a,b int) bool { if results[a].Layer != results[b].Layer { return results[a].Layer < results[b].Layer }; if results[a].Score != results[b].Score { return results[a].Score > results[b].Score }; return results[a].CatalogID < results[b].CatalogID })
	if limit <= 0 { limit = 10 }; if len(results) > limit { results = results[:limit] }
	return results
}

func (i *Index) Status() (int, time.Time) { i.mu.RLock(); defer i.mu.RUnlock(); return len(i.docs), i.indexedAt }
func tokenize(value string) []string { return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r < 'a' || r > 'z' }) }

func implementationExists(root, catalogID string) bool {
	patterns := []string{filepath.Join(root,"library","*","*","manifest.json"), filepath.Join(root,"library","*","*","versions","*","manifest.json")}
	for _, pattern := range patterns { paths,_ := filepath.Glob(pattern); for _, path := range paths { raw,err:=os.ReadFile(path); if err==nil && strings.Contains(string(raw), `"catalogId": "`+catalogID+`"`) { return true } } }
	return false
}

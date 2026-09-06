// Package catalogsearch builds the searchable projection of authored catalog declarations.
package catalogsearch

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"react-component-library/internal/availability"
	"react-component-library/internal/catalogcoverage"
)

var layerOrder = []string{"page-template", "pattern", "navigation", "component", "primitive", "foundation", "runtime-hook", "runtime-service", "adapter", "generator", "fixture"}

type Document struct {
	CatalogID, Name, Kind, Slot, Domain, Surface, Description, DeclarationPath string
	RequiredCapabilities, Regions                                              []string
	Implemented                                                                bool
	Layer                                                                      int
}

type Result struct {
	Document
	Availability availability.Result
	Score        float64
}

type Index struct {
	refreshMu           sync.Mutex
	mu                  sync.RWMutex
	docs                []Document
	indexedAt           time.Time
	lastError           string
	availabilityFactory func(context.Context) (*availability.Snapshot, error)
	availability        *availability.Snapshot
}

func New() *Index { return &Index{} }
func NewWithAvailability(factory func(context.Context) (*availability.Snapshot, error)) *Index {
	return &Index{availabilityFactory: factory}
}

func (i *Index) Reindex(scenarioRoot string) error {
	return i.ReindexContext(context.Background(), scenarioRoot)
}

// ReindexContext publishes only a complete projection. A failed refresh retains
// the prior watermark and results, with its failure exposed through Diagnostics.
func (i *Index) ReindexContext(ctx context.Context, scenarioRoot string) (err error) {
	i.refreshMu.Lock()
	defer i.refreshMu.Unlock()
	defer func() {
		if err != nil {
			i.mu.Lock()
			i.lastError = err.Error()
			i.mu.Unlock()
		}
	}()
	assets, err := catalogcoverage.LoadCatalogContext(ctx, filepath.Join(scenarioRoot, "catalog"))
	if err != nil {
		return err
	}
	var snapshot *availability.Snapshot
	if i.availabilityFactory != nil {
		snapshot, err = i.availabilityFactory(ctx)
		if err != nil {
			return err
		}
	}
	docs := make([]Document, 0, len(assets))
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return err
		}
		docs = append(docs, Document{CatalogID: asset.ID, Name: asset.Name, Kind: asset.Kind,
			Slot: asset.Slot, Domain: asset.Domain, Surface: asset.Surface, Description: asset.Description,
			DeclarationPath: asset.DeclarationPath, RequiredCapabilities: asset.Capabilities,
			Regions: asset.Regions, Layer: LayerRank(asset.Kind)})
	}
	i.mu.Lock()
	i.docs = docs
	i.availability = snapshot
	i.indexedAt = time.Now().UTC()
	i.lastError = ""
	i.mu.Unlock()
	return nil
}

type Diagnostics struct {
	Count     int
	IndexedAt time.Time
	Stale     bool
	LastError string
}

func (i *Index) Diagnostics() Diagnostics {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return Diagnostics{Count: len(i.docs), IndexedAt: i.indexedAt, Stale: i.lastError != "", LastError: i.lastError}
}

func LayerRank(kind string) int {
	for index, value := range layerOrder {
		if value == kind {
			return index
		}
	}
	return len(layerOrder)
}

func (i *Index) Search(query string, limit int, kind, domain, accepts string) []Result {
	return i.SearchContext(context.Background(), query, limit, kind, domain, accepts)
}
func (i *Index) SearchContext(ctx context.Context, query string, limit int, kind, domain, accepts string) []Result {
	i.mu.RLock()
	docs := append([]Document(nil), i.docs...)
	snapshot := i.availability
	i.mu.RUnlock()
	terms := tokenize(query)
	results := make([]Result, 0)
	for _, doc := range docs {
		if kind != "" && doc.Kind != kind || domain != "" && doc.Domain != domain || accepts != "" && doc.Domain != accepts && doc.Kind != accepts {
			continue
		}
		haystack := strings.ToLower(strings.Join(append([]string{doc.CatalogID, doc.Name, doc.Kind, doc.Domain, doc.Surface, doc.Description}, append(append([]string(nil), doc.RequiredCapabilities...), doc.Regions...)...), " "))
		score := 0.0
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				score += 1
			}
		}
		if score == 0 {
			continue
		}
		results = append(results, Result{Document: doc, Score: score / float64(len(terms))})
	}
	sort.SliceStable(results, func(a, b int) bool {
		if results[a].Score != results[b].Score {
			return results[a].Score > results[b].Score
		}
		if results[a].Layer != results[b].Layer {
			return results[a].Layer < results[b].Layer
		}
		return results[a].CatalogID < results[b].CatalogID
	})
	if limit <= 0 {
		limit = 10
	}
	if len(results) > limit {
		results = results[:limit]
	}
	for j := range results {
		result := availability.Result{CatalogID: results[j].CatalogID, State: availability.Unavailable, ReasonCode: "availability_unconfigured", Reason: "canonical implementation resolver is unavailable"}
		if snapshot != nil {
			result = snapshot.Resolve(ctx, results[j].CatalogID, snapshot.Latest(results[j].CatalogID))
		}
		results[j].Availability = result
		results[j].Implemented = result.IsBuilt()
	}
	return results
}

func (i *Index) Status() (int, time.Time) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return len(i.docs), i.indexedAt
}
func tokenize(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}

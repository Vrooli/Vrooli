package modules

import (
	"context"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/docaccess"
	"knowledge-observatory/internal/graph"
	"knowledge-observatory/internal/ingest"
	"knowledge-observatory/internal/metadata"
	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/quality"
	"knowledge-observatory/internal/search"
)

// Stores holds one repository per storage domain.
//
// Domains stay independent of each other: nothing in internal/<domain>/ imports
// a sibling. This package is the only place that knows the full set, which is
// why the cross-domain composition below lives here rather than inside a domain.
type Stores struct {
	Quality   quality.Repository
	Search    search.Repository
	Ingest    ingest.Repository
	Metadata  metadata.Repository
	Graph     graph.Repository
	DocAccess docaccess.Repository
}

// NewSQLiteStores builds every domain repository against one SQLite handle.
func NewSQLiteStores(db *apidb.RoutedDB) *Stores {
	return &Stores{
		Quality:   quality.NewSQLite(db),
		Search:    search.NewSQLite(db),
		Ingest:    ingest.NewSQLite(db),
		Metadata:  metadata.NewSQLite(db),
		Graph:     graph.NewSQLite(db),
		DocAccess: docaccess.NewSQLite(db),
	}
}

// MetadataStore adapts the per-domain repositories to ports.MetadataStore.
//
// ports.MetadataStore predates the domain split and spans five domains
// (metadata, ingest, search, quality, graph). Rather than give one domain
// ownership of another's tables, this fan-out keeps the port's callers
// unchanged while each write lands in the domain that owns the table.
type MetadataStore struct {
	stores *Stores
}

var _ ports.MetadataStore = (*MetadataStore)(nil)

// NewMetadataStore returns the composite view of stores.
func NewMetadataStore(stores *Stores) *MetadataStore {
	if stores == nil {
		return nil
	}
	return &MetadataStore{stores: stores}
}

func (m *MetadataStore) UpsertKnowledgeMetadata(ctx context.Context, vectorID, collectionName, contentHash, sourceScenario, sourceType string) error {
	if m == nil || m.stores == nil {
		return nil
	}
	return m.stores.Metadata.UpsertEntry(ctx, metadata.Entry{
		VectorID:       vectorID,
		CollectionName: collectionName,
		ContentHash:    contentHash,
		SourceScenario: sourceScenario,
		SourceType:     sourceType,
	})
}

func (m *MetadataStore) InsertIngestHistory(ctx context.Context, row ports.IngestHistoryRow) error {
	if m == nil || m.stores == nil {
		return nil
	}
	_, err := m.stores.Ingest.InsertHistory(ctx, ingest.HistoryEntry{
		RecordID:       row.RecordID,
		Namespace:      row.Namespace,
		CollectionName: row.Collection,
		ContentHash:    row.ContentHash,
		Visibility:     row.Visibility,
		Source:         row.Source,
		SourceType:     row.SourceType,
		Status:         row.Status,
		ErrorMessage:   row.ErrorMessage,
		TookMS:         row.TookMS,
	})
	return err
}

func (m *MetadataStore) InsertSearchHistory(ctx context.Context, row ports.SearchHistoryRow) error {
	if m == nil || m.stores == nil {
		return nil
	}
	_, err := m.stores.Search.InsertHistory(ctx, search.History{
		Query:          row.Query,
		Collection:     row.Collection,
		ResultCount:    row.ResultCount,
		AvgScore:       row.AvgScore,
		ResponseTimeMS: row.ResponseTimeMS,
		UserSession:    row.UserSession,
	})
	return err
}

func (m *MetadataStore) LookupCollectionForVectorID(ctx context.Context, vectorID string) (string, bool, error) {
	if m == nil || m.stores == nil {
		return "", false, nil
	}
	return m.stores.Metadata.LookupCollectionForVectorID(ctx, vectorID)
}

func (m *MetadataStore) UpsertExternalIDMapping(ctx context.Context, mapping ports.ExternalIDMapping) error {
	if m == nil || m.stores == nil {
		return nil
	}
	return m.stores.Metadata.UpsertExternalIDMapping(ctx, metadata.ExternalIDMapping{
		Namespace:   mapping.Namespace,
		ExternalID:  mapping.ExternalID,
		Kind:        mapping.Kind,
		RecordID:    mapping.RecordID,
		DocumentID:  mapping.DocumentID,
		ContentHash: mapping.ContentHash,
	})
}

func (m *MetadataStore) LookupExternalIDMapping(ctx context.Context, namespace, externalID, kind string) (ports.ExternalIDMapping, bool, error) {
	if m == nil || m.stores == nil {
		return ports.ExternalIDMapping{}, false, nil
	}
	found, ok, err := m.stores.Metadata.LookupExternalIDMapping(ctx, namespace, externalID, kind)
	if err != nil || !ok {
		return ports.ExternalIDMapping{}, ok, err
	}
	return ports.ExternalIDMapping{
		Namespace:   found.Namespace,
		ExternalID:  found.ExternalID,
		Kind:        found.Kind,
		RecordID:    found.RecordID,
		DocumentID:  found.DocumentID,
		ContentHash: found.ContentHash,
	}, true, nil
}

func (m *MetadataStore) UpsertQualityMetrics(ctx context.Context, row ports.QualityMetricsRow) error {
	if m == nil || m.stores == nil {
		return nil
	}
	_, err := m.stores.Quality.InsertMetric(ctx, quality.Metric{
		CollectionName: row.CollectionName,
		Coherence:      row.Coherence,
		Freshness:      row.Freshness,
		Redundancy:     row.Redundancy,
		Coverage:       row.Coverage,
		TotalEntries:   row.TotalEntries,
	})
	return err
}

func (m *MetadataStore) UpsertCollectionStats(ctx context.Context, row ports.CollectionStatsRow) error {
	if m == nil || m.stores == nil {
		return nil
	}
	return m.stores.Quality.UpsertCollectionStat(ctx, quality.CollectionStat{
		CollectionName: row.CollectionName,
		TotalEntries:   row.TotalEntries,
	})
}

func (m *MetadataStore) UpsertRelationshipEdges(ctx context.Context, edges []ports.RelationshipEdgeRow) error {
	if m == nil || m.stores == nil {
		return nil
	}
	converted := make([]graph.Edge, 0, len(edges))
	for _, e := range edges {
		converted = append(converted, graph.Edge{
			SourceID:         e.SourceID,
			TargetID:         e.TargetID,
			RelationshipType: e.RelationshipType,
			Weight:           e.Weight,
		})
	}
	return m.stores.Graph.UpsertEdges(ctx, converted)
}

// DocAccessLogger adapts the docaccess repository to ports.DocAccessLogger.
type DocAccessLogger struct {
	repo docaccess.Repository
}

var _ ports.DocAccessLogger = (*DocAccessLogger)(nil)

// NewDocAccessLogger returns the port view of the docaccess domain.
func NewDocAccessLogger(repo docaccess.Repository) *DocAccessLogger {
	if repo == nil {
		return nil
	}
	return &DocAccessLogger{repo: repo}
}

func (d *DocAccessLogger) LogAccess(ctx context.Context, row ports.DocAccessRow) error {
	if d == nil || d.repo == nil {
		return nil
	}
	return d.repo.LogAccess(ctx, docaccess.Access{
		ScenarioName: row.ScenarioName,
		DocType:      row.DocType,
		Operation:    row.Operation,
	})
}

func (d *DocAccessLogger) QueryStats(ctx context.Context, filter ports.DocAccessFilter) ([]ports.DocAccessStat, error) {
	if d == nil || d.repo == nil {
		return nil, nil
	}
	stats, err := d.repo.QueryStats(ctx, docaccess.Filter{
		ScenarioName: filter.ScenarioName,
		DocType:      filter.DocType,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ports.DocAccessStat, 0, len(stats))
	for _, s := range stats {
		out = append(out, ports.DocAccessStat{
			ScenarioName: s.ScenarioName,
			DocType:      s.DocType,
			ReadCount:    s.ReadCount,
			WriteCount:   s.WriteCount,
			ResetCount:   s.ResetCount,
		})
	}
	return out, nil
}

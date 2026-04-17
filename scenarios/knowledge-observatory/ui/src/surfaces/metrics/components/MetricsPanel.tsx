// DOC: docs/concepts/ARCHITECTURE.md#health-and-metrics-flow
import { Database, TrendingUp, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { cn } from "../../../shared/lib/utils";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";
import type { MetricCardView, MetricsViewModel } from "../../../shared/controllers/knowledgeController";
import type {
  CollectionDiagnostics,
  CollectionInventoryItem,
  CollectionMaintenanceResponse,
  CollectionRecordsResponse,
  DocumentDeleteResponse,
  IngestHealthResponse,
} from "../../../shared/services/api";

// AI_CHECK: REACT_STABILITY=1 | LAST: 2026-01-25

const toneStyles: Record<MetricCardView["tone"], string> = {
  good: "ko-tone-good",
  medium: "ko-tone-medium",
  poor: "ko-tone-poor",
};

const EMPTY_METRIC_CARDS: MetricCardView[] = [];
const EMPTY_COLLECTIONS: MetricsViewModel["collections"] = [];
const DEFAULT_VIEW_MODEL: MetricsViewModel = {
  metricCards: EMPTY_METRIC_CARDS,
  collections: EMPTY_COLLECTIONS,
  overallHealth: "unknown",
  lastUpdated: "Unknown",
  totalEntriesLabel: "Unknown",
  hasMetrics: false,
};

type MaintenanceAction = "prune-stale-chunks" | "dedupe-content";
type CollectionDrilldownTab = "integrity" | "chunking" | "failures" | "records" | "maintenance";
type OwnershipTone = "good" | "medium" | "poor";

type DocumentOption = {
  key: string;
  namespace: string;
  documentID: string;
  label: string;
  count: number;
};

function MetricCard({ label, percentageLabel, description, tone }: MetricCardView) {
  const styles = toneStyles[tone] ?? toneStyles.medium;

  return (
    <div className={cn("ko-metric-card", styles)}>
      <div className="flex items-start justify-between mb-2">
        <span className="ko-meta">{label}</span>
        <TrendingUp className="h-4 w-4" />
      </div>
      <div className="text-3xl font-bold">{percentageLabel}</div>
      <p className="ko-text-xs ko-subtle mt-2">{description}</p>
    </div>
  );
}

function inferCollectionOwnership(
  collectionName: string,
  selectedCollection: string,
  diagnostics: CollectionDiagnostics | null | undefined
): { label: string; tone: OwnershipTone; reason: string } {
  if (collectionName === "knowledge_chunks_v1") {
    return {
      label: "Likely KO-managed",
      tone: "good",
      reason: "Default collection used by knowledge-observatory ingest.",
    };
  }

  if (collectionName === selectedCollection && diagnostics) {
    const missingFields = diagnostics.missing_payload_fields ?? {};
    const missingCore =
      (missingFields.namespace ?? 0) +
      (missingFields.document_id ?? 0) +
      (missingFields.chunk_index ?? 0) +
      (missingFields.content_hash ?? 0);
    const hasIngestHistory = (diagnostics.ingest_history?.total_attempts ?? 0) > 0;
    if (hasIngestHistory || missingCore === 0) {
      return {
        label: "Likely KO-managed",
        tone: "good",
        reason: "Diagnostics indicate KO payload shape and/or ingest history.",
      };
    }
    if (missingCore > 0) {
      return {
        label: "Likely external/mixed",
        tone: "poor",
        reason: "Core KO payload fields are missing for some points.",
      };
    }
  }

  return {
    label: "Unknown ownership",
    tone: "medium",
    reason: "Inspect diagnostics to classify this collection.",
  };
}

function resolveCollectionOwnership(
  collectionName: string,
  selectedCollection: string,
  diagnostics: CollectionDiagnostics | null | undefined,
  inventory: CollectionInventoryItem | null
): { label: string; tone: OwnershipTone; reason: string } {
  if (inventory) {
    const tone: OwnershipTone =
      inventory.ownership === "knowledge_observatory"
        ? "good"
        : inventory.ownership === "mixed"
          ? "poor"
          : "medium";
    const provenanceBits = [
      `ingest attempts: ${inventory.ingest_attempts}`,
      `metadata rows: ${inventory.metadata_rows}`,
      `namespaces: ${inventory.distinct_namespaces}`,
    ];
    return {
      label: inventory.ownership_label || "Unknown ownership",
      tone,
      reason: provenanceBits.join(" · "),
    };
  }
  return inferCollectionOwnership(collectionName, selectedCollection, diagnostics);
}

function ownershipToneClass(tone: OwnershipTone): string {
  if (tone === "good") return "ko-tone-good";
  if (tone === "poor") return "ko-tone-poor";
  return "ko-tone-medium";
}

export type MetricsPanelProps = {
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  hasData: boolean;
  viewModel: MetricsViewModel;
  ingestHealth?: IngestHealthResponse | null;
  selectedCollection: string;
  diagnostics?: CollectionDiagnostics | null;
  diagnosticsError: string;
  diagnosticsLoading: boolean;
  diagnosticsMode: "sample" | "full";
  diagnosticsLimit: number;
  drilldownTab: CollectionDrilldownTab;
  maintenanceInFlight: boolean;
  maintenanceNotice: string;
  maintenanceMaxDeletes: number;
  getMaintenancePreview: (collection: string, action: MaintenanceAction) => CollectionMaintenanceResponse | null;
  getCollectionInventory: (collection: string) => CollectionInventoryItem | null;
  documentOptions: DocumentOption[];
  selectedDocumentKey: string;
  documentDeletePreview?: DocumentDeleteResponse | null;
  collectionRecords: CollectionRecordsResponse | null;
  recordsLoading: boolean;
  recordsError: string;
  recordsSearch: string;
  recordsNamespaceFilter: string;
  recordsDocumentFilter: string;
  onSelectCollection: (name: string) => void;
  onDrilldownTabChange: (tab: CollectionDrilldownTab) => void;
  onSelectedDocumentKeyChange: (key: string) => void;
  onUseSampleDiagnostics: () => void;
  onUseFullDiagnostics: () => void;
  onMaintenanceMaxDeletesChange: (value: number) => void;
  onRecordsSearchChange: (value: string) => void;
  onRecordsNamespaceFilterChange: (value: string) => void;
  onRecordsDocumentFilterChange: (value: string) => void;
  onRecordsNextPage: () => void;
  onRecordsPreviousPage: () => void;
  onPreviewMaintenance: (collection: string, action: MaintenanceAction) => void;
  onApplyMaintenance: (collection: string, action: MaintenanceAction) => void;
  onPreviewDeleteDocument: () => void;
  onApplyDeleteDocument: () => void;
  collectionDeleteInFlight: boolean;
  onDeleteCollection: (collection: string) => void;
  onRetry: () => void;
};

export function MetricsPanel({
  isLoading,
  hasError,
  errorMessage,
  hasData,
  viewModel,
  ingestHealth,
  selectedCollection,
  diagnostics,
  diagnosticsError: _diagnosticsError,
  diagnosticsLoading: _diagnosticsLoading,
  diagnosticsMode: _diagnosticsMode,
  diagnosticsLimit: _diagnosticsLimit,
  drilldownTab: _drilldownTab,
  maintenanceInFlight: _maintenanceInFlight,
  maintenanceNotice,
  maintenanceMaxDeletes: _maintenanceMaxDeletes,
  getMaintenancePreview: _getMaintenancePreview,
  getCollectionInventory,
  documentOptions: _documentOptions,
  selectedDocumentKey: _selectedDocumentKey,
  documentDeletePreview: _documentDeletePreview,
  collectionRecords: _collectionRecords,
  recordsLoading: _recordsLoading,
  recordsError: _recordsError,
  recordsSearch: _recordsSearch,
  recordsNamespaceFilter: _recordsNamespaceFilter,
  recordsDocumentFilter: _recordsDocumentFilter,
  onSelectCollection,
  onDrilldownTabChange: _onDrilldownTabChange,
  onSelectedDocumentKeyChange: _onSelectedDocumentKeyChange,
  onUseSampleDiagnostics: _onUseSampleDiagnostics,
  onUseFullDiagnostics: _onUseFullDiagnostics,
  onMaintenanceMaxDeletesChange: _onMaintenanceMaxDeletesChange,
  onRecordsSearchChange: _onRecordsSearchChange,
  onRecordsNamespaceFilterChange: _onRecordsNamespaceFilterChange,
  onRecordsDocumentFilterChange: _onRecordsDocumentFilterChange,
  onRecordsNextPage: _onRecordsNextPage,
  onRecordsPreviousPage: _onRecordsPreviousPage,
  onPreviewMaintenance: _onPreviewMaintenance,
  onApplyMaintenance: _onApplyMaintenance,
  onPreviewDeleteDocument: _onPreviewDeleteDocument,
  onApplyDeleteDocument: _onApplyDeleteDocument,
  onRetry,
}: MetricsPanelProps) {
  const handleRetry = () => {
    if (typeof onRetry === "function") {
      onRetry();
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 ko-icon animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading metrics...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-alert ko-alert-danger">
        <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="ko-alert-title ko-text-danger-strong">Failed to load metrics</p>
          <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          <Button onClick={handleRetry} variant="danger" className="mt-3">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!hasData) {
    return (
      <div className="ko-panel p-6 text-center">
        <p className="ko-text-sm ko-muted">Metrics data is not available yet.</p>
        <Button onClick={handleRetry} variant="primary" className="mt-3">
          Retry
        </Button>
      </div>
    );
  }

  const safeViewModel = viewModel ?? DEFAULT_VIEW_MODEL;
  const metricCards = Array.isArray(safeViewModel.metricCards) ? safeViewModel.metricCards : EMPTY_METRIC_CARDS;
  const collections = Array.isArray(safeViewModel.collections) ? safeViewModel.collections : EMPTY_COLLECTIONS;
  const overallHealth =
    typeof safeViewModel.overallHealth === "string" && safeViewModel.overallHealth.trim().length > 0
      ? safeViewModel.overallHealth
      : "unknown";
  const lastUpdated =
    typeof safeViewModel.lastUpdated === "string" && safeViewModel.lastUpdated.trim().length > 0
      ? safeViewModel.lastUpdated
      : "Unknown";
  const totalEntriesLabel =
    typeof safeViewModel.totalEntriesLabel === "string" && safeViewModel.totalEntriesLabel.trim().length > 0
      ? safeViewModel.totalEntriesLabel
      : "Unknown";
  const hasMetrics = safeViewModel.hasMetrics || metricCards.length > 0;
  const ingestStatus = ingestHealth?.status?.trim() || "unknown";

  return (
    <div className="ko-stack">
      {/* Overall Status */}
      <div className="flex items-center justify-between" data-testid={selectors.metrics.overall}>
        <div>
          <h3 className="ko-text-lg font-semibold ko-text-strong">Overall Health</h3>
          <p className="ko-text-sm ko-muted capitalize">{overallHealth} condition</p>
        </div>
        <Button onClick={handleRetry} variant="outline" size="sm" data-testid={selectors.metrics.refresh}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div
        className="ko-card p-3 ko-text-xs ko-subtle"
        data-testid={selectors.metrics.legend}
      >
        Green signals healthy quality, yellow means watch closely, and red indicates degradation. Redundancy scores are
        better when lower.
      </div>

      <div className="ko-panel p-3 ko-text-xs ko-subtle">
        Each collection card shows how many items (embeddings) are currently stored. Click <strong>Open Details</strong> to
        debug or manage a specific collection.
      </div>

      <div className="ko-panel p-4">
        <h4 className="font-semibold ko-text-strong mb-2">Ingest Pipeline</h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2 ko-text-xs">
          <div><span className="ko-subtle">Status:</span><span className="ko-text-strong ml-1 capitalize">{ingestStatus}</span></div>
          <div><span className="ko-subtle">Pending:</span><span className="ko-text-strong ml-1">{ingestHealth?.pending_jobs ?? 0}</span></div>
          <div><span className="ko-subtle">Running:</span><span className="ko-text-strong ml-1">{ingestHealth?.running_jobs ?? 0}</span></div>
          <div><span className="ko-subtle">Failures (24h):</span><span className="ko-text-strong ml-1">{ingestHealth?.failures_last_24h ?? 0}</span></div>
        </div>
        <p className="ko-text-xs ko-subtle mt-2">
          Runner interval: {ingestHealth?.runner_interval_ms ?? 500}ms
          {typeof ingestHealth?.oldest_pending_age_ms === "number"
            ? ` · Oldest pending age: ${Math.round(ingestHealth.oldest_pending_age_ms / 1000)}s`
            : ""}
        </p>
      </div>

      {!hasMetrics && (
        <div className="ko-panel p-4 ko-text-sm ko-muted">
          Quality metrics are unavailable (not computed yet). Vector counts below are live when Qdrant is reachable.
        </div>
      )}

      {/* Metrics Grid */}
      {hasMetrics && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {metricCards.map((card, index) => (
            <MetricCard key={card.label || `metric-${index + 1}`} {...card} />
          ))}
        </div>
      )}

      {/* Collections Breakdown */}
      {collections.length > 0 && (
        <div className="ko-panel p-4" data-testid={selectors.metrics.collections}>
          <div className="flex items-center gap-2 mb-4">
            <Database className="h-5 w-5 ko-icon" />
            <h4 className="font-semibold ko-text-strong">Collections</h4>
          </div>
          <div className="ko-stack-sm">
            {collections.map((collection, index) => {
              const metrics = Array.isArray(collection.metrics) ? collection.metrics : [];
              const name =
                typeof collection.name === "string" && collection.name.trim().length > 0
                  ? collection.name
                  : `Collection ${index + 1}`;
              const inventory = getCollectionInventory(name);
              const itemCountLabel =
                typeof inventory?.total_points === "number"
                  ? inventory.total_points.toLocaleString()
                  : typeof collection.sizeLabel === "string" && collection.sizeLabel.trim().length > 0
                    ? collection.sizeLabel
                    : "Unknown";
              const ownership = resolveCollectionOwnership(
                name,
                selectedCollection,
                diagnostics,
                inventory
              );

              return (
                <div key={name} className="ko-card p-3">
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="ko-text-sm font-semibold ko-text-primary">{name}</span>
                      <span className={cn("ko-text-[11px] px-2 py-0.5 rounded", ownershipToneClass(ownership.tone))}>
                        {ownership.label}
                      </span>
                    </div>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 mt-3 mb-3">
                    <div className="ko-panel p-2">
                      <p className="ko-meta">Items</p>
                      <p className="text-lg font-semibold ko-text-strong">{itemCountLabel}</p>
                      <p className="ko-subtle text-[11px]">Embeddings stored in this collection</p>
                    </div>
                    <div className="ko-panel p-2">
                      <p className="ko-meta">Namespaces</p>
                      <p className="text-lg font-semibold ko-text-strong">{inventory?.distinct_namespaces ?? 0}</p>
                    </div>
                    <div className="ko-panel p-2">
                      <p className="ko-meta">Ingest Attempts</p>
                      <p className="text-lg font-semibold ko-text-strong">{inventory?.ingest_attempts ?? 0}</p>
                    </div>
                  </div>
                  <p className="ko-text-xs ko-subtle mb-3">{ownership.reason}</p>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      variant="primary"
                      size="sm"
                      onClick={() => onSelectCollection(name)}
                    >
                      Open Details
                    </Button>
                  </div>
                  {metrics.length > 0 && (
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 ko-text-xs">
                      {metrics.map((metric, metricIndex) => (
                        <div key={`${name}-${metric.label || metricIndex}`}>
                          <span className="ko-subtle">{metric.label ?? "Metric"}:</span>
                          <span className="ko-text-strong ml-1">
                            {metric.percentageLabel ?? "N/A"}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                  <div className="ko-text-xs ko-subtle mt-2">Open details to inspect chunking, failures, records, and cleanup actions.</div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {maintenanceNotice && (
        <div className="ko-panel p-3 ko-text-xs ko-subtle">
          {maintenanceNotice}
        </div>
      )}

      {/* Summary Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 ko-text-sm" data-testid={selectors.metrics.summary}>
        <div className="ko-card p-3">
          <p className="ko-meta mb-1">Total Vectors</p>
          <p className="text-xl font-bold ko-text-strong">
            {totalEntriesLabel}
          </p>
        </div>
        <div className="ko-card p-3">
          <p className="ko-meta mb-1">Last Updated</p>
          <p className="ko-text-sm font-semibold ko-text-strong">
            {lastUpdated}
          </p>
        </div>
      </div>
    </div>
  );
}

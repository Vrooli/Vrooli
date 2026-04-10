import { useState } from "react";
import { AlertCircle, ArrowLeft, Database, Loader2, RefreshCw, Trash2 } from "lucide-react";
import { cn } from "../../../shared/lib/utils";
import { Button } from "../../../shared/ui/button";
import { type MetricsPanelProps } from "./MetricsPanel";

type CollectionDrilldownTab = "integrity" | "chunking" | "failures" | "records" | "maintenance";

const TAB_OPTIONS: Array<{ id: CollectionDrilldownTab; label: string }> = [
  { id: "integrity", label: "Integrity" },
  { id: "chunking", label: "Chunking" },
  { id: "failures", label: "Failures" },
  { id: "records", label: "Records" },
  { id: "maintenance", label: "Maintenance" },
];

const TAB_INTROS: Record<CollectionDrilldownTab, { title: string; description: string }> = {
  integrity: {
    title: "Integrity checks",
    description:
      "Use this tab first. It tells you whether this collection looks internally consistent and whether cleanup is likely safe.",
  },
  chunking: {
    title: "Chunk quality",
    description:
      "This tab explains how text was split into chunks. Very small or very large chunks can hurt retrieval quality.",
  },
  failures: {
    title: "Ingest reliability",
    description:
      "Use this tab when new data is not appearing. It shows failed ingest attempts and how often failures are happening.",
  },
  records: {
    title: "Raw record inspection",
    description:
      "This tab lets you inspect actual stored records and payload fields, useful when debugging specific documents or namespaces.",
  },
  maintenance: {
    title: "Safe cleanup actions",
    description:
      "Always preview first, then apply. This tab helps remove stale or duplicate embeddings with lower risk.",
  },
};

type Props = MetricsPanelProps & {
  onBackToMetrics: () => void;
};

function scoreDriverHints(diagnostics: NonNullable<Props["diagnostics"]>, failuresLast24h: number): string[] {
  const hints: string[] = [];
  if ((diagnostics.redundancy?.duplicate_ratio ?? 0) >= 0.15) {
    hints.push("Redundancy is high. Run dedupe preview before applying deletes.");
  }
  if ((diagnostics.stale_chunks?.candidate_delete_rows ?? 0) > 0) {
    hints.push("Stale chunk versions detected. Use prune stale to keep newest chunk indexes.");
  }
  if ((diagnostics.vector_dimensions?.length ?? 0) > 1) {
    hints.push("Mixed embedding dimensions detected. Reingest with one embedding model.");
  }
  if (failuresLast24h > 0) {
    hints.push("Recent ingest failures found. Validate Ollama and Qdrant reachability.");
  }
  if (hints.length === 0) {
    hints.push("No immediate integrity risks detected in current diagnostics.");
  }
  return hints;
}

export function CollectionDetailsPanel({
  isLoading,
  hasError,
  errorMessage,
  hasData,
  selectedCollection,
  diagnostics,
  diagnosticsError,
  diagnosticsLoading,
  diagnosticsMode,
  diagnosticsLimit,
  drilldownTab,
  maintenanceInFlight,
  maintenanceNotice,
  maintenanceMaxDeletes,
  getMaintenancePreview,
  getCollectionInventory,
  documentOptions,
  selectedDocumentKey,
  documentDeletePreview,
  collectionRecords,
  recordsLoading,
  recordsError,
  recordsSearch,
  recordsNamespaceFilter,
  recordsDocumentFilter,
  ingestHealth,
  onBackToMetrics,
  onDrilldownTabChange,
  onSelectedDocumentKeyChange,
  onUseSampleDiagnostics,
  onUseFullDiagnostics,
  onMaintenanceMaxDeletesChange,
  onRecordsSearchChange,
  onRecordsNamespaceFilterChange,
  onRecordsDocumentFilterChange,
  onRecordsNextPage,
  onRecordsPreviousPage,
  onPreviewMaintenance,
  onApplyMaintenance,
  onPreviewDeleteDocument,
  onApplyDeleteDocument,
  collectionDeleteInFlight,
  onDeleteCollection,
  onRetry,
}: Props) {
  const [isDeleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 ko-icon animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading collection details...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-alert ko-alert-danger">
        <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="ko-alert-title ko-text-danger-strong">Failed to load collection details</p>
          <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <Button onClick={onRetry} variant="danger">
              Retry
            </Button>
            <Button onClick={onBackToMetrics} variant="secondary">
              Back to Metrics
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (!hasData || !selectedCollection.trim()) {
    return (
      <div className="ko-panel p-6 text-center">
        <p className="ko-text-sm ko-muted">Collection details are not available yet.</p>
        <div className="mt-3 flex justify-center gap-2">
          <Button onClick={onRetry} variant="primary">
            Retry
          </Button>
          <Button onClick={onBackToMetrics} variant="secondary">
            Back to Metrics
          </Button>
        </div>
      </div>
    );
  }

  const inventory = getCollectionInventory(selectedCollection);
  const prunePreview = getMaintenancePreview(selectedCollection, "prune-stale-chunks");
  const dedupePreview = getMaintenancePreview(selectedCollection, "dedupe-content");
  const selectedDocumentOption = documentOptions.find((entry) => entry.key === selectedDocumentKey) ?? null;
  const hints = diagnostics ? scoreDriverHints(diagnostics, ingestHealth?.failures_last_24h ?? 0) : [];
  const ownershipTone =
    inventory?.ownership === "knowledge_observatory"
      ? "ko-tone-good"
      : inventory?.ownership === "mixed"
        ? "ko-tone-poor"
        : "ko-tone-medium";

  return (
    <div className="ko-stack">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Button onClick={onBackToMetrics} variant="outline" size="sm">
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back
          </Button>
          <div>
            <h3 className="ko-text-lg font-semibold ko-text-strong flex items-center gap-2">
              <Database className="h-5 w-5 ko-icon" />
              {selectedCollection}
            </h3>
            <p className="ko-text-sm ko-muted">Debug embedding quality, chunking, ingest reliability, and cleanup in one place.</p>
          </div>
        </div>
        <Button onClick={onRetry} variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      {inventory && (
        <div className="ko-card p-3">
          <div className="flex flex-wrap items-center gap-2 ko-text-xs">
            <span className={cn("rounded px-2 py-0.5", ownershipTone)}>{inventory.ownership_label}</span>
            <span className="ko-subtle">Points: {inventory.total_points ?? "unknown"}</span>
            <span className="ko-subtle">Ingest attempts: {inventory.ingest_attempts}</span>
            <span className="ko-subtle">Namespaces: {inventory.distinct_namespaces}</span>
          </div>
          <p className="ko-subtle text-xs mt-2">
            Quick read: <strong>Points</strong> is total embeddings, <strong>Ingest attempts</strong> is write activity, and{" "}
            <strong>Namespaces</strong> is source breadth.
          </p>
        </div>
      )}

      <div className="ko-card p-3 flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant={diagnosticsMode === "sample" ? "primary" : "outline"}
          size="sm"
          onClick={onUseSampleDiagnostics}
          disabled={diagnosticsLoading}
        >
          Sample Scan
        </Button>
        <Button
          type="button"
          variant={diagnosticsMode === "full" ? "primary" : "outline"}
          size="sm"
          onClick={onUseFullDiagnostics}
          disabled={diagnosticsLoading}
        >
          Full Scan
        </Button>
        <label className="ko-subtle text-xs">
          Max Deletes
          <input
            type="number"
            min={1}
            step={1}
            className="ml-2 w-24 ko-input h-8 text-xs"
            value={maintenanceMaxDeletes}
            onChange={(event) => onMaintenanceMaxDeletesChange(Math.max(1, Number(event.target.value) || 1))}
          />
        </label>
        <span className="ko-subtle text-xs">Limit: {diagnosticsLimit}</span>
      </div>
      <div className="ko-panel p-3 ko-text-xs ko-subtle">
        Recommended sequence: 1) Run <strong>Sample Scan</strong> for a quick read, 2) open tabs to diagnose issues,
        3) use <strong>Maintenance</strong> previews, 4) apply only after preview counts look expected.
      </div>

      <div className="flex flex-wrap gap-2">
        {TAB_OPTIONS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            className={cn("ko-tab", drilldownTab === tab.id ? "ko-tab-active" : "ko-tab-inactive")}
            onClick={() => onDrilldownTabChange(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {diagnosticsLoading && <p className="ko-subtle">Loading diagnostics…</p>}
      {!diagnosticsLoading && diagnosticsError && <p className="ko-text-danger-muted">{diagnosticsError}</p>}
      {!diagnosticsLoading && !diagnosticsError && diagnostics && (
        <div className="ko-panel p-4 ko-stack-xs">
          <div className="ko-card p-3">
            <p className="ko-text-sm font-semibold ko-text-strong">{TAB_INTROS[drilldownTab].title}</p>
            <p className="ko-subtle text-xs mt-1">{TAB_INTROS[drilldownTab].description}</p>
          </div>

          {drilldownTab === "integrity" && (
            <>
              <p className="ko-subtle">
                Analyzed {diagnostics.analyzed_points}
                {typeof diagnostics.total_points === "number" ? ` / ${diagnostics.total_points}` : ""} points.
              </p>
              <p className="ko-subtle">Duplicate ratio: {(diagnostics.redundancy.duplicate_ratio * 100).toFixed(1)}%</p>
              <p className="ko-subtle">
                Vector dimensions:{" "}
                {diagnostics.vector_dimensions.map((entry) => `${entry.dimension} (${entry.count})`).join(", ") || "unknown"}
              </p>
              <p className="ko-subtle">
                Missing payload fields:{" "}
                {Object.entries(diagnostics.missing_payload_fields ?? {})
                  .filter(([, count]) => count > 0)
                  .map(([field, count]) => `${field}:${count}`)
                  .join(", ") || "none"}
              </p>
              <p className="ko-subtle">
                Rule of thumb: lower duplicate ratio is better; one vector dimension is expected for a healthy single-model collection.
              </p>
              <ul className="list-disc list-inside ko-text-primary text-sm">
                {hints.map((entry, index) => (
                  <li key={`hint-${index}`}>{entry}</li>
                ))}
              </ul>
            </>
          )}

          {drilldownTab === "chunking" && (
            <>
              <p className="ko-subtle">
                Chunk length avg: {Math.round(diagnostics.chunk_length.avg_characters)} chars
                {" · "}
                range {diagnostics.chunk_length.min_characters}-{diagnostics.chunk_length.max_characters}
              </p>
              <p className="ko-subtle">Stale delete candidates: {diagnostics.stale_chunks.candidate_delete_rows}</p>
              <p className="ko-subtle">Rule of thumb: average chunk size around 300-1500 characters is usually easier to retrieve well.</p>
              {diagnostics.stale_chunks?.top_documents?.length > 0 && (
                <ul className="list-disc list-inside ko-text-primary text-sm">
                  {diagnostics.stale_chunks.top_documents.slice(0, 10).map((doc, index) => (
                    <li key={`doc-${index}`}>
                      {doc.name} ({doc.count})
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}

          {drilldownTab === "failures" && (
            <>
              {!diagnostics.ingest_history && <p className="ko-subtle">No ingest history available for this collection yet.</p>}
              {diagnostics.ingest_history && (
                <>
                  <p className="ko-subtle">Attempts: {diagnostics.ingest_history.total_attempts}</p>
                  <p className="ko-subtle">
                    Failures: {diagnostics.ingest_history.failure_count} total ·{" "}
                    {diagnostics.ingest_history.failure_count_last_24h} in 24h
                  </p>
                  <p className="ko-subtle">Failure rate: {(diagnostics.ingest_history.failure_rate * 100).toFixed(1)}%</p>
                  {diagnostics.ingest_history.last_failure_at && (
                    <p className="ko-subtle">
                      Last failure: {new Date(diagnostics.ingest_history.last_failure_at).toLocaleString()}
                    </p>
                  )}
                  <p className="ko-subtle">If failure rate is rising, fix connectivity/model issues first, then reingest.</p>
                </>
              )}
            </>
          )}

          {drilldownTab === "maintenance" && (
            <div className="ko-stack-xs">
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => onPreviewMaintenance(selectedCollection, "prune-stale-chunks")}
                  disabled={maintenanceInFlight}
                >
                  Preview Prune
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => onPreviewMaintenance(selectedCollection, "dedupe-content")}
                  disabled={maintenanceInFlight}
                >
                  Preview Dedupe
                </Button>
                <Button
                  type="button"
                  variant="primary"
                  size="sm"
                  onClick={() => onApplyMaintenance(selectedCollection, "prune-stale-chunks")}
                  disabled={maintenanceInFlight || !prunePreview}
                >
                  Apply Prune
                </Button>
                <Button
                  type="button"
                  variant="primary"
                  size="sm"
                  onClick={() => onApplyMaintenance(selectedCollection, "dedupe-content")}
                  disabled={maintenanceInFlight || !dedupePreview}
                >
                  Apply Dedupe
                </Button>
              </div>
              {(prunePreview || dedupePreview) && (
                <div className="ko-card p-3 ko-text-xs">
                  {prunePreview && (
                    <p>
                      Prune preview: {prunePreview.candidate_delete_count} candidates from {prunePreview.analyzed_points} analyzed points.
                    </p>
                  )}
                  {dedupePreview && (
                    <p>
                      Dedupe preview: {dedupePreview.candidate_delete_count} candidates from {dedupePreview.analyzed_points} analyzed points.
                    </p>
                  )}
                </div>
              )}
              <div className="ko-card p-3 ko-stack-xs">
                <p className="ko-subtle">Document-level delete (for stale document versions)</p>
                {documentOptions.length === 0 && (
                  <p className="ko-subtle">No stale documents available from diagnostics to investigate.</p>
                )}
                {documentOptions.length > 0 && (
                  <>
                    <select
                      className="ko-input h-9 text-xs"
                      value={selectedDocumentKey}
                      onChange={(event) => onSelectedDocumentKeyChange(event.target.value)}
                    >
                      {documentOptions.map((option) => (
                        <option key={option.key} value={option.key}>
                          {option.label} ({option.count})
                        </option>
                      ))}
                    </select>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        onClick={onPreviewDeleteDocument}
                        disabled={maintenanceInFlight || !selectedDocumentOption}
                      >
                        Preview Doc Delete
                      </Button>
                      <Button
                        type="button"
                        variant="danger"
                        size="sm"
                        onClick={onApplyDeleteDocument}
                        disabled={maintenanceInFlight || !documentDeletePreview}
                      >
                        Apply Doc Delete
                      </Button>
                    </div>
                    {documentDeletePreview && (
                      <p className="ko-subtle">
                        {documentDeletePreview.namespace}/{documentDeletePreview.document_id}: {documentDeletePreview.candidate_delete_count} candidates
                      </p>
                    )}
                  </>
                )}
              </div>
              <p className="ko-subtle">
                Safety note: preview operations do not delete data. Apply buttons are intentionally blocked until a preview exists.
              </p>
              {Array.isArray(diagnostics.recommendations) && diagnostics.recommendations.length > 0 && (
                <ul className="list-disc list-inside ko-text-primary text-sm">
                  {diagnostics.recommendations.map((entry, index) => (
                    <li key={`rec-${index}`}>{entry}</li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {drilldownTab === "records" && (
            <div className="ko-stack-xs">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
                <label className="ko-subtle">
                  Search
                  <input
                    className="ko-input mt-1 h-9 text-xs"
                    value={recordsSearch}
                    onChange={(event) => onRecordsSearchChange(event.target.value)}
                    placeholder="content, hash, or document id"
                  />
                </label>
                <label className="ko-subtle">
                  Namespace
                  <input
                    className="ko-input mt-1 h-9 text-xs"
                    value={recordsNamespaceFilter}
                    onChange={(event) => onRecordsNamespaceFilterChange(event.target.value)}
                    placeholder="optional"
                  />
                </label>
                <label className="ko-subtle">
                  Document
                  <input
                    className="ko-input mt-1 h-9 text-xs"
                    value={recordsDocumentFilter}
                    onChange={(event) => onRecordsDocumentFilterChange(event.target.value)}
                    placeholder="optional"
                  />
                </label>
              </div>
              {recordsLoading && <p className="ko-subtle">Loading record preview…</p>}
              {!recordsLoading && recordsError && <p className="ko-text-danger-muted">{recordsError}</p>}
              {!recordsLoading && !recordsError && (
                <>
                  <p className="ko-subtle">
                    Showing {collectionRecords?.records.length ?? 0} of {collectionRecords?.total_count ?? 0} records.
                  </p>
                  <p className="ko-subtle">
                    Tip: start with Namespace + Document filters when tracking one source, then use Search for content/hash checks.
                  </p>
                  {(collectionRecords?.records.length ?? 0) > 0 ? (
                    <div className="ko-stack-xs">
                      {collectionRecords?.records.map((record) => (
                        <div key={record.id} className="ko-card p-3">
                          <div className="flex flex-wrap gap-2 ko-text-[11px]">
                            <span className="ko-subtle">id:</span>
                            <span className="ko-text-strong">{record.id}</span>
                            {record.namespace && (
                              <>
                                <span className="ko-subtle">namespace:</span>
                                <span className="ko-text-strong">{record.namespace}</span>
                              </>
                            )}
                            {record.document_id && (
                              <>
                                <span className="ko-subtle">document:</span>
                                <span className="ko-text-strong">{record.document_id}</span>
                              </>
                            )}
                          </div>
                          {record.content_preview && <p className="ko-text-xs mt-2 whitespace-pre-wrap">{record.content_preview}</p>}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="ko-subtle">No records matched current filters.</p>
                  )}
                  <div className="flex items-center gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={onRecordsPreviousPage}
                      disabled={recordsLoading || (collectionRecords?.offset ?? 0) <= 0}
                    >
                      Previous
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={onRecordsNextPage}
                      disabled={recordsLoading || typeof collectionRecords?.next_offset !== "number"}
                    >
                      Next
                    </Button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      )}

      {maintenanceNotice && <div className="ko-panel p-3 ko-text-xs ko-subtle">{maintenanceNotice}</div>}

      <div className="ko-card p-4 border ko-stack-xs">
        <p className="ko-text-sm font-semibold ko-text-danger-strong">Danger Zone</p>
        <p className="ko-subtle text-xs">
          Delete collection removes all embeddings in this collection and clears related ingest history/metadata in
          knowledge-observatory.
        </p>
        <div>
          <Button
            type="button"
            variant="danger"
            size="sm"
            onClick={() => {
              setDeleteConfirmText("");
              setDeleteModalOpen(true);
            }}
            disabled={collectionDeleteInFlight}
          >
            <Trash2 className="h-4 w-4 mr-1" />
            Delete Collection
          </Button>
        </div>
      </div>

      {isDeleteModalOpen && (
        <div className="ko-modal-backdrop" onClick={() => setDeleteModalOpen(false)}>
          <div className="ko-modal max-w-xl" onClick={(event) => event.stopPropagation()}>
            <div className="ko-modal-header">
              <div>
                <h4 className="ko-text-danger-strong font-semibold">Confirm Collection Delete</h4>
                <p className="ko-subtle text-xs mt-1">
                  This permanently deletes all items in <strong>{selectedCollection}</strong>.
                </p>
              </div>
            </div>
            <div className="ko-modal-body">
              <p className="ko-subtle text-xs">
                To confirm, type the collection name exactly: <strong>{selectedCollection}</strong>
              </p>
              <input
                className="ko-input h-10 text-sm"
                value={deleteConfirmText}
                onChange={(event) => setDeleteConfirmText(event.target.value)}
                placeholder={selectedCollection}
              />
            </div>
            <div className="ko-modal-footer flex gap-2">
              <Button type="button" variant="outline" size="sm" onClick={() => setDeleteModalOpen(false)}>
                Cancel
              </Button>
              <Button
                type="button"
                variant="danger"
                size="sm"
                onClick={() => {
                  onDeleteCollection(selectedCollection);
                  setDeleteModalOpen(false);
                }}
                disabled={collectionDeleteInFlight || deleteConfirmText.trim() !== selectedCollection}
              >
                Confirm Delete
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

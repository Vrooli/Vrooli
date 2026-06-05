// DOC: docs/reference/api-endpoints.md#health-metrics
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { MetricsPanel } from "./components/MetricsPanel";
import { CollectionDetailsPanel } from "./components/CollectionDetailsPanel";
import { useKnowledgeMetrics } from "../../shared/hooks/knowledgeHooks";
import {
  type CollectionInventoryItem,
  type CollectionDeleteResponse,
  type CollectionMaintenanceResponse,
  type CollectionRecordsResponse,
  fetchCollectionInventory,
  fetchCollectionRecords,
  fetchCollectionDiagnostics,
  runCollectionDelete,
  runCollectionMaintenance,
} from "../../shared/services/api";

type MaintenanceAction = "prune-stale-chunks" | "dedupe-content";
type DiagnosticsMode = "sample" | "full";
type CollectionDrilldownTab = "integrity" | "chunking" | "failures" | "records" | "maintenance";

type MetricsPanelContainerProps = {
  mode?: "overview" | "details";
  initialCollection?: string;
  onOpenCollection?: (collectionName: string) => void;
  onBackToMetrics?: () => void;
};

export function MetricsPanelContainer({
  mode = "overview",
  initialCollection = "",
  onOpenCollection,
  onBackToMetrics,
}: MetricsPanelContainerProps = {}) {
  const { viewModel, isLoading, errorMessage, hasError, hasData, refetch } = useKnowledgeMetrics();
  const [selectedCollection, setSelectedCollection] = useState("");
  const [maintenanceNotice, setMaintenanceNotice] = useState("");
  const [diagnosticsMode, setDiagnosticsMode] = useState<DiagnosticsMode>("sample");
  const [diagnosticsLimit, setDiagnosticsLimit] = useState(1200);
  const [maintenanceMaxDeletes, setMaintenanceMaxDeletes] = useState(500);
  const [maintenancePreviews, setMaintenancePreviews] = useState<Record<string, CollectionMaintenanceResponse>>({});
  const [drilldownTab, setDrilldownTab] = useState<CollectionDrilldownTab>("integrity");
  const [recordOffset, setRecordOffset] = useState(0);
  const [recordSearch, setRecordSearch] = useState("");
  const [recordNamespaceFilter, setRecordNamespaceFilter] = useState("");
  const [recordDocumentFilter, setRecordDocumentFilter] = useState("");

  const previewKey = (collection: string, action: MaintenanceAction) => `${collection}::${action}`;

  const diagnosticsQuery = useQuery({
    queryKey: ["collection-diagnostics", selectedCollection, diagnosticsMode, diagnosticsLimit],
    queryFn: () => fetchCollectionDiagnostics(selectedCollection, diagnosticsMode, diagnosticsLimit),
    enabled: selectedCollection.trim().length > 0,
    refetchInterval: 30000,
  });

  const collectionInventoryQuery = useQuery({
    queryKey: ["collection-inventory"],
    queryFn: fetchCollectionInventory,
    refetchInterval: 30000,
  });

  const recordsQuery = useQuery({
    queryKey: [
      "collection-records",
      selectedCollection,
      recordOffset,
      recordSearch,
      recordNamespaceFilter,
      recordDocumentFilter,
    ],
    queryFn: () =>
      fetchCollectionRecords(selectedCollection, {
        limit: 25,
        offset: recordOffset,
        search: recordSearch,
        namespace: recordNamespaceFilter,
        document_id: recordDocumentFilter,
      }),
    enabled: selectedCollection.trim().length > 0,
    refetchInterval: 30000,
  });

  const collectionInventoryMap = useMemo(() => {
    const map = new Map<string, CollectionInventoryItem>();
    const rows = collectionInventoryQuery.data?.collections ?? [];
    rows.forEach((row) => map.set(row.name, row));
    return map;
  }, [collectionInventoryQuery.data?.collections]);

  const maintenanceMutation = useMutation({
    mutationFn: async (params: { collection: string; action: MaintenanceAction; dryRun: boolean }) =>
      runCollectionMaintenance(params.collection, params.action, {
        dry_run: params.dryRun,
        max_deletes: maintenanceMaxDeletes > 0 ? maintenanceMaxDeletes : undefined,
      }),
    onSuccess: (response, variables) => {
      if (variables.dryRun) {
        setMaintenancePreviews((previous) => ({
          ...previous,
          [previewKey(response.collection, variables.action)]: response,
        }));
        setMaintenanceNotice(
          `Preview ready for ${response.collection} (${response.action}): ${response.candidate_delete_count} candidates.`
        );
      } else {
        setMaintenancePreviews((previous) => {
          const next = { ...previous };
          delete next[previewKey(response.collection, variables.action)];
          return next;
        });
        setMaintenanceNotice(
          `${response.action} applied for ${response.collection}: ${response.deleted_count} deleted (${response.candidate_delete_count} candidates).`
        );
      }
      void refetch();
      void diagnosticsQuery.refetch();
    },
    onError: (error) => {
      const message = error instanceof Error ? error.message : "maintenance failed";
      setMaintenanceNotice(`Maintenance failed: ${message}`);
    },
  });

  const collectionDeleteMutation = useMutation({
    mutationFn: async (collection: string): Promise<CollectionDeleteResponse> => runCollectionDelete(collection),
    onSuccess: (response) => {
      setMaintenanceNotice(`Collection ${response.collection} deleted. ${response.metadata_rows_deleted} metadata rows cleaned.`);
      setSelectedCollection("");
      setMaintenancePreviews({});
      void refetch();
      void diagnosticsQuery.refetch();
      void recordsQuery.refetch();
      void collectionInventoryQuery.refetch();
      if (typeof onBackToMetrics === "function") {
        onBackToMetrics();
      }
    },
    onError: (error) => {
      const message = error instanceof Error ? error.message : "collection delete failed";
      setMaintenanceNotice(`Collection delete failed: ${message}`);
    },
  });

  const diagnosticsError = useMemo(() => {
    if (!diagnosticsQuery.error) return "";
    return diagnosticsQuery.error instanceof Error ? diagnosticsQuery.error.message : "Unable to load diagnostics.";
  }, [diagnosticsQuery.error]);

  useEffect(() => {
    const normalized = initialCollection.trim();
    if (!normalized) return;
    setSelectedCollection(normalized);
  }, [initialCollection]);

  useEffect(() => {
    if (selectedCollection.trim()) return;
    if (mode !== "details") return;
    if (initialCollection.trim()) return;
    if (viewModel.collections.length === 0) return;
    const first = viewModel.collections[0];
    if (first?.name) {
      setSelectedCollection(first.name);
    }
  }, [initialCollection, mode, selectedCollection, viewModel.collections]);

  useEffect(() => {
    setRecordOffset(0);
  }, [selectedCollection, recordSearch, recordNamespaceFilter, recordDocumentFilter]);

  const recordsError = useMemo(() => {
    if (!recordsQuery.error) return "";
    return recordsQuery.error instanceof Error ? recordsQuery.error.message : "Unable to load collection records.";
  }, [recordsQuery.error]);

  const recordsData: CollectionRecordsResponse | null = recordsQuery.data ?? null;

  const sharedProps = {
    isLoading,
    hasError,
    errorMessage,
    hasData,
    viewModel,
    selectedCollection,
    diagnostics: diagnosticsQuery.data ?? null,
    diagnosticsError,
    diagnosticsLoading: diagnosticsQuery.isLoading || diagnosticsQuery.isFetching,
    diagnosticsMode,
    diagnosticsLimit,
    drilldownTab,
    maintenanceInFlight: maintenanceMutation.isPending || collectionDeleteMutation.isPending,
    maintenanceNotice,
    maintenanceMaxDeletes,
    getMaintenancePreview: (collection: string, action: MaintenanceAction) =>
      maintenancePreviews[previewKey(collection, action)] ?? null,
    getCollectionInventory: (collection: string) => collectionInventoryMap.get(collection) ?? null,
    collectionRecords: recordsData,
    recordsLoading: recordsQuery.isLoading || recordsQuery.isFetching,
    recordsError,
    recordsSearch: recordSearch,
    recordsNamespaceFilter: recordNamespaceFilter,
    recordsDocumentFilter: recordDocumentFilter,
    onSelectCollection: (name: string) => {
      setSelectedCollection(name);
      setDrilldownTab("integrity");
      setMaintenanceNotice("");
    },
    onDrilldownTabChange: (tab: CollectionDrilldownTab) => setDrilldownTab(tab),
    onUseSampleDiagnostics: () => {
      setDiagnosticsMode("sample");
      setDiagnosticsLimit(1200);
    },
    onUseFullDiagnostics: () => {
      setDiagnosticsMode("full");
      setDiagnosticsLimit(5000);
    },
    onMaintenanceMaxDeletesChange: (value: number) => {
      setMaintenanceMaxDeletes(value);
    },
    onRecordsSearchChange: (value: string) => setRecordSearch(value),
    onRecordsNamespaceFilterChange: (value: string) => setRecordNamespaceFilter(value),
    onRecordsDocumentFilterChange: (value: string) => setRecordDocumentFilter(value),
    onRecordsNextPage: () => {
      if (typeof recordsData?.next_offset === "number") {
        setRecordOffset(recordsData.next_offset);
      }
    },
    onRecordsPreviousPage: () => {
      const previous = Math.max(0, recordOffset - (recordsData?.limit ?? 25));
      setRecordOffset(previous);
    },
    onPreviewMaintenance: (collection: string, action: MaintenanceAction) => {
      setMaintenanceNotice(`Running preview for ${action} on ${collection}...`);
      maintenanceMutation.mutate({ collection, action, dryRun: true });
    },
    onApplyMaintenance: (collection: string, action: MaintenanceAction) => {
      setMaintenanceNotice(`Applying ${action} on ${collection}...`);
      maintenanceMutation.mutate({ collection, action, dryRun: false });
    },
    collectionDeleteInFlight: collectionDeleteMutation.isPending,
    onDeleteCollection: (collection: string) => {
      const normalized = collection.trim();
      if (!normalized) return;
      setMaintenanceNotice(`Deleting collection ${normalized}...`);
      collectionDeleteMutation.mutate(normalized);
    },
    onRetry: () => {
      void refetch();
      void collectionInventoryQuery.refetch();
      if (selectedCollection.trim()) {
        void diagnosticsQuery.refetch();
        void recordsQuery.refetch();
      }
    },
  };

  if (mode === "details") {
    return (
      <CollectionDetailsPanel
        {...sharedProps}
        onBackToMetrics={() => {
          if (typeof onBackToMetrics === "function") {
            onBackToMetrics();
          }
        }}
      />
    );
  }

  return (
    <MetricsPanel
      {...sharedProps}
      onSelectCollection={(name) => {
        setSelectedCollection(name);
        setMaintenanceNotice("");
        if (typeof onOpenCollection === "function") {
          onOpenCollection(name);
        }
      }}
    />
  );
}

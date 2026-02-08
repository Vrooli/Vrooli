// DOC: docs/reference/api-endpoints.md#health-metrics
import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { MetricsPanel } from "./components/MetricsPanel";
import { useKnowledgeMetrics } from "../../shared/hooks/knowledgeHooks";
import {
  fetchCollectionDiagnostics,
  fetchIngestHealth,
  runCollectionMaintenance,
} from "../../shared/services/api";

export function MetricsPanelContainer() {
  const { viewModel, isLoading, errorMessage, hasError, hasData, refetch } = useKnowledgeMetrics();
  const [selectedCollection, setSelectedCollection] = useState("");
  const [maintenanceActionLabel, setMaintenanceActionLabel] = useState("");

  const ingestHealthQuery = useQuery({
    queryKey: ["ingest-health"],
    queryFn: fetchIngestHealth,
    refetchInterval: 15000,
  });

  const diagnosticsQuery = useQuery({
    queryKey: ["collection-diagnostics", selectedCollection],
    queryFn: () => fetchCollectionDiagnostics(selectedCollection, "sample", 1200),
    enabled: selectedCollection.trim().length > 0,
    refetchInterval: 30000,
  });

  const maintenanceMutation = useMutation({
    mutationFn: async (params: { collection: string; action: "prune-stale-chunks" | "dedupe-content" }) =>
      runCollectionMaintenance(params.collection, params.action, { dry_run: false }),
    onSuccess: (response) => {
      setMaintenanceActionLabel(
        `${response.action} completed for ${response.collection}: ${response.deleted_count} deleted (${response.candidate_delete_count} candidates).`
      );
      void refetch();
      void diagnosticsQuery.refetch();
      void ingestHealthQuery.refetch();
    },
    onError: (error) => {
      const message = error instanceof Error ? error.message : "maintenance failed";
      setMaintenanceActionLabel(`Maintenance failed: ${message}`);
    },
  });

  const diagnosticsError = useMemo(() => {
    if (!diagnosticsQuery.error) return "";
    return diagnosticsQuery.error instanceof Error ? diagnosticsQuery.error.message : "Unable to load diagnostics.";
  }, [diagnosticsQuery.error]);

  return (
    <MetricsPanel
      isLoading={isLoading}
      hasError={hasError}
      errorMessage={errorMessage}
      hasData={hasData}
      viewModel={viewModel}
      ingestHealth={ingestHealthQuery.data ?? null}
      selectedCollection={selectedCollection}
      diagnostics={diagnosticsQuery.data ?? null}
      diagnosticsError={diagnosticsError}
      diagnosticsLoading={diagnosticsQuery.isLoading || diagnosticsQuery.isFetching}
      maintenanceActionLabel={maintenanceActionLabel}
      onSelectCollection={(name) => {
        setSelectedCollection(name);
        setMaintenanceActionLabel("");
      }}
      onRunPruneStale={(collection) => {
        setMaintenanceActionLabel(`Running prune_stale_chunks for ${collection}...`);
        maintenanceMutation.mutate({ collection, action: "prune-stale-chunks" });
      }}
      onRunDedupe={(collection) => {
        setMaintenanceActionLabel(`Running dedupe_content_hash for ${collection}...`);
        maintenanceMutation.mutate({ collection, action: "dedupe-content" });
      }}
      onRetry={() => {
        void refetch();
        void ingestHealthQuery.refetch();
        if (selectedCollection.trim()) {
          void diagnosticsQuery.refetch();
        }
      }}
    />
  );
}

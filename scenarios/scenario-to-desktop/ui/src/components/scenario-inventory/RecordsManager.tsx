import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Package, RefreshCw, AlertCircle, Trash2 } from "lucide-react";
import type { DesktopRecordResponse, TestArtifactSummary } from "../../lib/api";
import {
  cleanupTestArtifacts,
  deleteDesktopBuild,
  fetchDesktopRecords,
  fetchTestArtifacts,
  moveDesktopRecord,
} from "../../lib/api";
import { formatBytes } from "../../domain/download";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { AppCard } from "../records/AppCard";
import { AppDetailDrawer } from "../records/AppDetailDrawer";

interface RecordsManagerProps {
  onSwitchTemplate?: (scenarioName: string, templateType?: string) => void;
  onEditSigning?: (scenarioName: string) => void;
  onRebuildWithSigning?: (scenarioName: string) => void;
}

export function RecordsManager({ onSwitchTemplate, onEditSigning, onRebuildWithSigning }: RecordsManagerProps) {
  const queryClient = useQueryClient();
  const [selectedRecordId, setSelectedRecordId] = useState<string | null>(null);

  const { data, isLoading, error, refetch, isFetching } = useQuery<DesktopRecordResponse>({
    queryKey: ["desktop-records"],
    queryFn: fetchDesktopRecords,
    refetchInterval: 20000,
  });

  const moveMutation = useMutation({
    mutationFn: async (params: { recordId: string; target?: "destination" | "custom"; destination_path?: string }) => {
      return moveDesktopRecord(params.recordId, { target: params.target, destination_path: params.destination_path });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["desktop-records"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (scenarioName: string) => deleteDesktopBuild(scenarioName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["desktop-records"] });
      setSelectedRecordId(null);
    },
  });

  const records = useMemo(() => data?.records || [], [data]);
  const selectedItem = useMemo(
    () => records.find((r) => r.record.id === selectedRecordId) ?? null,
    [records, selectedRecordId],
  );

  const {
    data: testArtifacts,
    refetch: refetchArtifacts,
    isFetching: fetchingArtifacts,
    error: testArtifactError,
  } = useQuery<TestArtifactSummary>({
    queryKey: ["desktop-test-artifacts"],
    queryFn: fetchTestArtifacts,
    refetchInterval: 30000,
  });

  const cleanupArtifactsMutation = useMutation({
    mutationFn: cleanupTestArtifacts,
    onSuccess: () => {
      refetchArtifacts();
    },
  });

  const handleMove = (recordId: string, target: "destination" | "custom", customPath?: string) => {
    moveMutation.mutate({
      recordId,
      target,
      destination_path: customPath,
    });
  };

  const handleDelete = async (scenarioName: string) => {
    await deleteMutation.mutateAsync(scenarioName);
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold text-slate-50">Your Desktop Apps</h2>
            <Badge variant="outline">{records.length}</Badge>
          </div>
          <p className="text-sm text-slate-400 mt-0.5">
            Manage your generated desktop wrappers. Click an app to view details and actions.
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
          className="gap-2 shrink-0"
        >
          <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>

      {/* Test artifacts cleanup banner */}
      {(testArtifactError || (testArtifacts && testArtifacts.count > 0)) && (
        <div className="rounded border border-amber-800 bg-amber-950/20 p-3 text-sm text-amber-100 space-y-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4" />
              <span>
                {testArtifactError
                  ? "Unable to load test artifact stats."
                  : `${testArtifacts?.count ?? 0} legacy test artifact folders in /tmp totaling ${formatBytes(
                      testArtifacts?.total_bytes,
                    )}.`}
              </span>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => cleanupArtifactsMutation.mutate()}
              disabled={cleanupArtifactsMutation.isPending || !!testArtifactError || (testArtifacts?.count ?? 0) === 0}
              className="gap-2"
            >
              <Trash2 className="h-4 w-4" />
              Clean up
            </Button>
          </div>
          {testArtifacts?.paths && testArtifacts.paths.length > 0 && (
            <p className="text-xs text-amber-200/80">Examples: {testArtifacts.paths.join(", ")}</p>
          )}
          {cleanupArtifactsMutation.isSuccess && <p className="text-xs text-green-300">Cleanup completed.</p>}
          {cleanupArtifactsMutation.isError && (
            <p className="text-xs text-red-300">
              {(cleanupArtifactsMutation.error as Error).message || "Cleanup failed"}
            </p>
          )}
          {fetchingArtifacts && <p className="text-xs text-amber-200/70">Refreshing artifact stats…</p>}
          {testArtifactError && (
            <p className="text-xs text-red-300">
              {(testArtifactError as Error).message || "Error loading artifact stats"}
            </p>
          )}
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="rounded border border-red-800 bg-red-950/30 p-3 text-sm text-red-200 flex items-center gap-2">
          <AlertCircle className="h-4 w-4" />
          {(error as Error).message}
        </div>
      )}

      {/* Move mutation feedback */}
      {moveMutation.isError && (
        <p className="text-xs text-red-300">{(moveMutation.error as Error).message || "Move failed"}</p>
      )}
      {moveMutation.isSuccess && <p className="text-xs text-green-300">Move updated.</p>}

      {/* Content */}
      {isLoading ? (
        <div className="text-sm text-slate-300">Loading generated apps…</div>
      ) : records.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-8 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-800/80">
            <Package className="h-7 w-7 text-slate-500" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-200">No desktop apps yet</p>
            <p className="text-xs text-slate-400 mt-1">
              Use the Generator tab to build a desktop wrapper for one of your scenarios.
            </p>
          </div>
        </div>
      ) : (
        <>
          {/* Desktop grid */}
          <div className="hidden md:grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {records.map((item) => (
              <AppCard key={item.record.id} item={item} onClick={() => setSelectedRecordId(item.record.id)} />
            ))}
          </div>
          {/* Mobile list */}
          <div className="md:hidden">
            {records.map((item) => (
              <AppCard key={item.record.id} item={item} onClick={() => setSelectedRecordId(item.record.id)} />
            ))}
          </div>
        </>
      )}

      {/* Detail drawer */}
      <AppDetailDrawer
        item={selectedItem}
        open={selectedRecordId !== null}
        onClose={() => setSelectedRecordId(null)}
        onMove={handleMove}
        onDelete={handleDelete}
        movePending={moveMutation.isPending}
        onSwitchTemplate={onSwitchTemplate}
        onEditSigning={onEditSigning}
        onRebuildWithSigning={onRebuildWithSigning}
      />
    </div>
  );
}

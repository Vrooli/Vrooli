import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createPortal } from "react-dom";
import { Folder, RefreshCw, MoveRight, AlertCircle, CheckCircle2, Trash2, MoreVertical } from "lucide-react";
import type { DesktopRecordResponse, SigningReadinessResponse, TestArtifactSummary } from "../lib/api";
import { checkSigningReadiness, cleanupTestArtifacts, deleteDesktopBuild, fetchDesktopRecords, fetchTestArtifacts, moveDesktopRecord } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

interface RecordsManagerProps {
  onSwitchTemplate?: (scenarioName: string, templateType?: string) => void;
  onEditSigning?: (scenarioName: string) => void;
  onRebuildWithSigning?: (scenarioName: string) => void;
}

function pathLabel(record: DesktopRecordResponse["records"][number]) {
  const rec = record.record;
  if (!rec) return "Unknown";
  if (rec.location_mode === "temp" || rec.location_mode === "staging") {
    return rec.output_path || rec.staging_path || "staging";
  }
  return rec.output_path || rec.destination_path || "unknown";
}

export function RecordsManager({ onSwitchTemplate, onEditSigning, onRebuildWithSigning }: RecordsManagerProps) {
  const queryClient = useQueryClient();
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [moveDialog, setMoveDialog] = useState<{
    recordId: string;
    scenarioName: string;
    destinationPath?: string;
  } | null>(null);
  const [movePath, setMovePath] = useState("");
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [deleteResults, setDeleteResults] = useState<Record<string, { status: "success" | "error"; message?: string }>>(
    {}
  );

  const { data, isLoading, error, refetch, isFetching } = useQuery<DesktopRecordResponse>({
    queryKey: ["desktop-records"],
    queryFn: fetchDesktopRecords,
    refetchInterval: 20000
  });

  const moveMutation = useMutation({
    mutationFn: async (params: { recordId: string; target?: "destination" | "custom"; destination_path?: string }) => {
      return moveDesktopRecord(params.recordId, { target: params.target, destination_path: params.destination_path });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["desktop-records"] });
    }
  });

  const records = useMemo(() => data?.records || [], [data]);
  const {
    data: testArtifacts,
    refetch: refetchArtifacts,
    isFetching: fetchingArtifacts,
    error: testArtifactError
  } = useQuery<TestArtifactSummary>({
    queryKey: ["desktop-test-artifacts"],
    queryFn: fetchTestArtifacts,
    refetchInterval: 30000
  });

  const cleanupArtifactsMutation = useMutation({
    mutationFn: cleanupTestArtifacts,
    onSuccess: () => {
      refetchArtifacts();
    }
  });

  const deleteMutation = useMutation({
    mutationFn: async (scenarioName: string) => deleteDesktopBuild(scenarioName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["desktop-records"] });
    }
  });

  const formatBytes = (value?: number) => {
    if (!value || value <= 0) return "0 B";
    const units = ["B", "KB", "MB", "GB", "TB"];
    let v = value;
    let i = 0;
    while (v >= 1024 && i < units.length - 1) {
      v /= 1024;
      i++;
    }
    return `${v.toFixed(1)} ${units[i]}`;
  };

  useEffect(() => {
    if (!openMenuId) return;

    const handleKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpenMenuId(null);
      }
    };

    const handleClick = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setOpenMenuId(null);
      }
    };

    document.addEventListener("keydown", handleKey);
    document.addEventListener("mousedown", handleClick);
    return () => {
      document.removeEventListener("keydown", handleKey);
      document.removeEventListener("mousedown", handleClick);
    };
  }, [openMenuId]);

  const openMoveDialog = (recordId: string, scenarioName: string, destinationPath?: string) => {
    setMoveDialog({ recordId, scenarioName, destinationPath });
    setMovePath(destinationPath || "");
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Folder className="h-5 w-5 text-blue-300" />
          <h2 className="text-lg font-semibold text-slate-50">Generated Apps</h2>
          <Badge variant="outline">{records.length}</Badge>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
          className="gap-2"
        >
          <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>

      {(testArtifactError || (testArtifacts && testArtifacts.count > 0)) && (
        <div className="rounded border border-amber-800 bg-amber-950/20 p-3 text-sm text-amber-100 space-y-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4" />
              <span>
                {testArtifactError
                  ? "Unable to load test artifact stats."
                  : `${testArtifacts?.count ?? 0} legacy test artifact folders in /tmp totaling ${formatBytes(
                      testArtifacts?.total_bytes
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
          {testArtifacts.paths && testArtifacts.paths.length > 0 && (
            <p className="text-xs text-amber-200/80">
              Examples: {testArtifacts.paths.join(", ")}
            </p>
          )}
          {cleanupArtifactsMutation.isSuccess && (
            <p className="text-xs text-green-300">Cleanup completed.</p>
          )}
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

      {error && (
        <div className="rounded border border-red-800 bg-red-950/30 p-3 text-sm text-red-200 flex items-center gap-2">
          <AlertCircle className="h-4 w-4" />
          {(error as Error).message}
        </div>
      )}

      {isLoading ? (
        <div className="text-sm text-slate-300">Loading generated apps…</div>
      ) : records.length === 0 ? (
        <div className="rounded border border-slate-800 bg-slate-900/60 p-4 text-sm text-slate-300">
          No generated apps recorded yet. Build a desktop wrapper and it will appear here for management.
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {records.map((item) => {
            const rec = item.record;
            const inStaging = rec.location_mode === "temp" || rec.location_mode === "staging";
            const moveDisabled = moveMutation.isPending;
            const deleteDisabled = deletingId === rec.id;
            const locationPath = pathLabel(item);
            const destinationPath = rec.destination_path;
            const showDestination = destinationPath && destinationPath !== locationPath;
            return (
              <Card
                key={rec.id}
                className="relative overflow-hidden border-slate-800/80 bg-slate-950/80 shadow-xl shadow-blue-950/30 backdrop-blur"
              >
                <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-blue-500 via-cyan-400 to-purple-500" />
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="space-y-1">
                      <CardTitle className="text-lg text-slate-50">
                        {rec.app_display_name || rec.scenario_name}
                      </CardTitle>
                      <p className="text-xs text-slate-400">{rec.scenario_name}</p>
                      <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
                        <Badge variant="secondary" className="capitalize">
                          {rec.template_type || item.build_status?.template_type || "Unknown"}
                        </Badge>
                        {item.build_status?.framework && (
                          <Badge variant="outline">{item.build_status.framework}</Badge>
                        )}
                        <Badge variant="outline" className="capitalize">
                          {rec.location_mode || "proper"}
                        </Badge>
                        <SigningBadge scenarioName={rec.scenario_name} />
                      </div>
                    </div>
                    <div className="relative">
                      <button
                        type="button"
                        className="rounded-full border border-slate-800 bg-slate-900/80 p-2 text-slate-300 transition hover:border-slate-600 hover:bg-slate-900 hover:text-slate-100"
                        aria-label="Manage generated app"
                        onClick={() => setOpenMenuId((current) => (current === rec.id ? null : rec.id))}
                      >
                        <MoreVertical className="h-4 w-4" />
                      </button>
                      {openMenuId === rec.id && (
                        <div
                          ref={menuRef}
                          className="absolute right-0 top-10 z-20 w-48 rounded-lg border border-slate-800 bg-slate-950/95 p-2 shadow-xl"
                        >
                          <button
                            type="button"
                            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-slate-200 hover:bg-slate-800/70"
                            disabled={moveDisabled || (!inStaging && rec.output_path === rec.destination_path)}
                            onClick={() => {
                              setOpenMenuId(null);
                              moveMutation.mutate({ recordId: rec.id, target: "destination" });
                            }}
                          >
                            <MoveRight className="h-3.5 w-3.5" />
                            Move to destination
                          </button>
                          <button
                            type="button"
                            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-slate-200 hover:bg-slate-800/70"
                            onClick={() => {
                              setOpenMenuId(null);
                              openMoveDialog(rec.id, rec.scenario_name, rec.destination_path || rec.output_path);
                            }}
                          >
                            <MoveRight className="h-3.5 w-3.5" />
                            Move to custom path
                          </button>
                          {onSwitchTemplate && (
                            <button
                              type="button"
                              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-slate-200 hover:bg-slate-800/70"
                              onClick={() => {
                                setOpenMenuId(null);
                                onSwitchTemplate(
                                  rec.scenario_name,
                                  rec.template_type || item.build_status?.template_type
                                );
                              }}
                            >
                              Change template
                            </button>
                          )}
                          {onEditSigning && (
                            <button
                              type="button"
                              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-slate-200 hover:bg-slate-800/70"
                              onClick={() => {
                                setOpenMenuId(null);
                                onEditSigning(rec.scenario_name);
                              }}
                            >
                              Configure signing
                            </button>
                          )}
                          {onRebuildWithSigning && (
                            <button
                              type="button"
                              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-slate-200 hover:bg-slate-800/70"
                              onClick={() => {
                                setOpenMenuId(null);
                                onRebuildWithSigning(rec.scenario_name);
                              }}
                            >
                              Rebuild with signing
                            </button>
                          )}
                          <div className="my-1 border-t border-slate-800" />
                          <button
                            type="button"
                            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-red-200 hover:bg-red-900/40"
                            disabled={deleteDisabled}
                            onClick={async () => {
                              setOpenMenuId(null);
                              if (
                                !window.confirm(
                                  `Delete desktop build for "${rec.scenario_name}"? This removes platforms/electron for that scenario.`
                                )
                              ) {
                                return;
                              }
                              setDeletingId(rec.id);
                              try {
                                await deleteMutation.mutateAsync(rec.scenario_name);
                                setDeleteResults((prev) => ({ ...prev, [rec.id]: { status: "success" } }));
                              } catch (err) {
                                const message = (err as Error)?.message || "Delete failed";
                                setDeleteResults((prev) => ({ ...prev, [rec.id]: { status: "error", message } }));
                              } finally {
                                setDeletingId(null);
                              }
                            }}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                            Delete wrapper
                          </button>
                        </div>
                      )}
                    </div>
                    {item.build_status && (
                      <div className="flex items-center gap-2 rounded-full border border-slate-800 bg-slate-900/70 px-3 py-1 text-xs text-slate-100">
                        {item.build_status.status === "ready" ? (
                          <CheckCircle2 className="h-4 w-4 text-green-400" />
                        ) : (
                          <AlertCircle className="h-4 w-4 text-yellow-300" />
                        )}
                        <span className="capitalize">{item.build_status.status}</span>
                      </div>
                    )}
                  </div>
                  {deleteResults[rec.id]?.status === "success" && (
                    <p className="text-xs text-green-300">Deleted wrapper</p>
                  )}
                  {deleteResults[rec.id]?.status === "error" && (
                    <p className="text-xs text-red-300">{deleteResults[rec.id]?.message || "Delete failed"}</p>
                  )}
                </CardHeader>
                <CardContent className="space-y-4 text-sm text-slate-200">
                  <div className="space-y-1">
                    <p className="text-xs text-slate-400">Template</p>
                    <p className="text-sm text-slate-100">
                      {rec.template_type || item.build_status?.template_type || "Unknown"}
                    </p>
                  </div>

                  <div className="space-y-2">
                    <PathBlock label="Location" value={locationPath} />
                    {showDestination && <PathBlock label="Destination" value={destinationPath} />}
                    {moveMutation.isError && (
                      <p className="text-xs text-red-300">
                        {(moveMutation.error as Error).message || "Move failed"}
                      </p>
                    )}
                    {moveMutation.isSuccess && <p className="text-xs text-green-300">Move updated.</p>}
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
      {moveDialog &&
        createPortal(
          <div
            className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
            onClick={(event) => {
              if (event.target === event.currentTarget) {
                setMoveDialog(null);
              }
            }}
          >
            <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-950 p-5 shadow-2xl">
              <div className="space-y-1">
                <p className="text-sm font-semibold text-slate-100">Move wrapper</p>
                <p className="text-xs text-slate-400">
                  Choose a new folder for {moveDialog.scenarioName}.
                </p>
              </div>
              <div className="mt-4 space-y-2">
                <Input
                  value={movePath}
                  onChange={(event) => setMovePath(event.target.value)}
                  placeholder="Custom path..."
                  className="text-xs"
                />
              </div>
              <div className="mt-5 flex items-center justify-end gap-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setMoveDialog(null)}>
                  Cancel
                </Button>
                <Button
                  type="button"
                  size="sm"
                  onClick={async () => {
                    if (!moveDialog) return;
                    try {
                      await moveMutation.mutateAsync({
                        recordId: moveDialog.recordId,
                        target: "custom",
                        destination_path: movePath
                      });
                      setMoveDialog(null);
                    } catch {
                      // Keep dialog open so the user can adjust the path.
                    }
                  }}
                  disabled={moveDisabled || !movePath}
                  className="gap-2"
                >
                  <MoveRight className="h-4 w-4" />
                  Move
                </Button>
              </div>
            </div>
          </div>,
          document.body
        )}
    </div>
  );
}

function SigningBadge({ scenarioName }: { scenarioName: string }) {
  const { data, isFetching } = useQuery<SigningReadinessResponse>({
    queryKey: ["signing-readiness-record", scenarioName],
    queryFn: () => checkSigningReadiness(scenarioName),
    enabled: Boolean(scenarioName),
    staleTime: 30000
  });

  if (isFetching) {
    return (
      <Badge variant="outline" className="gap-1 border-slate-700 text-slate-200">
        <RefreshCw className="h-3.5 w-3.5 animate-spin" />
        Checking signing…
      </Badge>
    );
  }

  if (!data || !data.ready) {
    return (
      <Badge variant="outline" className="gap-1 border-amber-800 text-amber-200">
        <AlertCircle className="h-3.5 w-3.5" />
        Signing not ready
      </Badge>
    );
  }

  return (
    <Badge variant="outline" className="gap-1 border-green-800 text-green-200">
      <CheckCircle2 className="h-3.5 w-3.5" />
      Signing ready
    </Badge>
  );
}

function PathBlock({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="space-y-1">
      <p className="text-xs text-slate-400">{label}</p>
      <code className="block rounded-md border border-slate-800 bg-slate-950/70 px-3 py-2 text-xs text-slate-100 break-words">
        {value || "—"}
      </code>
    </div>
  );
}

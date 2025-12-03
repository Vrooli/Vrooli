import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Package, HelpCircle, Rocket, Plus, Upload, Workflow, RefreshCw, FileText, AlertTriangle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { GuidedFlow } from "../components/GuidedFlow";
import { Tip } from "../components/ui/tip";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { getDeploymentStatus, listTelemetry, uploadTelemetry } from "../lib/api";

export function Deployments() {
  const [showHelp, setShowHelp] = useState(false);
  const [guidedOpen, setGuidedOpen] = useState(false);
  const { id } = useParams<{ id?: string }>();

  const { data: deployment, isFetching, refetch, error } = useQuery({
    queryKey: ["deployment", id],
    queryFn: () => getDeploymentStatus(id!),
    enabled: Boolean(id),
    staleTime: 5_000,
  });
  const {
    data: telemetry,
    isFetching: telemetryLoading,
    refetch: refetchTelemetry,
    error: telemetryError,
  } = useQuery({
    queryKey: ["telemetry"],
    queryFn: () => listTelemetry(),
    staleTime: 30_000,
  });
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [scenarioName, setScenarioName] = useState<string>("");
  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!selectedFile) throw new Error("Pick a telemetry file first");
      return uploadTelemetry(scenarioName || deployment?.profile_id, selectedFile);
    },
    onSuccess: () => {
      setSelectedFile(null);
      void refetchTelemetry();
    },
  });

  useEffect(() => {
    if (scenarioName || !deployment) return;
    if (deployment.profile_id) {
      setScenarioName(deployment.profile_id);
    }
  }, [deployment, scenarioName]);

  const statusPill = useMemo(() => {
    if (!deployment) return null;
    const color =
      deployment.status === "completed"
        ? "bg-emerald-500/20 text-emerald-200 border-emerald-500/30"
        : deployment.status === "failed"
          ? "bg-red-500/20 text-red-100 border-red-500/30"
          : "bg-cyan-500/10 text-cyan-100 border-cyan-500/30";
    return <span className={`rounded-full border px-3 py-1 text-xs ${color}`}>{deployment.status}</span>;
  }, [deployment]);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <h1 className="text-3xl font-bold">Deployments</h1>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setShowHelp((v) => !v)} className="gap-2">
            <HelpCircle className="h-4 w-4" />
            {showHelp ? "Hide help" : "How this works"}
          </Button>
          <Button size="sm" className="gap-2" onClick={() => setGuidedOpen(true)}>
            <Rocket className="h-4 w-4" />
            Start guided flow
          </Button>
        </div>
      </div>
      <p className="text-slate-400 mt-1">
        Monitor active and past deployments. Start with a profile, then export/hand-off to scenario-to-*.
      </p>

      {showHelp && (
        <Tip title="How deployments work">
          <p>Deployments come from profiles. If you see nothing here, create a profile and run a deployment.</p>
          <p className="text-slate-300">Use the guided flow to pick a scenario + tier, plan swaps/secrets, then export or trigger a deploy.</p>
        </Tip>
      )}

      {/* Deployment detail or empty state */}
      {id ? (
        <div className="space-y-4">
          <Card>
            <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>Deployment {id}</CardTitle>
                <CardDescription>Monitor status and collect telemetry</CardDescription>
              </div>
              <div className="flex items-center gap-2">
                {statusPill}
                <Button size="sm" variant="outline" onClick={() => refetch()} className="gap-1" disabled={isFetching}>
                  <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
                  Refresh
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {error && (
                <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-100">
                  Failed to load deployment: {(error as Error).message}
                </div>
              )}
              {deployment && (
                <>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                      <p className="text-xs text-slate-400">Profile</p>
                      <p className="font-semibold">{deployment.profile_id}</p>
                    </div>
                    <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                      <p className="text-xs text-slate-400">Started</p>
                      <p className="font-semibold">{deployment.started_at || "—"}</p>
                    </div>
                    <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                      <p className="text-xs text-slate-400">Completed</p>
                      <p className="font-semibold">{deployment.completed_at || "—"}</p>
                    </div>
                    <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                      <p className="text-xs text-slate-400">Artifacts</p>
                      <p className="font-semibold">{deployment.artifacts?.length ?? 0} file(s)</p>
                    </div>
                  </div>
                  {deployment.message && (
                    <div className="rounded-lg border border-white/10 bg-white/5 p-3 text-sm text-slate-200">
                      {deployment.message}
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Resolve issues & ingest telemetry</CardTitle>
              <CardDescription>Mirrors the scenario-to-desktop guidance</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="rounded-lg border border-white/10 bg-white/5 p-3 space-y-3">
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <Upload className="h-4 w-4" />
                  Upload deployment-telemetry.jsonl
                </div>
                <p className="text-sm text-slate-400">
                  Import the telemetry file from the packaged app to see which dependencies or secrets failed. We store it under
                  <code className="mx-1 rounded bg-black/30 px-1 py-0.5 text-[11px] text-slate-100">~/.vrooli/deployment/telemetry/</code>
                  and surface failures below.
                </p>
                <p className="text-xs text-slate-400">
                  Bundled runtimes will auto-upload when a telemetry upload URL is included in the build; use this form for manual or fallback uploads.
                </p>
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="scenario">Scenario (optional)</Label>
                    <Input
                      id="scenario"
                      placeholder="picker-wheel"
                      value={scenarioName}
                      onChange={(e) => setScenarioName(e.target.value)}
                    />
                    <p className="text-xs text-slate-400">
                      Used for naming the telemetry file and grouping events. Defaults to the profile id if left blank.
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="telemetry-file">Telemetry file</Label>
                    <Input
                      id="telemetry-file"
                      type="file"
                      accept=".jsonl,.json,.txt"
                      onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)}
                    />
                    <p className="text-xs text-slate-400">
                      Each line must be JSON (deployment-telemetry.jsonl). We’ll validate before ingesting.
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Button
                    size="sm"
                    className="gap-2"
                    onClick={() => uploadMutation.mutate()}
                    disabled={uploadMutation.isPending || !selectedFile}
                  >
                    {uploadMutation.isPending ? (
                      <>
                        <RefreshCw className="h-4 w-4 animate-spin" />
                        Uploading...
                      </>
                    ) : (
                      <>
                        <Upload className="h-4 w-4" />
                        Upload telemetry
                      </>
                    )}
                  </Button>
                  {selectedFile && <span className="text-xs text-slate-400 truncate max-w-[200px]">{selectedFile.name}</span>}
                </div>
                {uploadMutation.error && (
                  <div className="rounded border border-red-500/30 bg-red-500/10 p-2 text-xs text-red-100">
                    {(uploadMutation.error as Error).message}
                  </div>
                )}
                {uploadMutation.isSuccess && (
                  <div className="rounded border border-emerald-500/30 bg-emerald-500/10 p-2 text-xs text-emerald-100">
                    Uploaded. Telemetry saved to {uploadMutation.data?.path}.
                  </div>
                )}
              </div>

              <div className="rounded-lg border border-white/10 bg-white/5 p-3 space-y-2">
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <FileText className="h-4 w-4" />
                  Recent telemetry ingests
                </div>
                {telemetryError && (
                  <div className="rounded border border-red-500/30 bg-red-500/10 p-2 text-xs text-red-100">
                    {(telemetryError as Error).message}
                  </div>
                )}
                {telemetryLoading && <p className="text-xs text-slate-400">Loading telemetry...</p>}
                {!telemetryLoading && telemetry && telemetry.length === 0 && (
                  <p className="text-xs text-slate-400">No telemetry ingested yet.</p>
                )}
                <div className="space-y-2">
                  {telemetry?.map((entry) => {
                    const failureEntries = Object.entries(entry.failure_counts || {}).filter(([, val]) => (val || 0) > 0);
                    const failureTotal = failureEntries.reduce((sum, [, val]) => sum + (val || 0), 0);
                    const recentEvents = entry.recent_events || [];
                    return (
                      <div key={entry.path} className="rounded border border-slate-700 bg-black/20 p-3 space-y-1">
                        <div className="flex items-start justify-between gap-3">
                          <div>
                            <div className="flex items-center gap-2 text-sm font-semibold">
                              {entry.scenario}
                              {failureTotal > 0 ? (
                                <Badge variant="destructive" className="gap-1">
                                  <AlertTriangle className="h-3 w-3" />
                                  {failureTotal} failure{failureTotal === 1 ? "" : "s"}
                                </Badge>
                              ) : (
                                <Badge variant="secondary">Clean</Badge>
                              )}
                            </div>
                            <p className="text-xs text-slate-400">
                              {entry.total_events} events • last {entry.last_event || "event"} @ {entry.last_timestamp || "unknown"}
                            </p>
                            <p className="text-[11px] text-slate-500 break-all">{entry.path}</p>
                          </div>
                          <Button size="sm" variant="ghost" onClick={() => refetchTelemetry()} className="gap-1">
                            <RefreshCw className="h-4 w-4" /> Refresh
                          </Button>
                        </div>
                        {entry.recent_failures && entry.recent_failures.length > 0 && (
                          <div className="rounded border border-red-500/30 bg-red-500/5 p-2 text-[11px] text-red-100 space-y-1">
                            <p className="font-semibold">Recent failures</p>
                            {entry.recent_failures.map((evt, idx) => (
                              <div key={idx} className="flex items-start gap-2">
                                <span className="mt-[2px] block h-1.5 w-1.5 rounded-full bg-red-300" />
                                <div>
                                  <div className="font-semibold">{evt.event || "unknown"}</div>
                                  <div className="text-slate-200">{evt.timestamp}</div>
                                  {evt.details && (
                                    <pre className="mt-1 whitespace-pre-wrap break-words rounded bg-black/30 p-2 text-[10px] text-slate-100">
                                      {JSON.stringify(evt.details, null, 2)}
                                    </pre>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        )}
                        {failureEntries.length > 0 && (
                          <div className="rounded border border-red-500/20 bg-red-500/5 p-2 text-[11px] text-red-100 space-y-1">
                            <p className="font-semibold">Failure breakdown</p>
                            <div className="grid gap-2 sm:grid-cols-2">
                              {failureEntries.map(([event, count]) => (
                                <div key={event} className="rounded border border-red-500/20 bg-black/20 p-2 flex items-center justify-between">
                                  <div className="flex items-center gap-2">
                                    <Badge variant="destructive" className="uppercase tracking-wide">{event}</Badge>
                                    <span className="text-sm font-semibold">{count}</span>
                                  </div>
                                  <span className="text-[11px] text-slate-300">Bundled runtime telemetry</span>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                        {recentEvents.length > 0 && (
                          <div className="rounded border border-white/10 bg-white/5 p-2 text-[11px] text-slate-200 space-y-1">
                            <p className="font-semibold">Recent events</p>
                            <div className="space-y-1">
                              {recentEvents.map((evt, idx) => (
                                <div key={`${entry.path}-${idx}`} className="rounded border border-slate-700 bg-black/20 p-2">
                                  <div className="flex items-center justify-between gap-2">
                                    <span className="text-[12px] font-semibold uppercase tracking-wide">{evt.event || "unknown"}</span>
                                    <span className="text-[11px] text-slate-400">{evt.timestamp || "unknown time"}</span>
                                  </div>
                                  {evt.details && (
                                    <pre className="mt-1 whitespace-pre-wrap break-words rounded bg-black/40 p-2 text-[10px] text-slate-100">
                                      {JSON.stringify(evt.details, null, 2)}
                                    </pre>
                                  )}
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <Workflow className="h-4 w-4" />
                  Open an issue in app-issue-tracker
                </div>
                <p className="text-sm text-slate-400 mt-1">
                  Create a deployment issue with context + telemetry so the built-in AI can propose fixes. Track status here once the API integration lands.
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Link to="/profiles">
                  <Button variant="outline" size="sm">Back to profiles</Button>
                </Link>
                <Button size="sm" onClick={() => setGuidedOpen(true)} className="gap-2">
                  <Rocket className="h-4 w-4" />
                  Start new guided flow
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Active Deployments</CardTitle>
            <CardDescription>
              No active deployments
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Package className="h-12 w-12 text-slate-600 mb-4" />
              <p className="text-slate-400 mb-4">No deployments yet</p>
              <p className="text-sm text-slate-500">
                Create a deployment profile and deploy it to see active deployments here
              </p>
              <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
                <Button size="sm" onClick={() => setGuidedOpen(true)} className="gap-2">
                  <Rocket className="h-4 w-4" />
                  Start guided flow
                </Button>
                <Link to="/profiles/new">
                  <Button size="sm" variant="outline" className="gap-2">
                    <Plus className="h-4 w-4" />
                    Create profile
                  </Button>
                </Link>
                <Link to="/profiles">
                  <Button size="sm" variant="ghost" className="gap-2">
                    View profiles
                  </Button>
                </Link>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <GuidedFlow open={guidedOpen} onClose={() => setGuidedOpen(false)} />
    </div>
  );
}

import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Package, HelpCircle, Rocket, Plus, Upload, Workflow, RefreshCw } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { GuidedFlow } from "../components/GuidedFlow";
import { Tip } from "../components/ui/tip";
import { getDeploymentStatus } from "../lib/api";

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
              <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <Upload className="h-4 w-4" />
                  Upload deployment-telemetry.jsonl
                </div>
                <p className="text-sm text-slate-400 mt-1">
                  Import the telemetry file from the packaged app to see which dependencies or secrets failed. (Upload UI pending; use issue tracker in the meantime.)
                </p>
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

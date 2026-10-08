import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ExternalLink, Loader2, ShieldCheck } from "lucide-react";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card";
import { listProfiles, getEvidenceReview } from "../../lib/api";
import type { TargetVerdict, DeploymentProfile } from "../../lib/api";
import { getErrorMessage } from "../../lib/utils";

interface JourneyStep {
  name: string;
  action: string;
  disposition: string;
  before_capture_id?: string;
  after_capture_id?: string;
  error?: string;
  degraded_reason?: string;
}

interface JourneyDetail {
  recording_url?: string;
  journey?: { degraded_reason?: string; steps?: JourneyStep[] };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function parseDetail(value: string): JourneyDetail | null {
  try {
    const parsed: unknown = JSON.parse(value);
    if (!isRecord(parsed)) return null;
    const recordingURL = typeof parsed.recording_url === "string" ? parsed.recording_url : undefined;
    const journeyValue = isRecord(parsed.journey) ? parsed.journey : undefined;
    const stepsValue = journeyValue?.steps;
    const steps = Array.isArray(stepsValue)
      ? stepsValue.filter(isRecord).map((step) => ({
        name: typeof step.name === "string" ? step.name : "unnamed",
        action: typeof step.action === "string" ? step.action : "unknown",
        disposition: typeof step.disposition === "string" ? step.disposition : "unknown",
        before_capture_id: typeof step.before_capture_id === "string" ? step.before_capture_id : undefined,
        after_capture_id: typeof step.after_capture_id === "string" ? step.after_capture_id : undefined,
        error: typeof step.error === "string" ? step.error : undefined,
        degraded_reason: typeof step.degraded_reason === "string" ? step.degraded_reason : undefined,
      }))
      : undefined;
    return { recording_url: recordingURL, journey: journeyValue ? {
      degraded_reason: typeof journeyValue.degraded_reason === "string" ? journeyValue.degraded_reason : undefined,
      steps,
    } : undefined };
  } catch {
    return null;
  }
}

function dispositionLabel(value: string): string {
  return value.replace(/^DISPOSITION_/, "").toLowerCase();
}

function verdictVariant(value: string): "success" | "warning" | "destructive" | "secondary" {
  const normalized = dispositionLabel(value);
  if (normalized === "passed") return "success";
  if (normalized === "pending") return "warning";
  if (normalized === "failed") return "destructive";
  return "secondary";
}

function recordingURL(detail: JourneyDetail | null): string | undefined {
  const value = detail?.recording_url;
  if (!value) return undefined;
  try {
    const parsed = new URL(value, window.location.origin);
    return parsed.protocol === "http:" || parsed.protocol === "https:" ? parsed.toString() : undefined;
  } catch {
    return undefined;
  }
}

function VerdictCard({ verdict }: { verdict: TargetVerdict }) {
  const detail = parseDetail(verdict.detail);
  const href = recordingURL(detail);
  const target = verdict.target;
  const steps = detail?.journey?.steps ?? [];
  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-lg">{target?.ramp || "unknown ramp"} / {target?.platform || "unknown platform"}</CardTitle>
          <Badge variant={verdictVariant(verdict.disposition)}>{dispositionLabel(verdict.disposition)}</Badge>
        </div>
        <CardDescription>
          Device: {target?.device_kind || "unspecified"} · Run: <span className="font-mono">{verdict.run_id}</span>
          {target?.bridge_node_id ? ` · Bridge: ${target.bridge_node_id}` : ""}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {detail?.journey?.degraded_reason && (
          <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100">
            Degraded reason: <span className="font-mono">{detail.journey.degraded_reason}</span>
          </div>
        )}
        {href && (
          <a data-testid="evidence-producer-recording" className="inline-flex items-center gap-2 text-sm text-cyan-300 hover:text-cyan-200" href={href} target="_blank" rel="noreferrer">
            Open producer recording <ExternalLink className="h-4 w-4" />
          </a>
        )}
        {steps.length > 0 && (
          <ol className="space-y-2 text-sm">
            {steps.map((step, index) => (
              <li key={`${step.name}-${index}`} className="flex flex-wrap items-center gap-2 rounded border border-white/10 bg-white/5 px-3 py-2">
                <span className="w-6 text-slate-500">{index + 1}.</span>
                <span className="font-medium">{step.name}</span>
                <span className="text-slate-400">{step.action}</span>
                <Badge variant={step.disposition === "passed" ? "success" : "destructive"}>{step.disposition}</Badge>
                {step.degraded_reason && <span className="text-amber-200">{step.degraded_reason}</span>}
                {step.error && <span className="text-red-200">{step.error}</span>}
              </li>
            ))}
          </ol>
        )}
        <div className="space-y-1 text-xs text-slate-400">
          <p>References: {verdict.refs.length}</p>
          {verdict.refs.map((ref) => <p key={ref.artifact_id} className="font-mono">{ref.kind}: {ref.producer}/{ref.artifact_id} ({ref.size_bytes} bytes)</p>)}
        </div>
      </CardContent>
    </Card>
  );
}

export function EvidenceReview() {
  const [profileID, setProfileID] = useState("");
  const [commit, setCommit] = useState("");
  const profilesQuery = useQuery({ queryKey: ["profiles"], queryFn: listProfiles });
  const reviewQuery = useQuery({
    queryKey: ["evidence-review", profileID, commit],
    queryFn: () => getEvidenceReview(profileID, commit),
    enabled: profileID.length > 0 && commit.trim().length > 0,
  });
  const grouped = new Map<string, TargetVerdict[]>();
  reviewQuery.data?.verdicts.forEach((verdict) => {
    const key = `${verdict.target?.ramp || "unknown"}/${verdict.target?.platform || "unknown"}`;
    grouped.set(key, [...(grouped.get(key) ?? []), verdict]);
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 data-testid="evidence-heading" className="text-3xl font-bold">Evidence review</h1>
        <p data-testid="evidence-review-intro" className="mt-1 text-slate-400">Review every ramp and target before allowing a release.</p>
      </div>
      <Card>
        <CardContent className="flex flex-wrap items-end gap-4 pt-6">
          <label className="min-w-52 flex-1 text-sm text-slate-300">Profile
            <select className="mt-1 w-full rounded-md border border-white/10 bg-white/5 px-3 py-2" value={profileID} onChange={(event) => setProfileID(event.target.value)}>
              <option value="">Select profile</option>
              {profilesQuery.data?.map((profile: DeploymentProfile) => <option key={profile.id} value={profile.id}>{profile.name}</option>)}
            </select>
          </label>
          <label className="min-w-64 flex-1 text-sm text-slate-300">Exact commit
            <input data-testid="evidence-commit-input" className="mt-1 w-full rounded-md border border-white/10 bg-white/5 px-3 py-2 font-mono" value={commit} onChange={(event) => setCommit(event.target.value)} placeholder="40-character commit hash" />
          </label>
          <Button variant="outline" disabled={!profileID || !commit.trim()} onClick={() => void reviewQuery.refetch()}>Review</Button>
        </CardContent>
      </Card>
      {reviewQuery.isLoading && <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />}
      {reviewQuery.error && <p className="rounded border border-red-500/30 bg-red-500/10 p-3 text-red-200">{getErrorMessage(reviewQuery.error)}</p>}
      {reviewQuery.data && (
        <Card>
          <CardHeader><CardTitle data-testid="evidence-gate-status" className="flex items-center gap-2"><ShieldCheck className="h-5 w-5" /> Gate: {reviewQuery.data.ready ? "ready" : "blocked"}</CardTitle><CardDescription>{reviewQuery.data.reason || "All reported targets passed."}</CardDescription></CardHeader>
        </Card>
      )}
      {Array.from(grouped.entries()).map(([group, verdicts]) => <section key={group} className="space-y-3"><h2 className="text-xl font-semibold">{group}</h2>{verdicts.map((verdict) => <VerdictCard key={verdict.run_id + group} verdict={verdict} />)}</section>)}
    </div>
  );
}

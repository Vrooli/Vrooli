import { AlertCircle, CheckCircle2, HelpCircle, XCircle } from "lucide-react";
import { cn } from "../../../lib/utils";
import type { HealthObservation } from "../../../lib/api";

/**
 * HealthObservationBadge renders the typed deployment health observation
 * (vrooli.scenario_to_cloud.v1.health.HealthObservation) without collapsing
 * its axes: the verdict (status), how recent the evidence is (freshness),
 * when it was observed, and which release was observed. A current-unhealthy
 * report and a stale-healthy report therefore look different, and an unknown
 * or unspecified status never renders as healthy.
 */
export function HealthObservationBadge({ observation }: { observation: HealthObservation | null | undefined }) {
  if (!observation) {
    return null;
  }

  const status = statusLabel(observation.status);
  const freshness = freshnessLabel(observation.freshness);
  const observedAt = formatObservedAt(observation.observed_at);
  const release = shortDigest(observation.observed_release_digest);
  const missing = observation.partial ? (observation.missing_dependencies ?? []).join(", ") : "";

  return (
    <span
      data-testid="health-observation-badge"
      data-status={status.key}
      data-freshness={freshness.key}
      title={[
        `Deployment ${observation.deployment_id ?? ""}`,
        `Status: ${status.text}`,
        `Freshness: ${freshness.text}`,
        observedAt ? `Observed at ${observedAt}` : "",
        observation.observed_release_digest ? `Release ${observation.observed_release_digest}` : "",
        missing ? `Partial: missing ${missing}` : "",
      ]
        .filter(Boolean)
        .join("\n")}
      className={cn("inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border text-sm font-medium", status.className)}
    >
      <status.Icon className="h-4 w-4" />
      <span>{status.text}</span>
      <span className={cn("text-xs px-1.5 py-0.5 rounded border", freshness.className)}>{freshness.text}</span>
      {observedAt && <span className="text-xs opacity-80">observed {observedAt}</span>}
      {release && <span className="text-xs font-mono opacity-80">release {release}</span>}
      {missing && <span className="text-xs opacity-80">partial: missing {missing}</span>}
    </span>
  );
}

type StatusView = {
  key: "healthy" | "degraded" | "unhealthy" | "unknown";
  text: string;
  className: string;
  Icon: typeof CheckCircle2;
};

function statusLabel(status: string | undefined): StatusView {
  switch (status) {
    case "HEALTH_STATUS_HEALTHY":
      return { key: "healthy", text: "Healthy", className: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30", Icon: CheckCircle2 };
    case "HEALTH_STATUS_DEGRADED":
      return { key: "degraded", text: "Degraded", className: "bg-amber-500/20 text-amber-400 border-amber-500/30", Icon: AlertCircle };
    case "HEALTH_STATUS_UNHEALTHY":
      return { key: "unhealthy", text: "Unhealthy", className: "bg-red-500/20 text-red-400 border-red-500/30", Icon: XCircle };
    default:
      // UNKNOWN, UNSPECIFIED, missing, or a vocabulary this UI does not know.
      return { key: "unknown", text: "Health unknown", className: "bg-slate-500/20 text-slate-300 border-slate-500/30", Icon: HelpCircle };
  }
}

function freshnessLabel(freshness: string | undefined): { key: "current" | "stale" | "unknown"; text: string; className: string } {
  switch (freshness) {
    case "FRESHNESS_CURRENT":
      return { key: "current", text: "current", className: "border-emerald-500/30 text-emerald-300" };
    case "FRESHNESS_STALE":
      return { key: "stale", text: "stale", className: "border-amber-500/30 text-amber-300" };
    default:
      return { key: "unknown", text: "freshness unknown", className: "border-slate-500/30 text-slate-300" };
  }
}

function formatObservedAt(value: string | undefined): string {
  if (!value) return "";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toISOString().replace(".000Z", "Z");
}

function shortDigest(digest: string | undefined): string {
  if (!digest) return "";
  const hex = digest.includes(":") ? digest.slice(digest.indexOf(":") + 1) : digest;
  return hex.length > 12 ? hex.slice(0, 12) : hex;
}

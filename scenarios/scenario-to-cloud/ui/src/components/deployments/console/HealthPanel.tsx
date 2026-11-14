import { useEffect, useState } from "react";
import { formatAge } from "../../../lib/consoleActions";
import type { HealthObservation } from "../../../lib/api";
import { KeyValue, LiveRegion, Panel, SkeletonRows, StatusPill, type Tone } from "./ConsolePrimitives";
import { NextActionControl } from "./NextActionControl";

export interface HealthPanelProps {
  observation: HealthObservation | null | undefined;
  loading: boolean;
  error: unknown;
  /** Injectable clock for deterministic ages in tests. */
  now?: () => number;
}

function statusView(status: string | undefined): { tone: Tone; text: string; key: string } {
  switch (status) {
    case "HEALTH_STATUS_HEALTHY":
      return { tone: "success", text: "Healthy", key: "healthy" };
    case "HEALTH_STATUS_DEGRADED":
      return { tone: "warning", text: "Degraded", key: "degraded" };
    case "HEALTH_STATUS_UNHEALTHY":
      return { tone: "danger", text: "Unhealthy", key: "unhealthy" };
    default:
      return { tone: "neutral", text: "Health unknown", key: "unknown" };
  }
}

function freshnessView(freshness: string | undefined): { tone: Tone; text: string; key: string } {
  switch (freshness) {
    case "FRESHNESS_CURRENT":
      return { tone: "success", text: "current", key: "current" };
    case "FRESHNESS_STALE":
      return { tone: "warning", text: "stale evidence", key: "stale" };
    default:
      return { tone: "neutral", text: "freshness unknown", key: "unknown" };
  }
}

function checkTone(status: string | undefined): Tone {
  switch (status) {
    case "CHECK_STATUS_PASSED":
      return "success";
    case "CHECK_STATUS_WARNED":
      return "warning";
    case "CHECK_STATUS_FAILED":
      return "danger";
    default:
      return "neutral";
  }
}

/**
 * HealthPanel answers "is the application healthy and how recent is that?"
 * without collapsing verdict, freshness, time and producer into one colour.
 */
export function HealthPanel({ observation, loading, error, now = () => Date.now() }: HealthPanelProps) {
  const status = statusView(observation?.status);
  const freshness = freshnessView(observation?.freshness);
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const id = window.setInterval(() => setTick((t) => t + 1), 30_000);
    return () => window.clearInterval(id);
  }, []);
  void tick;
  const age = formatAge(observation?.observed_at, now());
  const degraded = status.key !== "healthy" || freshness.key !== "current" || Boolean(observation?.partial);
  const state = loading ? "loading" : !observation ? "empty" : degraded ? "degraded" : "ready";
  const announcement = observation ? `Health ${status.text}, ${freshness.text}${age ? `, observed ${age}` : ""}` : "";

  return (
    <Panel
      testId="console-health"
      title="Health"
      state={state}
      actions={
        observation ? (
          <>
            <StatusPill tone={status.tone} testId="console-health-status">
              {status.text}
            </StatusPill>
            <StatusPill tone={freshness.tone} testId="console-health-freshness">
              {freshness.text}
            </StatusPill>
          </>
        ) : null
      }
    >
      <LiveRegion message={announcement} testId="console-health-live" />
      {loading && <SkeletonRows rows={4} />}
      {!loading && !observation && (
        <p className="text-sm text-slate-200" data-testid="console-health-empty">
          {error ? `No health observation: ${error instanceof Error ? error.message : String(error)}` : "No health observation has been produced for this deployment yet."}
        </p>
      )}
      {observation && (
        <div className="space-y-3">
          <dl className="space-y-1">
            <KeyValue label="Observed" testId="console-health-observed-at">
              {age || "time unknown"}
              {observation.observed_at && <span className="ml-2 text-xs text-slate-300">({observation.observed_at})</span>}
            </KeyValue>
            <KeyValue label="Producer" mono testId="console-health-producer">
              {observation.producer_ref || "unknown"}
            </KeyValue>
            <KeyValue label="Target" mono>
              {observation.target_id || "unknown"}
            </KeyValue>
            {observation.partial && (
              <KeyValue label="Partial" testId="console-health-partial">
                <span className="text-amber-200">missing {(observation.missing_dependencies ?? []).join(", ") || "unnamed dependencies"}</span>
              </KeyValue>
            )}
          </dl>
          {observation.checks && observation.checks.length > 0 && (
            <ul className="grid grid-cols-1 gap-1 sm:grid-cols-2" aria-label="Health checks" data-testid="console-health-checks">
              {observation.checks.map((check) => (
                <li key={check.id} className="flex flex-wrap items-center gap-2 rounded border border-white/10 px-2 py-1 text-xs">
                  <StatusPill tone={checkTone(check.status)}>{(check.status ?? "").replace("CHECK_STATUS_", "").toLowerCase() || "unknown"}</StatusPill>
                  <span className="font-mono text-slate-100">{check.id}</span>
                  {check.reason_code && <span className="text-slate-300">{check.reason_code}</span>}
                  {check.detail && <span className="text-slate-300 break-words min-w-0">{check.detail}</span>}
                </li>
              ))}
            </ul>
          )}
          {observation.next_actions && observation.next_actions.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {observation.next_actions.map((next, i) => (
                <NextActionControl key={`${next.kind}-${i}`} next={next} testId={`console-health-next-action-${i}`} />
              ))}
            </div>
          )}
        </div>
      )}
    </Panel>
  );
}

import type { PhaseExecutionResult } from "../../lib/api";
import { cn } from "../../lib/utils";
import { formatDuration } from "../../lib/formatters";

interface UXFindingContext {
  rank: number;
  route: string;
  viewport: string;
  identity: string;
  nextAction: string;
  screenshotRef: string;
}

function uxFindingContext(payload?: string): UXFindingContext | undefined {
  if (!payload) return undefined;
  try {
    const decoded = atob(payload);
    const bytes = Uint8Array.from(decoded, (char) => char.charCodeAt(0));
    const value: unknown = JSON.parse(new TextDecoder().decode(bytes));
    if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
    const record = value as Record<string, unknown>;
    const stringValue = (key: string) => typeof record[key] === "string" ? record[key] as string : undefined;
    const context = {
      rank: typeof record.rank === "number" && Number.isFinite(record.rank) ? record.rank : undefined,
      route: stringValue("route"),
      viewport: stringValue("viewport"),
      identity: stringValue("identity"),
      nextAction: stringValue("nextAction"),
      screenshotRef: stringValue("screenshotRef"),
    };
    if (typeof context.rank !== "number" || !Number.isInteger(context.rank) || context.rank < 1 || !context.route || !context.viewport || !context.identity || !context.nextAction || !context.screenshotRef) return undefined;
    return {
      rank: context.rank,
      route: context.route,
      viewport: context.viewport,
      identity: context.identity,
      nextAction: context.nextAction,
      screenshotRef: context.screenshotRef,
    };
  } catch {
    return undefined;
  }
}

function safeLoopbackScreenshotURL(ref?: string): string | undefined {
  if (!ref) return undefined;
  try {
    const url = new URL(ref);
    if ((url.protocol !== "http:" && url.protocol !== "https:") || url.username || url.password || url.search || url.hash) return undefined;
    const hostname = url.hostname.toLowerCase().replace(/^\[|\]$/g, "");
    const isLoopback = hostname === "localhost" || hostname.endsWith(".localhost") || hostname === "::1" || /^127(?:\.\d{1,3}){3}$/.test(hostname);
    if (!isLoopback) return undefined;
    return `${url.origin}${url.pathname}`;
  } catch {
    return undefined;
  }
}

function screenshotReferenceLabel(ref: string): string {
  try {
    const url = new URL(ref);
    if (url.protocol === "http:" || url.protocol === "https:") return url.pathname || "/";
    return "[screenshot reference withheld]";
  } catch {
    return ref.startsWith("/") ? ref : "[screenshot reference withheld]";
  }
}

function ExperienceFindings({ phase }: { phase: PhaseExecutionResult }) {
  const findings = (phase.findings ?? [])
    .map((finding, index) => ({ finding, index, evidence: finding.evidence?.find((item) => item.kind === "ui_health_result.v1") }))
    .map((item) => ({ ...item, context: uxFindingContext(item.evidence?.payload) }))
    .filter((item): item is typeof item & { context: UXFindingContext } => item.context !== undefined)
    .sort((left, right) => left.context.rank - right.context.rank || left.index - right.index);
  if (phase.name !== "experience") return null;
  const criticSkipped = (phase.findings ?? []).some((finding) => finding.code === "vision_ux_judge_skipped");
  if (findings.length === 0) {
    return <p className="mt-3 rounded-lg border border-amber-300/20 bg-amber-300/[0.04] p-3 text-xs text-amber-100" role="status" data-testid="experience-evidence-status">
      {criticSkipped
        ? "Local visual critique was skipped. This receipt has no screenshot-backed UX verdict."
        : "No screenshot-backed UX findings are attached. This receipt alone does not establish that the UI has no visual defects."}
    </p>;
  }
  return <>
    {criticSkipped && <p className="mt-3 rounded-lg border border-amber-300/20 bg-amber-300/[0.04] p-3 text-xs text-amber-100" role="status" data-testid="experience-evidence-status">
      At least one local visual critique was skipped. The screenshot-backed findings below are incomplete.
    </p>}
    <ol className="mt-3 grid gap-2" aria-label="Ranked experience findings">
    {findings.map(({ finding, context }) => {
      const screenshotRef = context.screenshotRef;
      const previewURL = safeLoopbackScreenshotURL(screenshotRef);
      const referenceLabel = screenshotReferenceLabel(screenshotRef);
      return <li key={finding.stableId || `${finding.code}-${context.rank}`} className="rounded-lg border border-violet-300/20 bg-violet-300/[0.04] p-3 text-xs" data-testid="experience-finding">
        <div className="flex flex-wrap items-center gap-2">
          <span className="rounded bg-violet-300/10 px-2 py-0.5 font-semibold text-violet-100">#{context.rank}</span>
          <strong className="text-slate-100">{finding.message || finding.code}</strong>
          {finding.severity && <span className="text-slate-400">{finding.severity}</span>}
        </div>
        <p className="mt-1 text-slate-300">
          {[context.identity, context.route, context.viewport].join(" · ")}
        </p>
        <p className="mt-2 text-cyan-100">Next: {context.nextAction}</p>
        {previewURL && <img src={previewURL} alt={`Evidence screenshot for ${context.identity} at ${context.viewport}`} loading="lazy" referrerPolicy="no-referrer" className="mt-3 max-h-72 max-w-full rounded-lg border border-white/10 object-contain" data-testid="experience-screenshot-preview" />}
        {screenshotRef && <p className="mt-2 break-all text-slate-400">Screenshot evidence: {previewURL ? <a href={previewURL} target="_blank" rel="noreferrer" className="text-cyan-200 underline">{referenceLabel}</a> : referenceLabel}</p>}
      </li>;
    })}
    </ol>
  </>;
}

interface PhaseResultCardProps {
  phase: PhaseExecutionResult;
  className?: string;
}

export function PhaseResultCard({ phase, className }: PhaseResultCardProps) {
  return (
    <div className={cn("rounded-xl border border-white/5 bg-black/20 p-3 text-sm", className)}>
      <div className="flex items-center justify-between">
        <span className="font-medium capitalize">{phase.name}</span>
        <span
          className={cn(
            "rounded-full px-2 py-0.5 text-xs uppercase",
            phase.status === "passed"
              ? "bg-emerald-500/20 text-emerald-300"
              : "bg-red-500/20 text-red-300"
          )}
        >
          {phase.status}
        </span>
      </div>
      <p className="mt-1 text-xs text-slate-400">
        {formatDuration(phase.durationSeconds)} · {phase.observations?.length ?? 0} observations
      </p>
      {phase.error && (
        <p className="mt-1 text-xs text-red-300">
          {phase.error.slice(0, 120)}
          {phase.error.length > 120 ? "…" : ""}
        </p>
      )}
      <ExperienceFindings phase={phase} />
    </div>
  );
}

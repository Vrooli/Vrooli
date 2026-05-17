import { useState } from "react";
import { Loader2, Play, RotateCcw } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { StatusDot } from "../../components/ui/status-dot";
import { cn } from "../../lib/utils";
import {
  getLastSuiteRun,
  runSuite,
  type SuiteCapability,
  type SuiteOverallStatus,
  type SuiteRun,
  type SuiteStep,
} from "../../services/diagnostics";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import type { ProviderTrace } from "../../services/diagnostics";

interface SuiteCardProps {
  /** Forward each step's trace into the right-rail timeline. */
  onTrace?: (capability: string, trace: ProviderTrace) => void;
  /** Click a tile — page may scroll to / highlight a capability panel. */
  onTileClick?: (capability: SuiteCapability, failed: boolean) => void;
}

const SUITE_CAPABILITIES: SuiteCapability[] = ["stt", "tts", "summarize", "transcode"];

export function SuiteCard({ onTrace, onTileClick }: SuiteCardProps) {
  const { t } = useTranslation();
  const qc = useQueryClient();

  const lastQuery = useQuery({ queryKey: ["diagnostics", "last"], queryFn: getLastSuiteRun });

  const runMutation = useMutation({
    mutationFn: () => runSuite([]),
    onSuccess: (result) => {
      if (result.ok) {
        if (onTrace) {
          for (const step of result.data.steps) {
            if (step.providerTier || step.providerId) {
              onTrace(step.capability, {
                providerTier: step.providerTier,
                providerId: step.providerId,
                modelId: step.modelId,
                latencyMs: step.latencyMs,
              });
            }
          }
        }
        qc.setQueryData(["diagnostics", "last"], result);
      }
    },
  });

  const liveRun: SuiteRun | null = runMutation.data?.ok
    ? runMutation.data.data
    : lastQuery.data?.ok
      ? lastQuery.data.data
      : null;

  const busy = runMutation.isPending;
  const everRan = !!liveRun && liveRun.runId !== "";
  // Auto-expand the step log whenever overall ≠ pass so failures are visible
  // without an extra click. User can still collapse manually.
  const autoOpen = everRan && liveRun!.overall !== "pass";
  const [logOpenOverride, setLogOpenOverride] = useState<boolean | null>(null);
  const logOpen = logOpenOverride ?? autoOpen;

  return (
    <section
      className="rounded-panel border border-app-border bg-app-surface text-app-foreground"
      aria-label={t(strings.diagnostics.suite.title)}
    >
      <OverallStrip
        run={liveRun}
        busy={busy}
        everRan={everRan}
        onRun={() => runMutation.mutate()}
      />

      <div className="grid gap-2 border-t border-app-border p-3 sm:grid-cols-2 lg:grid-cols-4">
        {SUITE_CAPABILITIES.map((cap) => (
          <CapabilityTile
            key={cap}
            capability={cap}
            step={findStep(liveRun, cap)}
            busy={busy}
            onClick={onTileClick}
          />
        ))}
      </div>

      <details
        open={logOpen}
        onToggle={(e) => setLogOpenOverride((e.currentTarget as HTMLDetailsElement).open)}
        className="border-t border-app-border bg-app-surface-muted"
      >
        <summary className="cursor-pointer px-3 py-2 text-xs font-medium text-app-muted-foreground select-none">
          {logOpen
            ? t(strings.diagnostics.suite.logToggleClose)
            : t(strings.diagnostics.suite.logToggleOpen)}
        </summary>
        <StepLog run={liveRun} />
      </details>

      {runMutation.data && !runMutation.data.ok ? (
        <p className="border-t border-app-border px-3 py-2 text-sm text-app-danger">
          {t(strings.diagnostics.suite.runFailed, { message: runMutation.data.error.message })}
        </p>
      ) : null}
    </section>
  );
}

function OverallStrip({
  run,
  busy,
  everRan,
  onRun,
}: {
  run: SuiteRun | null;
  busy: boolean;
  everRan: boolean;
  onRun: () => void;
}) {
  const { t } = useTranslation();
  const tone = overallTone(run?.overall ?? "never");
  const label = overallLabel(t, run?.overall ?? "never");
  const finishedAt = everRan ? new Date(run!.finishedAtUnixMs) : null;

  return (
    <div className="flex flex-wrap items-center gap-3 px-3 py-3">
      <StatusDot tone={tone} label={label} pulse={busy} />
      {everRan ? (
        <>
          <Badge variant="neutral">{run!.passCount}/{run!.totalCount}</Badge>
          <span
            className="hidden text-xs text-app-muted-foreground sm:inline"
            title={finishedAt!.toLocaleString()}
            data-testid="suite-last-run"
          >
            {t(strings.diagnostics.suite.lastRunLabel, {
              relative: relativeTime(t, finishedAt!),
              absolute: finishedAt!.toLocaleString(),
            })}
          </span>
        </>
      ) : (
        <span className="text-xs text-app-muted-foreground">
          {t(strings.diagnostics.suite.lastRunNever)}
        </span>
      )}
      <div className="ml-auto">
        <Button onClick={onRun} disabled={busy} data-testid="suite-run">
          {busy ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : everRan ? (
            <RotateCcw className="h-4 w-4" aria-hidden="true" />
          ) : (
            <Play className="h-4 w-4" aria-hidden="true" />
          )}
          {busy
            ? t(strings.diagnostics.suite.runningAction)
            : everRan
              ? t(strings.diagnostics.suite.rerunAction)
              : t(strings.diagnostics.suite.runAction)}
        </Button>
      </div>
    </div>
  );
}

function CapabilityTile({
  capability,
  step,
  busy,
  onClick,
}: {
  capability: SuiteCapability;
  step?: SuiteStep;
  busy: boolean;
  onClick?: (capability: SuiteCapability, failed: boolean) => void;
}) {
  const { t } = useTranslation();
  const tone = tileTone(step, busy);
  const status = tileStatus(t, step, busy);
  const failed = !!step && !step.ok;
  const interactive = !!onClick;
  return (
    <button
      type="button"
      onClick={interactive ? () => onClick!(capability, failed) : undefined}
      disabled={!interactive}
      className={cn(
        "flex flex-col gap-1 rounded-control border bg-app-surface px-3 py-2 text-left transition",
        failed
          ? "border-app-danger border-l-4 border-l-app-danger"
          : "border-app-border",
        interactive && "hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus",
        !interactive && "cursor-default",
      )}
      data-testid={`suite-tile-${capability}`}
      aria-label={`${capabilityLabel(t, capability)} — ${status}`}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-semibold text-app-foreground">{capabilityLabel(t, capability)}</span>
        <StatusDot tone={tone} label={status} pulse={busy} />
      </div>
      <div className="text-xs text-app-muted-foreground">
        {!step ? (
          <span>{t(strings.diagnostics.suite.tileNever)}</span>
        ) : step.ok ? (
          <span className="flex flex-wrap items-center gap-1">
            <Badge variant="info">{step.providerTier || "-"}</Badge>
            <span>{step.providerId || "-"}</span>
            <span>·</span>
            <span>{t(strings.diagnostics.suite.tileLatencyMs, { ms: Math.round(step.latencyMs) })}</span>
          </span>
        ) : (
          <span className="text-app-danger">
            {errorCodeLabel(t, step.errorCode) || step.errorMessage || "—"}
          </span>
        )}
      </div>
    </button>
  );
}

function StepLog({ run }: { run: SuiteRun | null }) {
  const { t } = useTranslation();
  if (!run || run.steps.length === 0) {
    return <p className="px-3 py-2 text-xs text-app-muted-foreground">{t(strings.diagnostics.suite.logEmpty)}</p>;
  }
  return (
    <ul className="divide-y divide-app-border">
      {run.steps.map((s, i) => (
        <li key={`${s.capability}-${i}`} className="px-3 py-2 text-xs">
          <span className="font-mono">
            {t(strings.diagnostics.suite.logEntry, {
              capability: s.capability,
              status: s.ok
                ? t(strings.diagnostics.suite.logStatusOK)
                : t(strings.diagnostics.suite.logStatusFail),
              latencyMs: Math.round(s.latencyMs),
            })}
          </span>
          {!s.ok ? (
            <span className="ml-2 text-app-danger">
              {s.errorCode}{s.errorMessage ? `: ${s.errorMessage}` : ""}
            </span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}

function findStep(run: SuiteRun | null, cap: SuiteCapability): SuiteStep | undefined {
  if (!run) return undefined;
  return run.steps.find((s) => s.capability === cap);
}

function overallTone(s: SuiteOverallStatus): "neutral" | "success" | "warning" | "danger" {
  switch (s) {
    case "pass": return "success";
    case "partial": return "warning";
    case "fail": return "danger";
    case "never":
    case "unknown":
    default:
      return "neutral";
  }
}

function overallLabel(t: ReturnType<typeof useTranslation>["t"], s: SuiteOverallStatus): string {
  switch (s) {
    case "pass": return t(strings.diagnostics.suite.overallPass);
    case "partial": return t(strings.diagnostics.suite.overallPartial);
    case "fail": return t(strings.diagnostics.suite.overallFail);
    case "never": return t(strings.diagnostics.suite.overallNever);
    default: return t(strings.diagnostics.suite.overallUnknown);
  }
}

function tileTone(step?: SuiteStep, busy?: boolean): "neutral" | "success" | "danger" | "warning" {
  if (busy && !step) return "warning";
  if (!step) return "neutral";
  return step.ok ? "success" : "danger";
}

function tileStatus(
  t: ReturnType<typeof useTranslation>["t"],
  step?: SuiteStep,
  busy?: boolean,
): string {
  if (busy && !step) return t(strings.diagnostics.suite.runningAction);
  if (!step) return t(strings.diagnostics.suite.tileNever);
  return step.ok ? t(strings.diagnostics.suite.tileOK) : t(strings.diagnostics.suite.tileFail);
}

function capabilityLabel(t: ReturnType<typeof useTranslation>["t"], c: SuiteCapability): string {
  switch (c) {
    case "stt": return t(strings.diagnostics.suite.tileLabelSTT);
    case "tts": return t(strings.diagnostics.suite.tileLabelTTS);
    case "summarize": return t(strings.diagnostics.suite.tileLabelSummarize);
    case "transcode": return t(strings.diagnostics.suite.tileLabelTranscode);
  }
}

function errorCodeLabel(t: ReturnType<typeof useTranslation>["t"], code: string): string {
  switch (code) {
    case "provider_unavailable": return t(strings.diagnostics.suite.errorCodeProviderUnavailable);
    case "not_configured": return t(strings.diagnostics.suite.errorCodeNotConfigured);
    case "deadline_exceeded": return t(strings.diagnostics.suite.errorCodeDeadlineExceeded);
    case "insufficient_credits": return t(strings.diagnostics.suite.errorCodeInsufficientCredits);
    case "internal": return t(strings.diagnostics.suite.errorCodeInternal);
    default: return "";
  }
}

function relativeTime(t: ReturnType<typeof useTranslation>["t"], when: Date): string {
  const seconds = Math.floor((Date.now() - when.getTime()) / 1000);
  if (seconds < 5) return t(strings.diagnostics.suite.lastRunJustNow);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return when.toLocaleDateString();
}

import { useState } from "react";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import {
  CheckCircle2,
  CircleSlash,
  Download,
  HelpCircle,
  Loader2,
  XCircle,
  type LucideIcon,
} from "lucide-react";

import { artifactDownloadUrl } from "../../api/client";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { RunEventKind, type Run, type RunEvent } from "../../api/runs";
import {
  RunStatus,
  isRunActive,
  useAbortRunMutation,
  useRunDetailQuery,
  useRunsQuery,
} from "./queries";

const STATUS_LABEL = {
  [RunStatus.UNSPECIFIED]: strings.runs.status.unspecified,
  [RunStatus.QUEUED]: strings.runs.status.queued,
  [RunStatus.RUNNING]: strings.runs.status.running,
  [RunStatus.PASSED]: strings.runs.status.passed,
  [RunStatus.FAILED]: strings.runs.status.failed,
  [RunStatus.ABORTED]: strings.runs.status.aborted,
} as const satisfies Record<RunStatus, string>;

// Status is conveyed by a distinct icon AND a text label — never color alone.
const STATUS_ICON: Record<RunStatus, LucideIcon> = {
  [RunStatus.UNSPECIFIED]: HelpCircle,
  [RunStatus.QUEUED]: Loader2,
  [RunStatus.RUNNING]: Loader2,
  [RunStatus.PASSED]: CheckCircle2,
  [RunStatus.FAILED]: XCircle,
  [RunStatus.ABORTED]: CircleSlash,
};

function jobLabel(run: Run): string {
  return [run.scenario, run.verb, ...run.args].filter(Boolean).join(" ");
}

/** Elapsed seconds for a started run, clamped at 0. */
function elapsedSeconds(run: Run, now: number): number {
  if (!run.startedAt) return 0;
  const started = timestampDate(run.startedAt).getTime();
  return Math.max(0, Math.floor((now - started) / 1000));
}

/**
 * Fractional progress (0..1) for an in-flight run, derived from elapsed time
 * against the wall-clock budget (`timeoutSeconds`). Terminal runs are 100%.
 * Capped just under 1 while running so the bar never visually "completes"
 * before the run actually does.
 */
function progressFraction(run: Run, now: number): number {
  if (!isRunActive(run.status)) return 1;
  const budget = Number(run.timeoutSeconds);
  if (!budget || budget <= 0) return 0;
  return Math.min(0.99, elapsedSeconds(run, now) / budget);
}

/** Remaining-budget ETA in whole seconds, or null when unknowable. */
function etaSeconds(run: Run, now: number): number | null {
  if (!isRunActive(run.status)) return null;
  const budget = Number(run.timeoutSeconds);
  if (!budget || budget <= 0) return null;
  if (run.status === RunStatus.QUEUED || !run.startedAt) return null;
  return Math.max(0, budget - elapsedSeconds(run, now));
}

function durationLabel(run: Run): string | null {
  if (!run.startedAt || !run.finishedAt) return null;
  const secs = Math.max(
    0,
    Math.round(
      (timestampDate(run.finishedAt).getTime() - timestampDate(run.startedAt).getTime()) / 1000,
    ),
  );
  return `${secs}s`;
}

/** The per-run live progress block: bar + ETA + cancel for in-flight runs. */
function RunProgress({ run }: { run: Run }) {
  const { t } = useTranslation();
  const abort = useAbortRunMutation();
  const now = Date.now();
  const StatusIcon = STATUS_ICON[run.status];
  const active = isRunActive(run.status);
  const eta = etaSeconds(run, now);
  const pct = Math.round(progressFraction(run, now) * 100);

  const handleCancel = () => {
    if (window.confirm(t(strings.runs.cancelConfirm))) {
      abort.mutate(run.id);
    }
  };

  return (
    <div className="mt-2 flex flex-col gap-1">
      <div className="flex items-center justify-between gap-2">
        <span className="inline-flex items-center gap-1 text-xs text-app-muted-foreground">
          <StatusIcon
            aria-hidden="true"
            className={["h-3.5 w-3.5", active ? "animate-spin" : ""].join(" ")}
          />
          {t(STATUS_LABEL[run.status])}
          {!active && run.status !== RunStatus.UNSPECIFIED && (
            <span>· {t(strings.runs.exitLabel)} {run.exitCode}</span>
          )}
        </span>
        {active && (
          <Button
            size="sm"
            variant="outline"
            data-testid={selectors.runs.cancel({ id: run.id })}
            onClick={handleCancel}
            disabled={abort.isPending}
          >
            {abort.isPending ? t(strings.runs.cancelling) : t(strings.runs.cancel)}
          </Button>
        )}
      </div>

      {active && (
        <>
          {/* Always a determinate bar — never a frozen spinner. */}
          <div
            role="progressbar"
            aria-valuenow={pct}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label={t(strings.runs.progressLabel)}
            className="h-1.5 w-full overflow-hidden rounded-pill bg-app-border"
          >
            <div className="h-full rounded-pill bg-app-primary transition-all" style={{ width: `${pct}%` }} />
          </div>
          <p className="text-xs text-app-muted-foreground">
            {t(strings.runs.etaLabel)}:{" "}
            {eta == null
              ? t(strings.runs.etaPending)
              : `~${eta}s`}
          </p>
        </>
      )}
    </div>
  );
}

function eventLine(event: RunEvent, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (event.kind) {
    case RunEventKind.LOG:
      return event.logChunk;
    case RunEventKind.STATUS:
      return `· ${event.status}`;
    case RunEventKind.EXIT:
      return `${t(strings.runs.exitLabel)} ${event.exitCode}`;
    case RunEventKind.ARTIFACT_REF:
      return `${t(strings.runs.artifactsHeading)}: ${event.artifactRef}`;
    default:
      return "";
  }
}

/** Drill-in detail: live/persisted output + downloadable artifacts. */
function RunDetail({ runId }: { runId: string }) {
  const { t } = useTranslation();
  const detail = useRunDetailQuery(runId);
  const run = detail.data?.run;
  const events = detail.data?.events ?? [];
  const artifacts = run?.artifactRefs ?? [];

  return (
    <div
      data-testid={selectors.runs.detail}
      className="mt-2 flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3"
    >
      {detail.isLoading && (
        <p className="text-sm text-app-muted-foreground">{t(strings.runs.loading)}</p>
      )}
      {detail.error && (
        <p role="alert" className="text-sm text-app-danger">
          {errorMessage(detail.error, t)}
        </p>
      )}

      <div>
        <p className="text-xs font-semibold text-app-foreground">{t(strings.runs.outputHeading)}</p>
        <pre
          data-testid={selectors.runs.output}
          className="mt-1 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-control bg-app-background p-2 font-mono text-xs text-app-foreground"
        >
          {events.length > 0
            ? events.map((e) => eventLine(e, t)).filter(Boolean).join("\n")
            : t(strings.runs.outputEmpty)}
        </pre>
      </div>

      <div data-testid={selectors.runs.artifacts}>
        <p className="text-xs font-semibold text-app-foreground">{t(strings.runs.artifactsHeading)}</p>
        {artifacts.length === 0 ? (
          <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.runs.artifactsEmpty)}</p>
        ) : (
          <ul className="mt-1 flex flex-col gap-1">
            {artifacts.map((ref, index) => (
              <li key={ref}>
                <a
                  data-testid={selectors.runs.artifact({ id: runId, index })}
                  href={artifactDownloadUrl(ref)}
                  download
                  className="inline-flex items-center gap-1 text-xs font-medium text-app-primary hover:underline"
                >
                  <Download aria-hidden="true" className="h-3.5 w-3.5" />
                  <span className="break-all">{ref}</span>
                  <span className="sr-only">{t(strings.runs.download)}</span>
                </a>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

/**
 * RunHistory is the durable remote-execution surface (OT-P0-005, OT-P1-005):
 * a newest-first feed of runs with live progress / ETA / cancel for in-flight
 * jobs and a drill-in showing the run's output and downloadable artifacts.
 * Loading / error / empty are explicit; in-flight runs always show a
 * determinate progress bar (never a frozen spinner) and advance via polling.
 */
export function RunHistory() {
  const { t } = useTranslation();
  const runsQuery = useRunsQuery();
  const [openId, setOpenId] = useState<string | null>(null);

  return (
    <section
      data-testid={selectors.runs.panel}
      aria-labelledby="runs-heading"
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 id="runs-heading" className="text-sm font-semibold text-app-foreground">
        {t(strings.runs.title)}
      </h3>
      <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.runs.description)}</p>

      {runsQuery.isLoading && (
        <p data-testid={selectors.runs.loading} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.runs.loading)}
        </p>
      )}

      {runsQuery.error && (
        <p data-testid={selectors.runs.error} className="mt-3 text-sm text-app-danger">
          {errorMessage(runsQuery.error, t)}
        </p>
      )}

      {runsQuery.data && runsQuery.data.length === 0 && (
        <p data-testid={selectors.runs.empty} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.runs.empty)}
        </p>
      )}

      {runsQuery.data && runsQuery.data.length > 0 && (
        <ul data-testid={selectors.runs.list} className="mt-3 flex flex-col gap-2">
          {runsQuery.data.map((run) => {
            const open = openId === run.id;
            const duration = durationLabel(run);
            return (
              <li
                key={run.id}
                data-testid={selectors.runs.row({ id: run.id })}
                className="rounded-panel border border-app-border bg-app-background p-3"
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-app-foreground">
                      <span className="sr-only">{t(strings.runs.jobLabel)}: </span>
                      {jobLabel(run)}
                    </p>
                    <p className="mt-0.5 text-xs text-app-muted-foreground">
                      {t(strings.runs.nodeLabel)}: {run.nodeId}
                      {run.startedAt
                        ? ` · ${t(strings.runs.startedLabel)} ${formatDate(timestampDate(run.startedAt), { dateStyle: "short", timeStyle: "short" })}`
                        : ""}
                      {duration ? ` · ${t(strings.runs.durationLabel)} ${duration}` : ""}
                    </p>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    data-testid={selectors.runs.view({ id: run.id })}
                    aria-expanded={open}
                    onClick={() => setOpenId(open ? null : run.id)}
                  >
                    {open ? t(strings.runs.close) : t(strings.runs.view)}
                  </Button>
                </div>

                <RunProgress run={run} />

                {open && <RunDetail runId={run.id} />}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

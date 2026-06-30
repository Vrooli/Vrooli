import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import {
  runEval,
  type ClipReportRow,
  type EditOperationRow,
  type EvalReportData,
  type StrategyRow,
} from "../../services/corpus";

const DASH = "—";

function pct(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}
function ratio(value: number): string {
  return value.toFixed(2);
}
function seconds(value: number): string {
  return value.toFixed(1);
}
function ms(value: number): string {
  return String(Math.round(value));
}
function deltaPct(value: number): string {
  if (Math.abs(value) < 0.0005) return "0.0 pp";
  const sign = value > 0 ? "+" : "";
  return `${sign}${(value * 100).toFixed(1)} pp`;
}
function deltaMs(value: number): string {
  if (Math.abs(value) < 0.5) return "0 ms";
  const sign = value > 0 ? "+" : "";
  return `${sign}${Math.round(value)} ms`;
}
function arrayOrEmpty<T>(value: T[] | undefined): T[] {
  return value ?? [];
}

// EvalReportView replays the corpus through every strategy and renders the
// quality-vs-latency comparison table. Latency columns degrade to a dash
// when the run did not pace clips in real time (latency_measured = false).
export function EvalReportView() {
  const { t } = useTranslation();
  const [repeats, setRepeats] = useState(0);

  const run = useMutation({
    mutationFn: () => runEval({ realtimeRepeats: repeats }),
  });

  const report = run.data;

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.reportHint)}</p>

      <div className="flex flex-wrap items-end gap-3">
        <label htmlFor="eval-repeats" className="flex flex-col gap-1 text-xs">
          {t(strings.dictationStudio.repeatsLabel)}
          <Input
            id="eval-repeats"
            data-testid={selectors.dictationStudio.repeatsInput}
            type="number"
            min={0}
            max={20}
            className="w-28"
            value={repeats}
            onChange={(e) => setRepeats(Math.max(0, Math.min(20, Number(e.currentTarget.value) || 0)))}
          />
        </label>
        <Button
          type="button"
          data-testid={selectors.dictationStudio.runEval}
          disabled={run.isPending}
          onClick={() => run.mutate()}
        >
          {run.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              {t(strings.dictationStudio.running)}
            </>
          ) : (
            t(strings.dictationStudio.runEval)
          )}
        </Button>
      </div>
      <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.repeatsHint)}</p>

      {run.isError ? (
        <ApiErrorState error={run.error} title={t(strings.dictationStudio.reportError)} onRetry={() => run.mutate()} />
      ) : null}

      {report ? (
        <EvalReportTable report={report} />
      ) : !run.isError ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.reportEmpty)}</p>
      ) : null}
    </div>
  );
}

export function EvalReportTable({ report }: { report: EvalReportData }) {
  const { t } = useTranslation();
  const latency = report.latencyMeasured;
  const worstClips = report.perStrategy.flatMap((strategy) =>
    strategy.perClip
      .map((clip) => ({ strategy, clip }))
      .filter(({ clip }) => clip.error || clip.wer > 0 || clip.insertions > 0 || clip.deletions > 0 || clip.substitutions > 0),
  )
    .sort((a, b) => {
      if (a.clip.error && !b.clip.error) return -1;
      if (!a.clip.error && b.clip.error) return 1;
      return b.clip.wer - a.clip.wer;
    })
    .slice(0, 8);

  return (
    <div className="flex flex-col gap-4">
      {report.summary ? (
        <section data-testid={selectors.dictationStudio.evalSummary} className="border-y border-app-border py-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="text-sm font-semibold">{t(strings.dictationStudio.recommendationTitle)}</h3>
              <p className="text-sm text-app-foreground">{report.summary.recommendation}</p>
            </div>
            <span className="rounded border border-app-border px-2 py-1 text-xs text-app-muted-foreground">
              {t(strings.dictationStudio.confidenceLabel)}: {report.summary.confidence}
            </span>
          </div>
          <ul className="mt-2 list-disc pl-5 text-xs text-app-muted-foreground">
            {report.summary.reasons.map((reason) => (
              <li key={reason}>{reason}</li>
            ))}
            {report.summary.confidenceNotes.map((note) => (
              <li key={note}>{note}</li>
            ))}
          </ul>
        </section>
      ) : null}

      {!report.qualityMeasured ? (
        <p className="text-xs text-app-warning">{t(strings.dictationStudio.qualityNotMeasured)}</p>
      ) : null}
      {!latency ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.latencyNotMeasured)}</p>
      ) : null}
      {report.warnings.length > 0 ? (
        <div className="flex flex-col gap-1">
          {report.warnings.map((warning) => (
            <p key={`${warning.code}-${warning.message}`} className="text-xs text-app-warning">
              {warning.severity ? `${warning.severity}: ` : ""}{warning.code ? `${warning.code} — ` : ""}{warning.message}
            </p>
          ))}
        </div>
      ) : null}
      {report.latencyHonesty ? (
        <p className="text-xs text-app-muted-foreground">{report.latencyHonesty}</p>
      ) : null}

      {report.perStrategy.some((row) => row.safety || row.stageAttribution || arrayOrEmpty(row.lengthCurves).length > 0) ? (
        <section className="grid gap-3 border-y border-app-border py-3 md:grid-cols-3">
          {report.perStrategy.map((row) => (
            <div key={`${row.strategy}-safety`} className="rounded-control border border-app-border p-3">
              <div className="flex items-center justify-between gap-2">
                <h3 className="text-sm font-semibold">{row.label}</h3>
                <span className={row.safety?.passed ? "text-xs text-app-success" : "text-xs text-app-danger"}>
                  {row.safety
                    ? (row.safety.passed ? t(strings.dictationStudio.safetySafe) : t(strings.dictationStudio.safetyUnsafe))
                    : t(strings.dictationStudio.safetyMeasured)}
                </span>
              </div>
              {row.safety ? (
                <dl className="mt-2 grid grid-cols-2 gap-2 text-xs text-app-muted-foreground">
                  <div>
                    <dt>{t(strings.dictationStudio.safetyRetractions)}</dt>
                    <dd className="font-medium text-app-foreground">{row.safety.retractionFree ? "0" : String(row.safety.retractionEvents.length)}</dd>
                  </div>
                  <div>
                    <dt>{t(strings.dictationStudio.safetyMaxDrop)}</dt>
                    <dd className="font-medium text-app-foreground">
                      {row.safety.maxDroppedSpanWords}/{row.safety.droppedSpanThresholdWords}
                    </dd>
                  </div>
                </dl>
              ) : null}
              {row.stageAttribution ? (
                <p className="mt-2 text-xs text-app-muted-foreground">
                  {t(strings.dictationStudio.stageIngress)} {row.stageAttribution.ingressLostWords} · {t(strings.dictationStudio.stageStrategy)} {row.stageAttribution.strategyLostWords} · {t(strings.dictationStudio.stageEgress)} {row.stageAttribution.egressLostWords}
                </p>
              ) : null}
              {arrayOrEmpty(row.lengthCurves).length > 0 ? (
                <div className="mt-2 flex flex-wrap gap-1">
                  {arrayOrEmpty(row.lengthCurves).map((curve) => (
                    <span key={curve.bucket} className="rounded border border-app-border px-1.5 py-0.5 text-xs text-app-muted-foreground">
                      {curve.bucket}: {curve.clipCount} {t(strings.dictationStudio.clipsShort)} · {t(strings.dictationStudio.colWer)} {pct(curve.wer)} · {t(strings.dictationStudio.colP95)} {latency ? ms(curve.finalizationLatencyP95Ms) : DASH} · {t(strings.dictationStudio.ttfcShort)} {latency ? ms(curve.meanTimeToFirstCommitMs) : DASH}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>
          ))}
        </section>
      ) : null}

      <details className="border-y border-app-border py-2">
        <summary className="cursor-pointer text-xs font-medium text-app-muted-foreground">
          {t(strings.dictationStudio.metricGlossary)}
        </summary>
        <dl className="mt-2 grid gap-2 text-xs text-app-muted-foreground sm:grid-cols-2">
          <GlossaryTerm term={t(strings.dictationStudio.colWer)} text={t(strings.dictationStudio.glossaryWer)} />
          <GlossaryTerm term={t(strings.dictationStudio.colRtf)} text={t(strings.dictationStudio.glossaryRtf)} />
          <GlossaryTerm term={t(strings.dictationStudio.colWhisperCalls)} text={t(strings.dictationStudio.glossaryCalls)} />
          <GlossaryTerm term={t(strings.dictationStudio.colP95)} text={t(strings.dictationStudio.glossaryLatency)} />
          <GlossaryTerm term={t(strings.dictationStudio.colRevisions)} text={t(strings.dictationStudio.glossaryRevisions)} />
          <GlossaryTerm term={t(strings.dictationStudio.editBreakdown)} text={t(strings.dictationStudio.glossaryEdits)} />
        </dl>
        {report.normalizationPolicy ? (
          <div className="mt-3 space-y-1 text-xs text-app-muted-foreground">
            <p>{report.normalizationPolicy.werPolicy}</p>
            <p>{report.normalizationPolicy.overlapAgreementPolicy}</p>
          </div>
        ) : null}
      </details>

      <Table data-testid={selectors.dictationStudio.evalTable}>
        <THead>
          <TR>
            <TH>{t(strings.dictationStudio.colStrategy)}</TH>
            <TH>{t(strings.dictationStudio.colWer)}</TH>
            <TH>{t(strings.dictationStudio.colWerDelta)}</TH>
            <TH>{t(strings.dictationStudio.editBreakdown)}</TH>
            <TH>{t(strings.dictationStudio.colRtf)}</TH>
            <TH>{t(strings.dictationStudio.colWhisperCalls)}</TH>
            <TH>{t(strings.dictationStudio.colAudioSeconds)}</TH>
            <TH>{t(strings.dictationStudio.colP50)}</TH>
            <TH>{t(strings.dictationStudio.colP95Delta)}</TH>
            <TH>{t(strings.dictationStudio.colP95)}</TH>
            <TH>{t(strings.dictationStudio.colRevisions)}</TH>
            <TH>{t(strings.dictationStudio.colVerdict)}</TH>
          </TR>
        </THead>
        <TBody>
          {report.perStrategy.map((row) => (
            <TR key={row.strategy} data-testid={selectors.dictationStudio.evalRow({ strategy: row.strategy })}>
              <TD className="font-medium">{row.label}</TD>
              <TD>{report.qualityMeasured ? pct(row.wer) : DASH}</TD>
              <TD>{report.qualityMeasured ? deltaPct(row.werDeltaVsWinner) : DASH}</TD>
              <TD>{row.substitutions}/{row.insertions}/{row.deletions}</TD>
              <TD>{ratio(row.rtf)}</TD>
              <TD>{String(row.whisperCalls)}</TD>
              <TD>{seconds(row.whisperAudioSeconds)}</TD>
              <TD>{latency ? ms(row.finalizationLatencyP50Ms) : DASH}</TD>
              <TD>{latency ? deltaMs(row.p95DeltaMsVsWinner) : DASH}</TD>
              <TD>{latency ? ms(row.finalizationLatencyP95Ms) : DASH}</TD>
              <TD>{String(row.partialRevisions)}</TD>
              <TD>
                <StrategyVerdict row={row} />
              </TD>
            </TR>
          ))}
        </TBody>
      </Table>

      {worstClips.length > 0 ? (
        <section data-testid={selectors.dictationStudio.evalClips} className="border-t border-app-border pt-3">
          <h3 className="text-sm font-semibold">{t(strings.dictationStudio.clipDrilldown)}</h3>
          <div className="mt-2 flex flex-col gap-2">
            {worstClips.map(({ strategy, clip }) => (
              <ClipDetails key={`${strategy.strategy}-${clip.clipId}`} strategy={strategy} clip={clip} latency={latency} />
            ))}
          </div>
        </section>
      ) : null}
    </div>
  );
}

function GlossaryTerm({ term, text }: { term: string; text: string }) {
  return (
    <div>
      <dt className="font-medium text-app-foreground">{term}</dt>
      <dd>{text}</dd>
    </div>
  );
}

function StrategyVerdict({ row }: { row: StrategyRow }) {
  return (
    <details className="min-w-36">
      <summary className="cursor-pointer text-xs font-medium">{row.verdict || "measured"}</summary>
      <ul className="mt-1 list-disc pl-4 text-xs text-app-muted-foreground">
        {row.reasons.map((reason) => (
          <li key={reason}>{reason}</li>
        ))}
        {row.warnings.map((warning) => (
          <li key={`${warning.code}-${warning.message}`}>{warning.message}</li>
        ))}
      </ul>
    </details>
  );
}

function ClipDetails({ strategy, clip, latency }: { strategy: StrategyRow; clip: ClipReportRow; latency: boolean }) {
  const { t } = useTranslation();
  return (
    <details
      data-testid={selectors.dictationStudio.evalClip({ strategy: strategy.strategy, clipId: clip.clipId })}
      className="border-b border-app-border pb-2"
    >
      <summary className="cursor-pointer text-sm">
        <span className="font-medium">{strategy.label}</span> · {clip.clipId} · {pct(clip.wer)}
        {clip.error ? <span className="text-app-danger"> · {clip.error}</span> : null}
      </summary>
      <div className="mt-2 grid gap-3 text-xs md:grid-cols-2">
        <TranscriptBlock title={t(strings.dictationStudio.referenceLabel)} text={clip.reference} />
        <TranscriptBlock title={t(strings.dictationStudio.hypothesisLabel)} text={clip.hypothesis || DASH} />
        <TranscriptBlock title={t(strings.dictationStudio.normalizedReferenceLabel)} text={clip.normalizedReference || DASH} />
        <TranscriptBlock title={t(strings.dictationStudio.normalizedHypothesisLabel)} text={clip.normalizedHypothesis || DASH} />
      </div>
      <div className="mt-2 flex flex-wrap gap-2 text-xs text-app-muted-foreground">
        <span>{t(strings.dictationStudio.editBreakdown)}: {clip.substitutions}/{clip.insertions}/{clip.deletions}</span>
        <span>{t(strings.dictationStudio.colWhisperCalls)}: {clip.whisperCalls}</span>
        <span>{t(strings.dictationStudio.colRevisions)}: {clip.partialRevisions}</span>
        <span>{t(strings.dictationStudio.colP95)}: {latency ? ms(clip.finalizationLatencyP95Ms) : DASH}</span>
      </div>
      {clip.editOperations.length > 0 ? (
        <div className="mt-2 flex flex-wrap gap-1" aria-label={t(strings.dictationStudio.wordDiffLabel)}>
          {clip.editOperations.map((op, index) => (
            <DiffToken key={`${op.kind}-${index}`} op={op} />
          ))}
        </div>
      ) : null}
    </details>
  );
}

function TranscriptBlock({ title, text }: { title: string; text: string }) {
  return (
    <div>
      <div className="font-medium text-app-foreground">{title}</div>
      <p className="mt-1 whitespace-pre-wrap break-words text-app-muted-foreground">{text}</p>
    </div>
  );
}

function DiffToken({ op }: { op: EditOperationRow }) {
  const text = op.kind === "insertion" ? op.hypothesisToken : op.kind === "deletion" ? op.referenceToken : op.hypothesisToken || op.referenceToken;
  const tone =
    op.kind === "match"
      ? "border-app-border text-app-muted-foreground"
      : op.kind === "substitution"
        ? "border-app-warning text-app-warning"
        : op.kind === "insertion"
          ? "border-app-success text-app-success"
          : "border-app-danger text-app-danger";
  return (
    <span className={`rounded border px-1.5 py-0.5 text-xs ${tone}`} title={op.kind}>
      {text || op.kind}
    </span>
  );
}

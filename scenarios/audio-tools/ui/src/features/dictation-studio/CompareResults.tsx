import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { type ExperimentReportRow, type ExperimentRow } from "../../services/experiment";
import { ms, pct } from "./ExperimentLabFormat";
import { StatusBadge, StrategyName } from "./ExperimentLabShared";

function recipeDiffLines(rows: ExperimentReportRow[]): string[] {
  if (rows.length < 2) return [];
  const maps = rows.map((row) => recipeFields(row.experiment?.recipe));
  const keys = Array.from(new Set(maps.flatMap((map) => Object.keys(map)))).sort();
  return keys.flatMap((key) => {
    const first = maps[0]?.[key] ?? "";
    if (maps.every((map) => (map[key] ?? "") === first)) return [];
    if (rows.length === 2) return [`${key}: ${valueOrDash(maps[0]?.[key])} -> ${valueOrDash(maps[1]?.[key])}`];
    return [`${key}: ${maps.map((map, index) => `${shortExperimentLabel(rows[index]?.experiment)}=${valueOrDash(map[key])}`).join(", ")}`];
  });
}

function recipeFields(recipe?: ExperimentRow["recipe"]): Record<string, string> {
  if (!recipe) return {};
  const fields: Record<string, string> = {
    clip_ids: recipe.clipIds.join(","),
    realtime_repeats: String(recipe.realtimeRepeats),
    chunk_ms: String(recipe.chunkMs),
    seed: String(recipe.seed),
    latency_tail_seconds: String(recipe.latencyTailSeconds),
    dropped_span_threshold_words: String(recipe.droppedSpanThresholdWords),
    "long_form.enabled": String(recipe.longFormEnabled),
    "long_form.target_duration_seconds": String(recipe.targetDurationSeconds),
    "long_form.gap_ms": String(recipe.gapMs),
    "long_form.tag_contains": recipe.tagContains,
    "long_form.sweep_durations_seconds": recipe.sweepDurationsSeconds.join(","),
    "augmentation.noise_types": recipe.noiseTypes.join(","),
    "augmentation.snr_db": recipe.snrDb.join(","),
    "augmentation.competing_voice_ids": recipe.competingVoiceIds.join(","),
    "augmentation.competing_text": recipe.competingText,
    "speaker.target_profile_id": recipe.speakerTargetProfileId,
    "speaker.extraction_enabled": String(recipe.speakerExtraction),
    "speaker.verification_enabled": String(recipe.speakerVerification),
    "speaker.verification_mode": recipe.speakerMode,
    "speaker.threshold": String(recipe.speakerThreshold),
    "speaker.ablation_enabled": String(recipe.speakerAblation),
  };
  for (const strategy of recipe.strategyDetails ?? []) {
    const key = `strategy.${strategy.kind || strategy.label || "unknown"}`;
    fields[`${key}.overlap_max_window_ms`] = String(strategy.overlapMaxWindowMs);
    fields[`${key}.overlap_max_stall_rejects`] = String(strategy.overlapMaxStallRejects);
    fields[`${key}.overlap_window_ms`] = String(strategy.overlapWindowMs);
    fields[`${key}.overlap_commit_runs`] = String(strategy.overlapCommitRuns);
    fields[`${key}.vad_silence_ms`] = String(strategy.vadSilenceMs);
  }
  return fields;
}

function alignedStrategyKeys(rows: ExperimentReportRow[]): string[] {
  return Array.from(new Set(rows.flatMap((row) => row.report.perStrategy.map(strategyAlignmentKey)))).sort();
}

function strategyMetricCell(row: ExperimentReportRow, key: string): string {
  const strategy = row.report.perStrategy.find((item) => strategyAlignmentKey(item) === key);
  if (!strategy) return "-";
  const p95 = row.report.latencyMeasured ? ms(strategy.finalizationLatencyP95Ms) : "-";
  return `${pct(strategy.wer)} / ${p95}`;
}

function strategyAlignmentKey(strategy: ExperimentReportRow["report"]["perStrategy"][number]): string {
  return strategy.strategy.split("/")[0] || strategy.label.split("/")[0] || "-";
}

function shortExperimentLabel(row: ExperimentRow | null | undefined): string {
  return row?.name || row?.id || "-";
}

function valueOrDash(value: string | undefined): string {
  return value && value.trim() ? value : "-";
}

export function CompareResults({ rows }: { rows: ExperimentReportRow[] }) {
  const { t } = useTranslation();
  const recipeDiffs = recipeDiffLines(rows);
  const strategyKeys = alignedStrategyKeys(rows);
  return (
    <div data-testid={selectors.dictationStudio.compareResults} className="space-y-4 overflow-x-auto">
      <div className="min-w-full">
        <Table>
          <THead>
            <TR>
              <TH>{t(strings.dictationStudio.colName)}</TH>
              <TH>{t(strings.dictationStudio.colStatus)}</TH>
              <TH>{t(strings.dictationStudio.compareColWinner)}</TH>
              <TH>{t(strings.dictationStudio.colWer)}</TH>
              <TH>{t(strings.dictationStudio.compareColP95)}</TH>
              <TH>{t(strings.dictationStudio.compareColSafety)}</TH>
            </TR>
          </THead>
          <TBody>
            {rows.map((row) => {
              const winner = row.report.summary?.winnerStrategy;
              const strategy = row.report.perStrategy.find((item) => item.strategy === winner) ?? row.report.perStrategy[0];
              const latency = row.report.latencyMeasured;
              const quality = row.report.qualityMeasured;
              const unsafeWinner = strategy?.safety?.passed === false;
              return (
                <TR key={row.experiment?.id ?? row.report.summary?.recommendation ?? "compare-row"}>
                  <TD>{row.experiment?.name ?? row.experiment?.id ?? t(strings.common.dash)}</TD>
                  <TD>{row.experiment ? <StatusBadge status={row.experiment.status} /> : t(strings.common.dash)}</TD>
                  <TD className={unsafeWinner ? "text-app-warning" : undefined}>
                    {strategy ? <StrategyName kind={strategy.strategy} /> : t(strings.common.dash)}
                    {unsafeWinner ? <div className="text-xs">{t(strings.dictationStudio.recommendedDespiteSafetyFailure)}</div> : null}
                  </TD>
                  <TD>{strategy && quality ? pct(strategy.wer) : t(strings.common.dash)}</TD>
                  <TD>{strategy && latency ? ms(strategy.finalizationLatencyP95Ms) : t(strings.common.dash)}</TD>
                  <TD>
                    {strategy?.safety ? (
                      <span className={strategy.safety.passed ? "text-app-success" : "text-app-danger"}>
                        {strategy.safety.passed ? t(strings.dictationStudio.safetySafe) : t(strings.dictationStudio.safetyUnsafe)}
                      </span>
                    ) : (
                      t(strings.dictationStudio.safetyMeasured)
                    )}
                  </TD>
                </TR>
              );
            })}
          </TBody>
        </Table>
      </div>
      {recipeDiffs.length > 0 ? (
        <div className="rounded-control border border-app-border p-3">
          <h3 className="text-sm font-semibold">{t(strings.dictationStudio.compareRecipeDiffTitle)}</h3>
          <ul className="mt-2 space-y-1 text-xs text-app-muted-foreground">
            {recipeDiffs.map((line) => (
              <li key={line} className="break-all">
                {line}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {strategyKeys.length > 0 ? (
        <div>
          <h3 className="mb-2 text-sm font-semibold">{t(strings.dictationStudio.compareStrategyAlignmentTitle)}</h3>
          <Table>
            <THead>
              <TR>
                <TH>{t(strings.dictationStudio.colStrategy)}</TH>
                {rows.map((row) => (
                  <TH key={row.experiment?.id ?? shortExperimentLabel(row.experiment)}>{shortExperimentLabel(row.experiment)}</TH>
                ))}
              </TR>
            </THead>
            <TBody>
              {strategyKeys.map((key) => (
                <TR key={key}>
                  <TD>
                    <StrategyName kind={key} />
                  </TD>
                  {rows.map((row) => (
                    <TD key={`${key}-${row.experiment?.id ?? shortExperimentLabel(row.experiment)}`}>{strategyMetricCell(row, key)}</TD>
                  ))}
                </TR>
              ))}
            </TBody>
          </Table>
        </div>
      ) : null}
    </div>
  );
}

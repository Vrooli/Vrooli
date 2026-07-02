import { createClient } from "@connectrpc/connect";

import { SpeakerMode } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import {
  ExperimentService,
  type ExperimentEvent,
  ExperimentStatus,
  type Experiment,
  type ExperimentRecipe,
  type ExperimentRun,
} from "@vrooli/proto-types/audio-tools/v1/experiment/experiment_pb";

import { transport } from "../api/client";
import { decodeEvalReport, type EvalReportData } from "./corpus";

const client = createClient(ExperimentService, transport);

function tsToISO(ts?: { seconds?: bigint | number; nanos?: number }): string {
  if (!ts) return "";
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : (ts.seconds ?? 0);
  return new Date(seconds * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000)).toISOString();
}

function csv(value: string): string[] {
  return value
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean);
}

function csvNumbers(value: string): number[] {
  return csv(value)
    .map((part) => Number(part))
    .filter((n) => Number.isFinite(n));
}

function statusLabel(status: ExperimentStatus): ExperimentStatusLabel {
  switch (status) {
    case ExperimentStatus.QUEUED:
      return "queued";
    case ExperimentStatus.RUNNING:
      return "running";
    case ExperimentStatus.SUCCEEDED:
      return "succeeded";
    case ExperimentStatus.FAILED:
      return "failed";
    case ExperimentStatus.CANCELED:
      return "canceled";
    default:
      return "unspecified";
  }
}

function speakerMode(value: "off" | "filter" | "advisory"): SpeakerMode {
  switch (value) {
    case "off":
      return SpeakerMode.OFF;
    case "filter":
      return SpeakerMode.FILTER;
    case "advisory":
      return SpeakerMode.ADVISORY;
    default:
      return SpeakerMode.UNSPECIFIED;
  }
}

export type ExperimentStatusLabel = "unspecified" | "queued" | "running" | "succeeded" | "failed" | "canceled";

export interface ExperimentRow {
  id: string;
  name: string;
  status: ExperimentStatusLabel;
  createdAt: string;
  startedAt: string;
  finishedAt: string;
  error: string;
  resultRef: string;
  machineJson: string;
  recipe: RecipeSummary;
}

export interface RecipeSummary {
  clipIds: string[];
  strategies: string[];
  strategyDetails?: Array<{
    kind: string;
    label: string;
    overlapMaxWindowMs: number;
    overlapMaxStallRejects: number;
    overlapWindowMs: number;
    overlapCommitRuns: number;
    vadSilenceMs: number;
  }>;
  realtimeRepeats: number;
  latencyTailSeconds: number;
  chunkMs: number;
  seed: number;
  longFormEnabled: boolean;
  targetDurationSeconds: number;
  gapMs: number;
  tagContains: string;
  sweepDurationsSeconds: number[];
  realizedClipIds: string[];
  realizedDurationMs: number;
  realizedReference: string;
  noiseTypes: string[];
  snrDb: number[];
  competingVoiceIds: string[];
  competingText: string;
  augmentationConditions: Array<{ id: string; kind: string; source: string; snrDb: number; skipped: boolean; note: string }>;
  speakerTargetProfileId: string;
  speakerExtraction: boolean;
  speakerVerification: boolean;
  speakerMode: string;
  speakerThreshold: number;
  speakerAblation: boolean;
  speakerConditions: Array<{ id: string; extraction: boolean; verification: boolean; skipped: boolean; note: string }>;
  droppedSpanThresholdWords: number;
}

export interface ExperimentRunRow {
  id: string;
  experimentId: string;
  strategy: string;
  conditionJson: string;
  createdAt: string;
}

export interface ExperimentEventRow {
  experimentId: string;
  status: ExperimentStatusLabel;
  progress: number;
  message: string;
  at: string;
}

export interface ExperimentReportRow {
  experiment: ExperimentRow | null;
  report: EvalReportData;
  runs: ExperimentRunRow[];
}

export interface StartExperimentInput {
  name: string;
  clipIds: string[];
  strategies: string[];
  realtimeRepeats: number;
  latencyTailSeconds: number;
  chunkMs: number;
  seed: number;
  longForm: boolean;
  targetDurationSeconds: number;
  gapMs: number;
  tagContains: string;
  sweepDurationsCsv: string;
  overlapMaxStallRejects: number;
  overlapWindowMs: number;
  overlapCommitRuns: number;
  overlapMaxWindowMs: number;
  vadSilenceMs: number;
  noiseTypesCsv: string;
  snrDbCsv: string;
  competingVoicesCsv: string;
  competingText: string;
  speakerTargetProfileId: string;
  speakerExtraction: boolean;
  speakerVerification: boolean;
  speakerMode: "off" | "filter" | "advisory";
  speakerThreshold: number;
  speakerFallback: boolean;
  speakerAblation: boolean;
  droppedSpanThresholdWords: number;
}

function decodeRecipe(r?: ExperimentRecipe): RecipeSummary {
  const speakerModeLabel = r?.speaker
    ? SpeakerMode[r.speaker.verificationMode].toLowerCase()
    : "unspecified";
  return {
    clipIds: r?.clipIds ?? [],
    strategies: (r?.strategies ?? []).map((s) => s.kind || s.label).filter(Boolean),
    strategyDetails: (r?.strategies ?? []).map((s) => ({
      kind: s.kind,
      label: s.label,
      overlapMaxWindowMs: s.overlapMaxWindowMs,
      overlapMaxStallRejects: s.overlapMaxStallRejects,
      overlapWindowMs: s.overlapWindowMs,
      overlapCommitRuns: s.overlapCommitRuns,
      vadSilenceMs: s.vadSilenceMs,
    })),
    realtimeRepeats: r?.realtimeRepeats ?? 0,
    latencyTailSeconds: r?.latencyTailSeconds ?? 0,
    chunkMs: r?.chunkMs ?? 0,
    seed: Number(r?.seed ?? 0),
    longFormEnabled: Boolean(r?.longForm?.enabled),
    targetDurationSeconds: r?.longForm?.targetDurationSeconds ?? 0,
    gapMs: r?.longForm?.gapMs ?? 0,
    tagContains: r?.longForm?.tagContains ?? "",
    sweepDurationsSeconds: r?.longForm?.sweepDurationsSeconds ?? [],
    realizedClipIds: r?.realizedClipIds ?? [],
    realizedDurationMs: Number(r?.realizedDurationMs ?? 0),
    realizedReference: r?.realizedReference ?? "",
    noiseTypes: r?.augmentation?.noiseTypes ?? [],
    snrDb: r?.augmentation?.snrDb ?? [],
    competingVoiceIds: r?.augmentation?.competingVoiceIds ?? [],
    competingText: r?.augmentation?.competingText ?? "",
    augmentationConditions: (r?.realizedAugmentationConditions ?? []).map((c) => ({
      id: c.id,
      kind: c.kind,
      source: c.source,
      snrDb: c.snrDb,
      skipped: c.skipped,
      note: c.note,
    })),
    speakerTargetProfileId: r?.speaker?.targetProfileId ?? "",
    speakerExtraction: Boolean(r?.speaker?.extractionEnabled),
    speakerVerification: Boolean(r?.speaker?.verificationEnabled),
    speakerMode: speakerModeLabel,
    speakerThreshold: r?.speaker?.threshold ?? 0,
    speakerAblation: Boolean(r?.speaker?.ablationEnabled),
    speakerConditions: (r?.realizedSpeakerConditions ?? []).map((c) => ({
      id: c.id,
      extraction: c.extractionEnabled,
      verification: c.verificationEnabled,
      skipped: c.skipped,
      note: c.note,
    })),
    droppedSpanThresholdWords: r?.droppedSpanThresholdWords ?? 0,
  };
}

function decodeExperiment(e?: Experiment): ExperimentRow | null {
  if (!e) return null;
  return {
    id: e.id,
    name: e.name,
    status: statusLabel(e.status),
    createdAt: tsToISO(e.createdAt),
    startedAt: tsToISO(e.startedAt),
    finishedAt: tsToISO(e.finishedAt),
    error: e.error,
    resultRef: e.resultRef,
    machineJson: e.machineJson,
    recipe: decodeRecipe(e.recipe),
  };
}

function decodeRun(r: ExperimentRun): ExperimentRunRow {
  return {
    id: r.id,
    experimentId: r.experimentId,
    strategy: r.strategy,
    conditionJson: r.conditionJson,
    createdAt: tsToISO(r.createdAt),
  };
}

function decodeEvent(e: ExperimentEvent): ExperimentEventRow {
  return {
    experimentId: e.experimentId,
    status: statusLabel(e.status),
    progress: e.progress,
    message: e.message,
    at: tsToISO(e.at),
  };
}

function buildRecipe(input: StartExperimentInput) {
  const strategies = input.strategies.map((kind) => ({
    kind,
    label: kind,
    overlapMaxStallRejects: input.overlapMaxStallRejects,
    overlapWindowMs: input.overlapWindowMs,
    overlapCommitRuns: input.overlapCommitRuns,
    overlapMaxWindowMs: input.overlapMaxWindowMs,
    vadSilenceMs: input.vadSilenceMs,
  }));
  const sweepDurationsSeconds = csvNumbers(input.sweepDurationsCsv)
    .map((n) => Math.max(0, Math.round(n)))
    .filter((n) => n > 0);
  return {
    clipIds: input.clipIds,
    strategies,
    realtimeRepeats: input.realtimeRepeats,
    latencyTailSeconds: input.latencyTailSeconds,
    chunkMs: input.chunkMs,
    seed: BigInt(input.seed),
    longForm: {
      // A length sweep implies long-form synthesis, so enabling it (or
      // listing any sweep durations) flips the recipe into long-form mode.
      enabled: input.longForm || sweepDurationsSeconds.length > 0,
      targetDurationSeconds: input.targetDurationSeconds,
      gapMs: input.gapMs,
      tagContains: input.tagContains,
      sweepDurationsSeconds,
    },
    realizedClipIds: [],
    realizedReference: "",
    realizedDurationMs: 0n,
    augmentation: {
      noiseTypes: csv(input.noiseTypesCsv),
      snrDb: csvNumbers(input.snrDbCsv),
      competingVoiceIds: csv(input.competingVoicesCsv),
      competingText: input.competingText,
    },
    realizedAugmentationConditions: [],
    speaker: {
      targetProfileId: input.speakerTargetProfileId,
      extractionEnabled: input.speakerExtraction,
      verificationEnabled: input.speakerVerification,
      verificationMode: speakerMode(input.speakerMode),
      threshold: input.speakerThreshold,
      fallbackWithoutVerification: input.speakerFallback,
      ablationEnabled: input.speakerAblation,
    },
    realizedSpeakerConditions: [],
    droppedSpanThresholdWords: input.droppedSpanThresholdWords,
  };
}

export async function startExperiment(input: StartExperimentInput): Promise<ExperimentRow> {
  const resp = await client.startExperiment({
    name: input.name,
    recipe: buildRecipe(input),
    estimatedSeconds: 0,
  });
  const row = decodeExperiment(resp.experiment);
  if (!row) throw new Error("start experiment response missing experiment");
  return row;
}

export async function listExperiments(): Promise<ExperimentRow[]> {
  const resp = await client.listExperiments({ status: ExperimentStatus.UNSPECIFIED, limit: 50, offset: 0 });
  return resp.experiments.map((e) => decodeExperiment(e)).filter((e): e is ExperimentRow => e !== null);
}

export async function getExperiment(id: string): Promise<{ experiment: ExperimentRow | null; runs: ExperimentRunRow[] }> {
  const resp = await client.getExperiment({ id });
  return { experiment: decodeExperiment(resp.experiment), runs: resp.runs.map(decodeRun) };
}

export async function waitExperiment(id: string): Promise<{ experiment: ExperimentRow | null; runs: ExperimentRunRow[] }> {
  const resp = await client.waitExperiment({ id });
  return { experiment: decodeExperiment(resp.experiment), runs: resp.runs.map(decodeRun) };
}

export async function cancelExperiment(id: string): Promise<ExperimentRow | null> {
  const resp = await client.cancelExperiment({ id });
  return decodeExperiment(resp.experiment);
}

export async function getExperimentReport(id: string): Promise<ExperimentReportRow> {
  const resp = await client.getExperimentReport({ id });
  return {
    experiment: decodeExperiment(resp.experiment),
    report: decodeEvalReport(resp.report),
    runs: resp.runs.map(decodeRun),
  };
}

export async function compareExperiments(ids: string[]): Promise<ExperimentReportRow[]> {
  const resp = await client.compareExperiments({ ids });
  return resp.experiments.map((row) => ({
    experiment: decodeExperiment(row.experiment),
    report: decodeEvalReport(row.report),
    runs: ((row as { runs?: ExperimentRun[] }).runs ?? []).map(decodeRun),
  }));
}

export async function streamExperimentEvents(
  id: string,
  onEvent: (event: ExperimentEventRow) => void,
  signal?: AbortSignal,
): Promise<void> {
  for await (const event of client.streamExperimentEvents({ id }, { signal })) {
    onEvent(decodeEvent(event));
  }
}

// Corpus + Eval client: calls audio-tools' own CorpusService and
// EvalService over the same-origin Connect transport. No cross-scenario
// calls — audio-tools owns these surfaces and serves them to its own UI.
//
// Mirrors the thin-wrapper pattern in speakerAdmin.ts / settings.ts: the
// generated Connect clients stay private to this module and every export
// returns a plain, UI-friendly shape (numbers not bigints, ISO strings not
// Timestamps) so feature components never touch protobuf wire types.

import { createClient } from "@connectrpc/connect";

import {
  CorpusService,
  ClipSource,
  type Clip,
} from "@vrooli/proto-types/audio-tools/v1/corpus/corpus_pb";
import {
  EvalService,
  type EditOperation,
  type EvalReportSummary,
  type NormalizationPolicy,
  type ReportWarning,
  type StrategyReport,
} from "@vrooli/proto-types/audio-tools/v1/eval/eval_pb";

import { transport } from "../api/client";

const corpusClient = createClient(CorpusService, transport);
const evalClient = createClient(EvalService, transport);

export { ClipSource };

function tsToISO(ts?: { seconds?: bigint | number; nanos?: number }): string {
  if (!ts) return "";
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : (ts.seconds ?? 0);
  return new Date(seconds * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000)).toISOString();
}

export interface ClipMeta {
  id: string;
  referenceText: string;
  tags: string[];
  durationMs: number;
  sampleRateHz: number;
  format: string;
  source: ClipSource;
  createdAt: string;
}

function decodeClip(c: Clip): ClipMeta {
  return {
    id: c.id,
    referenceText: c.referenceText,
    tags: c.tags,
    durationMs: Number(c.durationMs),
    sampleRateHz: c.sampleRateHz,
    format: c.format,
    source: c.source,
    createdAt: tsToISO(c.createdAt),
  };
}

export interface CreateClipArgs {
  audio: Uint8Array;
  referenceText: string;
  tags: string[];
  durationMs: number;
  sampleRateHz: number;
  format: string;
  source: ClipSource;
}

export async function createClip(args: CreateClipArgs): Promise<ClipMeta> {
  const resp = await corpusClient.createClip({
    audio: args.audio,
    referenceText: args.referenceText,
    tags: args.tags,
    durationMs: BigInt(Math.max(0, Math.round(args.durationMs))),
    sampleRateHz: args.sampleRateHz,
    format: args.format,
    source: args.source,
  });
  if (!resp.clip) throw new Error("create clip response missing clip");
  return decodeClip(resp.clip);
}

export interface ListClipsArgs {
  tagContains?: string;
  limit?: number;
  offset?: number;
}

export async function listClips(args: ListClipsArgs = {}): Promise<ClipMeta[]> {
  const resp = await corpusClient.listClips({
    tagContains: args.tagContains ?? "",
    limit: args.limit ?? 0,
    offset: args.offset ?? 0,
  });
  return resp.clips.map(decodeClip);
}

export async function deleteClip(id: string): Promise<void> {
  await corpusClient.deleteClip({ id });
}

export async function getClipAudio(id: string): Promise<{ audio: Uint8Array; clip: ClipMeta | null }> {
  const resp = await corpusClient.getClipAudio({ id });
  return { audio: resp.audio, clip: resp.clip ? decodeClip(resp.clip) : null };
}

// =============================================================================
// Eval
// =============================================================================

export interface EvalStrategyInput {
  kind: string;
  label?: string;
  /** -1 = use the persisted default; 0 = disabled. */
  overlapMaxStallRejects?: number;
  overlapWindowMs?: number;
  overlapCommitRuns?: number;
  vadSilenceMs?: number;
}

export interface StrategyRow {
  strategy: string;
  label: string;
  wer: number;
  substitutions: number;
  insertions: number;
  deletions: number;
  refWords: number;
  whisperCalls: number;
  whisperAudioSeconds: number;
  rtf: number;
  finalizationLatencyP50Ms: number;
  finalizationLatencyP95Ms: number;
  partialRevisions: number;
  perClip: ClipReportRow[];
  werDeltaVsWinner: number;
  p95DeltaMsVsWinner: number;
  callMultiplierVsWinner: number;
  verdict: string;
  reasons: string[];
  warnings: WarningRow[];
}

export interface ClipReportRow {
  clipId: string;
  reference: string;
  hypothesis: string;
  wer: number;
  whisperCalls: number;
  whisperAudioSeconds: number;
  rtf: number;
  segmentCount: number;
  partialRevisions: number;
  finalizationLatencyP50Ms: number;
  finalizationLatencyP95Ms: number;
  error: string;
  substitutions: number;
  insertions: number;
  deletions: number;
  refWords: number;
  hypWords: number;
  normalizedReference: string;
  normalizedHypothesis: string;
  editOperations: EditOperationRow[];
}

export interface EditOperationRow {
  kind: string;
  referenceToken: string;
  hypothesisToken: string;
  referenceIndex: number;
  hypothesisIndex: number;
}

export interface WarningRow {
  code: string;
  message: string;
  severity: string;
}

export interface ReportSummaryRow {
  winnerStrategy: string;
  winnerLabel: string;
  recommendation: string;
  confidence: string;
  reasons: string[];
  confidenceNotes: string[];
}

export interface NormalizationPolicyRow {
  werPolicy: string;
  overlapAgreementPolicy: string;
}

export interface EvalReportData {
  perStrategy: StrategyRow[];
  qualityMeasured: boolean;
  latencyMeasured: boolean;
  summary: ReportSummaryRow | null;
  warnings: WarningRow[];
  normalizationPolicy: NormalizationPolicyRow | null;
}

function decodeWarning(w: ReportWarning): WarningRow {
  return { code: w.code, message: w.message, severity: w.severity };
}

function decodeSummary(s?: EvalReportSummary): ReportSummaryRow | null {
  if (!s) return null;
  return {
    winnerStrategy: s.winnerStrategy,
    winnerLabel: s.winnerLabel,
    recommendation: s.recommendation,
    confidence: s.confidence,
    reasons: s.reasons,
    confidenceNotes: s.confidenceNotes,
  };
}

function decodePolicy(p?: NormalizationPolicy): NormalizationPolicyRow | null {
  if (!p) return null;
  return {
    werPolicy: p.werPolicy,
    overlapAgreementPolicy: p.overlapAgreementPolicy,
  };
}

function decodeEditOperation(op: EditOperation): EditOperationRow {
  return {
    kind: op.kind,
    referenceToken: op.referenceToken,
    hypothesisToken: op.hypothesisToken,
    referenceIndex: op.referenceIndex,
    hypothesisIndex: op.hypothesisIndex,
  };
}

function decodeStrategy(s: StrategyReport): StrategyRow {
  return {
    strategy: s.strategy,
    label: s.label || s.strategy,
    wer: s.wer,
    substitutions: s.substitutions,
    insertions: s.insertions,
    deletions: s.deletions,
    refWords: s.refWords,
    whisperCalls: s.whisperCalls,
    whisperAudioSeconds: s.whisperAudioSeconds,
    rtf: s.rtf,
    finalizationLatencyP50Ms: s.finalizationLatencyP50Ms,
    finalizationLatencyP95Ms: s.finalizationLatencyP95Ms,
    partialRevisions: s.partialRevisions,
    perClip: s.perClip.map((c) => ({
      clipId: c.clipId,
      reference: c.reference,
      hypothesis: c.hypothesis,
      wer: c.wer,
      whisperCalls: c.whisperCalls,
      whisperAudioSeconds: c.whisperAudioSeconds,
      rtf: c.rtf,
      segmentCount: c.segmentCount,
      partialRevisions: c.partialRevisions,
      finalizationLatencyP50Ms: c.finalizationLatencyP50Ms,
      finalizationLatencyP95Ms: c.finalizationLatencyP95Ms,
      error: c.error,
      substitutions: c.substitutions,
      insertions: c.insertions,
      deletions: c.deletions,
      refWords: c.refWords,
      hypWords: c.hypWords,
      normalizedReference: c.normalizedReference,
      normalizedHypothesis: c.normalizedHypothesis,
      editOperations: c.editOperations.map(decodeEditOperation),
    })),
    werDeltaVsWinner: s.werDeltaVsWinner,
    p95DeltaMsVsWinner: s.p95DeltaMsVsWinner,
    callMultiplierVsWinner: s.callMultiplierVsWinner,
    verdict: s.verdict,
    reasons: s.reasons,
    warnings: s.warnings.map(decodeWarning),
  };
}

export interface RunEvalArgs {
  clipIds?: string[];
  strategies?: EvalStrategyInput[];
  realtimeRepeats?: number;
  chunkMs?: number;
}

export async function runEval(args: RunEvalArgs = {}): Promise<EvalReportData> {
  const resp = await evalClient.runEval({
    clipIds: args.clipIds ?? [],
    strategies: (args.strategies ?? []).map((s) => ({
      kind: s.kind,
      label: s.label ?? "",
      overlapMaxStallRejects: s.overlapMaxStallRejects ?? -1,
      overlapWindowMs: s.overlapWindowMs ?? 0,
      overlapCommitRuns: s.overlapCommitRuns ?? 0,
      vadSilenceMs: s.vadSilenceMs ?? 0,
    })),
    realtimeRepeats: args.realtimeRepeats ?? 0,
    chunkMs: args.chunkMs ?? 0,
  });
  const report = resp.report;
  return {
    perStrategy: (report?.perStrategy ?? []).map(decodeStrategy),
    qualityMeasured: report?.qualityMeasured ?? false,
    latencyMeasured: report?.latencyMeasured ?? false,
    summary: decodeSummary(report?.summary),
    warnings: (report?.warnings ?? []).map(decodeWarning),
    normalizationPolicy: decodePolicy(report?.normalizationPolicy),
  };
}

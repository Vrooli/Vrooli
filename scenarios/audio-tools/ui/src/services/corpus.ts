// Corpus client: calls audio-tools' own CorpusService over the same-origin
// Connect transport. No cross-scenario calls.
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
  type LengthBucketCurve,
  type SafetyGateReport,
  type StageAttribution,
  type EditOperation,
  type EvalReportSummary,
  type NormalizationPolicy,
  type ReportWarning,
  type EvalReport,
  type StrategyReport,
  type ScalingAnalysis,
  type ScalingModelFit,
  type ScalingPoint,
} from "@vrooli/proto-types/audio-tools/v1/eval/eval_pb";

import { transport } from "../api/client";

const corpusClient = createClient(CorpusService, transport);

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
  safety: SafetyGateRow | null;
  stageAttribution: StageAttributionRow | null;
  lengthCurves: LengthCurveRow[];
  scaling: ScalingAnalysisRow | null;
  commitCount: number;
  speakerRejectionCount: number;
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

export interface SafetyGateRow {
  passed: boolean;
  retractionFree: boolean;
  droppedSpanFree: boolean;
  maxDroppedSpanWords: number;
  droppedSpanThresholdWords: number;
  retractionEvents: Array<{ previousText: string; currentText: string; atMs: number }>;
  reasons: string[];
}

export interface StageAttributionRow {
  ingressLostWords: number;
  strategyLostWords: number;
  egressLostWords: number;
  egressRejectEvents: number;
  notes: string[];
}

export interface LengthCurveRow {
  bucket: string;
  minDurationMs: number;
  maxDurationMs: number;
  clipCount: number;
  wer: number;
  finalizationLatencyP95Ms: number;
  meanTimeToFirstCommitMs: number;
  maxDroppedSpanWords: number;
}

export interface ScalingPointRow {
  clipId: string;
  targetDurationMs: number;
  realizedDurationMs: number;
  wer: number;
  finalizationLatencyP50Ms: number;
  finalizationLatencyP95Ms: number;
  finalizationLatencySampleCount: number;
  timeToFirstCommitMs: number;
  commitCount: number;
  partialRevisions: number;
  maxDroppedSpanWords: number;
  whisperCalls: number;
  whisperAudioSeconds: number;
  providerLatencyMs: number;
  rtf: number;
}

export interface ScalingModelFitRow {
  metric: string;
  unit: string;
  model: string;
  slopePerSecond: number;
  intercept: number;
  rSquared: number;
  sampleCount: number;
  exponent: number;
  exponentRSquared: number;
  reason: string;
}

export interface ScalingAnalysisRow {
  points: ScalingPointRow[];
  latencyClassification: string;
  computeClassification: string;
  confidence: string;
  reasons: string[];
  warnings: WarningRow[];
  latencyFit: ScalingModelFitRow | null;
  computeFit: ScalingModelFitRow | null;
  metricFits: ScalingModelFitRow[];
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
  latencyHonesty: string;
	promotionVerdicts: PromotionVerdictRow[];
}

export interface PromotionVerdictRow {
	engineId: string;
	modelId: string;
	strategy: string;
	policyProfile: string;
	stable: boolean;
	reasons: string[];
}

function decodeWarning(w: ReportWarning): WarningRow {
  return { code: w.code, message: w.message, severity: w.severity };
}

function arrayOrEmpty<T>(value: T[] | undefined): T[] {
  return value ?? [];
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

function decodeSafety(s?: SafetyGateReport): SafetyGateRow | null {
  if (!s) return null;
  return {
    passed: s.passed,
    retractionFree: s.retractionFree,
    droppedSpanFree: s.droppedSpanFree,
    maxDroppedSpanWords: s.maxDroppedSpanWords,
    droppedSpanThresholdWords: s.droppedSpanThresholdWords,
    retractionEvents: s.retractionEvents.map((event) => ({
      previousText: event.previousText,
      currentText: event.currentText,
      atMs: Number(event.atMs),
    })),
    reasons: s.reasons,
  };
}

function decodeStageAttribution(a?: StageAttribution): StageAttributionRow | null {
  if (!a) return null;
  return {
    ingressLostWords: a.ingressLostWords,
    strategyLostWords: a.strategyLostWords,
    egressLostWords: a.egressLostWords,
    egressRejectEvents: a.egressRejectEvents,
    notes: a.notes,
  };
}

function decodeLengthCurve(c: LengthBucketCurve): LengthCurveRow {
  return {
    bucket: c.bucket,
    minDurationMs: Number(c.minDurationMs),
    maxDurationMs: Number(c.maxDurationMs),
    clipCount: c.clipCount,
    wer: c.wer,
    finalizationLatencyP95Ms: c.finalizationLatencyP95Ms,
    meanTimeToFirstCommitMs: c.meanTimeToFirstCommitMs,
    maxDroppedSpanWords: c.maxDroppedSpanWords,
  };
}

function decodeScalingPoint(p: ScalingPoint): ScalingPointRow {
  return {
    clipId: p.clipId,
    targetDurationMs: Number(p.targetDurationMs),
    realizedDurationMs: Number(p.realizedDurationMs),
    wer: p.wer,
    finalizationLatencyP50Ms: p.finalizationLatencyP50Ms,
    finalizationLatencyP95Ms: p.finalizationLatencyP95Ms,
    finalizationLatencySampleCount: p.finalizationLatencySampleCount,
    timeToFirstCommitMs: p.timeToFirstCommitMs,
    commitCount: p.commitCount,
    partialRevisions: p.partialRevisions,
    maxDroppedSpanWords: p.maxDroppedSpanWords,
    whisperCalls: p.whisperCalls,
    whisperAudioSeconds: p.whisperAudioSeconds,
    providerLatencyMs: p.providerLatencyMs,
    rtf: p.rtf,
  };
}

function decodeScalingModelFit(f?: ScalingModelFit): ScalingModelFitRow | null {
  if (!f) return null;
  return {
    metric: f.metric,
    unit: f.unit,
    model: f.model,
    slopePerSecond: f.slopePerSecond,
    intercept: f.intercept,
    rSquared: f.rSquared,
    sampleCount: f.sampleCount,
    exponent: f.exponent,
    exponentRSquared: f.exponentRSquared,
    reason: f.reason,
  };
}

function decodeScaling(s?: ScalingAnalysis): ScalingAnalysisRow | null {
  if (!s) return null;
  return {
    points: arrayOrEmpty(s.points).map(decodeScalingPoint),
    latencyClassification: s.latencyClassification,
    computeClassification: s.computeClassification,
    confidence: s.confidence,
    reasons: arrayOrEmpty(s.reasons),
    warnings: arrayOrEmpty(s.warnings).map(decodeWarning),
    latencyFit: decodeScalingModelFit(s.latencyFit),
    computeFit: decodeScalingModelFit(s.computeFit),
    metricFits: arrayOrEmpty(s.metricFits).map(decodeScalingModelFit).filter((fit): fit is ScalingModelFitRow => Boolean(fit)),
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
    perClip: arrayOrEmpty(s.perClip).map((c) => ({
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
      editOperations: arrayOrEmpty(c.editOperations).map(decodeEditOperation),
    })),
    werDeltaVsWinner: s.werDeltaVsWinner,
    p95DeltaMsVsWinner: s.p95DeltaMsVsWinner,
    callMultiplierVsWinner: s.callMultiplierVsWinner,
    verdict: s.verdict,
    reasons: arrayOrEmpty(s.reasons),
    warnings: arrayOrEmpty(s.warnings).map(decodeWarning),
    safety: decodeSafety(s.safety),
    stageAttribution: decodeStageAttribution(s.stageAttribution),
    lengthCurves: arrayOrEmpty(s.lengthCurves).map(decodeLengthCurve),
    scaling: decodeScaling(s.scaling),
    commitCount: s.commitCount,
    speakerRejectionCount: s.speakerRejectionCount,
  };
}

export function decodeEvalReport(report?: EvalReport): EvalReportData {
	const promotionReport = report as (EvalReport & {
		promotionVerdicts?: Array<{ engineId?: string; modelId?: string; strategy?: string; policyProfile?: string; stable?: boolean; reasons?: string[] }>;
	}) | undefined;
	return {
    perStrategy: (report?.perStrategy ?? []).map(decodeStrategy),
    qualityMeasured: report?.qualityMeasured ?? false,
    latencyMeasured: report?.latencyMeasured ?? false,
    summary: decodeSummary(report?.summary),
    warnings: (report?.warnings ?? []).map(decodeWarning),
    normalizationPolicy: decodePolicy(report?.normalizationPolicy),
	latencyHonesty: report?.latencyHonesty ?? "",
		promotionVerdicts: (promotionReport?.promotionVerdicts ?? []).map((verdict) => ({
			engineId: verdict.engineId ?? "",
			modelId: verdict.modelId ?? "",
			strategy: verdict.strategy ?? "",
			policyProfile: verdict.policyProfile ?? "",
			stable: Boolean(verdict.stable),
			reasons: verdict.reasons ?? [],
		})),
	};
}

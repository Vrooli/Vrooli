import { createClient } from "@connectrpc/connect";
import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import {
  Capability,
  DiagnosticsService,
  SuiteOverall_Status,
} from "@vrooli/proto-types/audio-tools/v1/diagnostics/diagnostics_pb";
import { SummarizeService, SummarizeLevel } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";
import { transport, uploadFile } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";

const summarizeClient = createClient(SummarizeService, transport);
const diagnosticsClient = createClient(DiagnosticsService, transport);

function providerTierLabel(t: ProviderTier): string {
  switch (t) {
    case ProviderTier.LOCAL: return "local";
    case ProviderTier.BYOK: return "byok";
    case ProviderTier.VROOLI: return "vrooli";
    default: return "";
  }
}

function summarizeLevelFromString(s: "light" | "moderate" | "heavy"): SummarizeLevel {
  switch (s) {
    case "light": return SummarizeLevel.LIGHT;
    case "moderate": return SummarizeLevel.MODERATE;
    case "heavy": return SummarizeLevel.HEAVY;
  }
}

export interface ProviderTrace {
  providerTier: string;
  providerId: string;
  modelId: string;
  latencyMs: number;
}

export interface SummarizeResult {
  text: string;
  trace: ProviderTrace;
  promptTokens: number;
  outputTokens: number;
}

export async function summarize(
  text: string,
  level: "light" | "moderate" | "heavy" = "moderate",
): Promise<Result<SummarizeResult>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await summarizeClient.summarize({ text, level: summarizeLevelFromString(level), model: "", timeoutSeconds: 30 });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return {
      text: resp.text,
      promptTokens: resp.promptTokens,
      outputTokens: resp.outputTokens,
      trace: {
        providerTier: providerTierLabel(resp.providerTier),
        providerId: resp.providerId,
        modelId: resp.modelId,
        latencyMs: resp.latencyMs,
      },
    };
  });
}

export interface TranscribeResult {
  text: string;
  trace: ProviderTrace;
}

/**
 * Transcribe an uploaded audio file. Uses the REST multipart endpoint
 * (`POST /api/v1/voice/transcribe`) — proto JSON would inflate binary payloads.
 * The backend responds with the same shape as the Connect TranscribeResponse.
 */
// -----------------------------------------------------------------------------
// Suite runner — drives the server-side DiagnosticsService.
// -----------------------------------------------------------------------------

export type SuiteCapability = "stt" | "tts" | "summarize" | "transcode";
export type SuiteOverallStatus = "never" | "pass" | "partial" | "fail" | "unknown";

export type QualityStatus = "pass" | "warn" | "fail" | "unknown";

/** One quality-smoke fixture's verdict, decoded from the STT step details. */
export interface QualityFixtureResult {
  fixtureId: string;
  expectedKind: "no_speech" | "speech" | string;
  status: QualityStatus;
  wer: number;
  werThreshold: number;
  filtered: boolean;
  filterReason: string;
  hallucinationDetected: boolean;
  preview: string;
  note: string;
}

/**
 * Layer-2 STT quality-smoke evidence. `assessed` is false when only the
 * readiness probe ran (no fixtures configured). Kept separate from readiness
 * (the step's ok/provider trace) so the UI can show "reachable but degraded".
 */
export interface QualitySmoke {
  assessed: boolean;
  status: QualityStatus;
  hallucinationDetected: boolean;
  fixtures: QualityFixtureResult[];
}

export interface SuiteStep {
  capability: SuiteCapability | "unknown";
  ok: boolean;
  errorCode: string;
  errorMessage: string;
  startedAtUnixMs: number;
  finishedAtUnixMs: number;
  providerTier: string;
  providerId: string;
  modelId: string;
  latencyMs: number;
  details: Record<string, string>;
  /** Present on the STT step; parsed from the opaque quality_* details. */
  quality?: QualitySmoke;
}

function qualityStatusKey(s: string): QualityStatus {
  switch (s) {
    case "pass": return "pass";
    case "warn": return "warn";
    case "fail": return "fail";
    default: return "unknown";
  }
}

/**
 * parseQuality decodes the STT step's opaque quality-smoke details into typed
 * state. Returns undefined when quality was not assessed so callers can fall
 * back to the readiness-only presentation.
 */
function parseQuality(details: Record<string, string>): QualitySmoke | undefined {
  if (details.quality_assessed !== "true") return undefined;
  let fixtures: QualityFixtureResult[] = [];
  const raw = details.quality_fixtures;
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as Array<Record<string, unknown>>;
      fixtures = parsed.map((f) => ({
        fixtureId: String(f.fixture_id ?? ""),
        expectedKind: String(f.expected_kind ?? ""),
        status: qualityStatusKey(String(f.status ?? "")),
        wer: Number(f.wer ?? 0),
        werThreshold: Number(f.wer_threshold ?? 0),
        filtered: Boolean(f.transcript_filtered),
        filterReason: String(f.filter_reason ?? ""),
        hallucinationDetected: Boolean(f.hallucination_detected),
        preview: String(f.preview ?? ""),
        note: String(f.note ?? ""),
      }));
    } catch {
      fixtures = [];
    }
  }
  return {
    assessed: true,
    status: qualityStatusKey(details.quality_status ?? ""),
    hallucinationDetected: details.quality_hallucination_detected === "true",
    fixtures,
  };
}

export interface SuiteRun {
  runId: string;
  startedAtUnixMs: number;
  finishedAtUnixMs: number;
  overall: SuiteOverallStatus;
  passCount: number;
  failCount: number;
  totalCount: number;
  steps: SuiteStep[];
}

function capabilityKey(c: Capability): SuiteStep["capability"] {
  switch (c) {
    case Capability.STT: return "stt";
    case Capability.TTS: return "tts";
    case Capability.SUMMARIZE: return "summarize";
    case Capability.TRANSCODE: return "transcode";
    default: return "unknown";
  }
}

function overallStatusKey(s: SuiteOverall_Status): SuiteOverallStatus {
  switch (s) {
    case SuiteOverall_Status.NEVER: return "never";
    case SuiteOverall_Status.PASS: return "pass";
    case SuiteOverall_Status.PARTIAL: return "partial";
    case SuiteOverall_Status.FAIL: return "fail";
    default: return "unknown";
  }
}

function capabilityToProto(c: SuiteCapability): Capability {
  switch (c) {
    case "stt": return Capability.STT;
    case "tts": return Capability.TTS;
    case "summarize": return Capability.SUMMARIZE;
    case "transcode": return Capability.TRANSCODE;
  }
}

function shapeRun(run: {
  runId: string;
  startedAtUnixMs: bigint;
  finishedAtUnixMs: bigint;
  steps: Array<{
    capability: Capability;
    ok: boolean;
    errorCode: string;
    errorMessage: string;
    startedAtUnixMs: bigint;
    finishedAtUnixMs: bigint;
    providerTier: ProviderTier;
    providerId: string;
    modelId: string;
    latencyMs: number;
    details: Record<string, string>;
  }>;
  overall?: {
    status: SuiteOverall_Status;
    passCount: number;
    failCount: number;
    totalCount: number;
  };
} | undefined): SuiteRun {
  if (!run) {
    return {
      runId: "",
      startedAtUnixMs: 0,
      finishedAtUnixMs: 0,
      overall: "never",
      passCount: 0,
      failCount: 0,
      totalCount: 0,
      steps: [],
    };
  }
  return {
    runId: run.runId,
    startedAtUnixMs: Number(run.startedAtUnixMs),
    finishedAtUnixMs: Number(run.finishedAtUnixMs),
    overall: overallStatusKey(run.overall?.status ?? SuiteOverall_Status.UNSPECIFIED),
    passCount: run.overall?.passCount ?? 0,
    failCount: run.overall?.failCount ?? 0,
    totalCount: run.overall?.totalCount ?? 0,
    steps: run.steps.map((s) => ({
      capability: capabilityKey(s.capability),
      ok: s.ok,
      errorCode: s.errorCode,
      errorMessage: s.errorMessage,
      startedAtUnixMs: Number(s.startedAtUnixMs),
      finishedAtUnixMs: Number(s.finishedAtUnixMs),
      providerTier: providerTierLabel(s.providerTier),
      providerId: s.providerId,
      modelId: s.modelId,
      latencyMs: s.latencyMs,
      details: s.details,
      quality: parseQuality(s.details),
    })),
  };
}

export async function runSuite(capabilities: SuiteCapability[] = []): Promise<Result<SuiteRun>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await diagnosticsClient.runSuite({ capabilities: capabilities.map(capabilityToProto) });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return shapeRun(resp.run);
  });
}

export async function getLastSuiteRun(): Promise<Result<SuiteRun>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await diagnosticsClient.getLastRun({});
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return shapeRun(resp.run);
  });
}

export async function transcribe(file: File): Promise<Result<TranscribeResult>> {
  return tryCall(async () => {
    const fd = new FormData();
    fd.append("audio", file);
    const res = await uploadFile("/api/v1/voice/transcribe", fd);
    if (!res.ok) {
      const { decodeApiError } = await import("../api/client");
      throw await decodeApiError(res);
    }
    const json = (await res.json()) as {
      text?: string;
      provider_tier?: string;
      provider_id?: string;
      model_id?: string;
      latency_ms?: number;
    };
    return {
      text: json.text ?? "",
      trace: {
        providerTier: json.provider_tier ?? "",
        providerId: json.provider_id ?? "",
        modelId: json.model_id ?? "",
        latencyMs: json.latency_ms ?? 0,
      },
    };
  });
}

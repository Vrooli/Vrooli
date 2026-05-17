import { createClient } from "@connectrpc/connect";
import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { SummarizeService } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";
import { SummarizeLevel } from "@vrooli/proto-types/audio-tools/v1/tts/tts_pb";
import { transport, uploadFile } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";

const summarizeClient = createClient(SummarizeService, transport);

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

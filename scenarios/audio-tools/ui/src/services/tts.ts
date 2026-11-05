import { createClient } from "@connectrpc/connect";
import { ProviderTier, ResponseFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { ProviderState } from "@vrooli/proto-types/audio-tools/v1/shared/shared_pb";
import { TTSService } from "@vrooli/proto-types/audio-tools/v1/tts/tts_pb";
import { transport } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";

const client = createClient(TTSService, transport);

function providerTierLabel(t: ProviderTier): string {
  switch (t) {
    case ProviderTier.LOCAL: return "local";
    case ProviderTier.BYOK: return "byok";
    case ProviderTier.VROOLI: return "vrooli";
    default: return "";
  }
}

function responseFormatFromString(s: string): ResponseFormat {
  switch (s) {
    case "mp3": return ResponseFormat.MP3;
    case "wav": return ResponseFormat.WAV;
    case "opus": return ResponseFormat.OPUS;
    case "flac": return ResponseFormat.FLAC;
    default: return ResponseFormat.UNSPECIFIED;
  }
}

export interface SynthesizeResult {
  audio: Uint8Array;
  contentType: string;
  providerTier: string;
  providerId: string;
  modelId: string;
  latencyMs: number;
}

export async function synthesize(text: string, voice: string, speed = 1.0, format = "wav"): Promise<Result<SynthesizeResult>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.synthesize({ text, voice, speed, responseFormat: responseFormatFromString(format) });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return {
      audio: resp.audio,
      contentType: resp.contentType,
      providerTier: providerTierLabel(resp.providerTier),
      providerId: resp.providerId,
      modelId: resp.modelId,
      latencyMs: resp.latencyMs,
    };
  });
}

export async function listVoices(): Promise<Result<{ id: string; name: string }[]>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.listVoices({});
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return resp.voices.map((v) => ({ id: v.id, name: v.name }));
  });
}

export interface TtsStatus {
  capability: string;
  capabilityLabel: string;
  availability: { tier: string; providerId: string; available: boolean }[];
}

export async function getStatus(): Promise<Result<TtsStatus>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.getStatus({});
    } catch (e) {
      throw normalizeConnectError(e);
    }
    const s = resp.status;
    if (!s) {
      throw new Error("getStatus returned no status");
    }
    return {
      capability: s.capability,
      capabilityLabel: s.capabilityLabel,
      availability: s.availability.map((a) => ({ tier: providerTierLabel(a.tier), providerId: a.providerId, available: a.state === ProviderState.AVAILABLE })),
    };
  });
}

import { createClient } from "@connectrpc/connect";
import { TTSService } from "@vrooli/proto-types/audio-tools/v1/tts/tts_pb";
import { transport } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";

const client = createClient(TTSService, transport);

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
      resp = await client.synthesize({ text, voice, speed, responseFormat: format });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return {
      audio: resp.audio,
      contentType: resp.contentType,
      providerTier: resp.providerTier,
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
      availability: s.availability.map((a) => ({ tier: a.tier, providerId: a.providerId, available: a.available })),
    };
  });
}

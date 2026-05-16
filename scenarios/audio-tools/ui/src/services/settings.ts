import { createClient } from "@connectrpc/connect";
import { SettingsService } from "@vrooli/proto-types/audio-tools/v1/settings/settings_pb";
import { transport, ApiError, makeApiError } from "../api/client";
import { tryCall, type Result } from "./result";

const client = createClient(SettingsService, transport);

export interface ProviderTierFlags {
  byokEnabled: boolean;
  vrooliEnabled: boolean;
  localEnabled: boolean;
  whisperUrl?: string;
  kokoroUrl?: string;
  ollamaUrl?: string;
}

export async function getProviderConfig(): Promise<Result<ProviderTierFlags>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.getProviderConfig({});
    } catch (e) {
      throw normalizeConnectError(e);
    }
    const c = resp.config;
    return {
      byokEnabled: Boolean(c?.byokEnabled),
      vrooliEnabled: Boolean(c?.vrooliEnabled),
      localEnabled: Boolean(c?.localEnabled),
      whisperUrl: c?.whisperUrl || undefined,
      kokoroUrl: c?.kokoroUrl || undefined,
      ollamaUrl: (c as { ollamaUrl?: string } | undefined)?.ollamaUrl || undefined,
    };
  });
}

export interface ByokCredentialRow {
  providerId: string;
  capability: string;
  fingerprint: string;
  createdAt: string;
}

export async function listByokCredentials(): Promise<Result<ByokCredentialRow[]>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.listBYOKCredentials({});
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return resp.credentials.map((c) => ({
      providerId: c.providerId,
      capability: c.capability,
      fingerprint: c.fingerprint,
      createdAt: c.createdAt,
    }));
  });
}

export interface VoiceOverrideRow {
  canonicalVoice: string;
  tierProvider: string;
  adapterVoice: string;
}

export async function getVoiceOverrides(): Promise<Result<VoiceOverrideRow[]>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.getVoiceOverrides({});
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return resp.overrides.map((o) => ({
      canonicalVoice: o.canonicalVoice,
      tierProvider: o.tierProvider,
      adapterVoice: o.adapterVoice,
    }));
  });
}

function normalizeConnectError(e: unknown): ApiError {
  if (e instanceof ApiError) return e;
  const message = e instanceof Error ? e.message : String(e);
  const code = (e as { code?: string | number }).code ?? "internal";
  const codeStr = typeof code === "string" ? code : String(code);
  const status = codeStr === "unimplemented" ? 501 : 500;
  return makeApiError(codeStr, message, status);
}

export { normalizeConnectError };

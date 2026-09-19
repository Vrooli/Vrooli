import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema, type Timestamp } from "@bufbuild/protobuf/wkt";
import { SettingsService } from "@vrooli/proto-types/audio-tools/v1/settings/settings_pb";
import { transport, ApiError, makeApiError } from "../api/client";
import { tryCall, type Result } from "./result";

function timestampToISO(ts: Timestamp | string | undefined): string {
  if (!ts) return "";
  if (typeof ts === "string") return ts;
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : ts.seconds;
  const nanos = ts.nanos;
  return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
}

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
      createdAt: timestampToISO(c.createdAt),
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

export interface UpdateProviderConfigInput {
  byokEnabled?: boolean;
  vrooliEnabled?: boolean;
  localEnabled?: boolean;
  whisperUrl?: string;
  kokoroUrl?: string;
  ollamaUrl?: string;
  lpbsBaseUrl?: string;
}

export async function updateProviderConfig(input: UpdateProviderConfigInput): Promise<Result<ProviderTierFlags>> {
  return tryCall(async () => {
    const paths: string[] = [];
    const cfg: Record<string, unknown> = {};
    if (input.byokEnabled !== undefined) { cfg.byokEnabled = input.byokEnabled; paths.push("byok_enabled"); }
    if (input.vrooliEnabled !== undefined) { cfg.vrooliEnabled = input.vrooliEnabled; paths.push("vrooli_enabled"); }
    if (input.localEnabled !== undefined) { cfg.localEnabled = input.localEnabled; paths.push("local_enabled"); }
    if (input.whisperUrl !== undefined) { cfg.whisperUrl = input.whisperUrl; paths.push("whisper_url"); }
    if (input.kokoroUrl !== undefined) { cfg.kokoroUrl = input.kokoroUrl; paths.push("kokoro_url"); }
    if (input.ollamaUrl !== undefined) { cfg.ollamaUrl = input.ollamaUrl; paths.push("ollama_url"); }
    if (input.lpbsBaseUrl !== undefined) { cfg.lpbsBaseUrl = input.lpbsBaseUrl; paths.push("lpbs_base_url"); }
    let resp;
    try {
      resp = await client.updateProviderConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
      });
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

export async function upsertByokCredential(providerId: string, capability: string, apiKey: string): Promise<Result<ByokCredentialRow>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.upsertBYOKCredential({ providerId, capability, secret: { case: "apiKey", value: apiKey } } as never);
    } catch (e) {
      throw normalizeConnectError(e);
    }
    const c = resp.credential;
    if (!c) {
      throw new Error("upsertBYOKCredential returned no credential");
    }
    return { providerId: c.providerId, capability: c.capability, fingerprint: c.fingerprint, createdAt: timestampToISO(c.createdAt) };
  });
}

export async function deleteByokCredential(providerId: string, capability: string): Promise<Result<void>> {
  return tryCall(async () => {
    try {
      await client.deleteBYOKCredential({ providerId, capability });
    } catch (e) {
      throw normalizeConnectError(e);
    }
  });
}

export async function setVoiceOverride(canonicalVoice: string, tierProvider: string, adapterVoice: string): Promise<Result<VoiceOverrideRow[]>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.setVoiceOverride({ override: { canonicalVoice, tierProvider, adapterVoice } });
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

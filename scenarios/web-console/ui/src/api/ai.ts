// Connect-RPC client + wrappers for the AI domain.

import { createClient } from "@connectrpc/connect";
import { AIService } from "@vrooli/proto-types/web-console/v1/ai/ai_pb";

import { transport } from "./client";
import { extractAPIError } from "../lib/errors";
import { getAccessToken } from "../lib/auth";

export const aiClient = createClient(AIService, transport);

// [REQ:P0-005a] AI Command Generation
export interface AIGenerateResponse {
  command: string;
  provider: string;
}

export interface AIPrivacyConsentGrant {
  id: string;
  source: "own-key" | "vrooli-credits";
  created_at: string;
  expires_at: string;
  revoked_at?: string;
}

async function consentRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  const accessToken = await getAccessToken();
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (init?.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(path, { ...init, headers });
  if (!response.ok) throw await extractAPIError(response, "AI data-sharing consent request failed");
  if (response.status === 204) return undefined as T;
  return await response.json() as T;
}

export const listAIPrivacyConsent = () => consentRequest<{ grants: AIPrivacyConsentGrant[] }>("/api/v1/ai/privacy-consent");
export const createAIPrivacyConsent = (source: AIPrivacyConsentGrant["source"]) =>
  consentRequest<{ grant: AIPrivacyConsentGrant; receipt: string }>("/api/v1/ai/privacy-consent", {
    method: "POST", body: JSON.stringify({ source, lifetime_days: 30 }),
  });
export const revokeAIPrivacyConsent = (grantID: string) =>
  consentRequest<void>(`/api/v1/ai/privacy-consent/${encodeURIComponent(grantID)}`, { method: "DELETE" });

export const AI_PRIVACY_CONSENT_SESSION_KEY = "web-console.ai.remote-consent-receipt";
export const AI_PRIVACY_CONSENT_SOURCE_SESSION_KEY = "web-console.ai.remote-consent-source";

function newAIRequestID(): string {
  return crypto.randomUUID();
}

export async function generateAICommand(prompt: string, context?: string): Promise<AIGenerateResponse> {
  const headers = new Headers({ "Content-Type": "application/json", "X-Idempotency-Key": newAIRequestID() });
  const accessToken = await getAccessToken();
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  const receipt = window.sessionStorage.getItem(AI_PRIVACY_CONSENT_SESSION_KEY);
  const receiptSource = window.sessionStorage.getItem(AI_PRIVACY_CONSENT_SOURCE_SESSION_KEY);
  if (receipt && receiptSource) {
    headers.set("X-Vrooli-AI-Consent-Grant", receipt);
    headers.set("X-Vrooli-AI-Consent-Source", receiptSource);
  }
  const response = await fetch("/api/v1/ai/generate", {
    method: "POST",
    headers,
    body: JSON.stringify({ prompt, context: context ?? "" }),
  });
  if (!response.ok) throw await extractAPIError(response, "AI generation failed");
  const resp = await response.json() as AIGenerateResponse;
  return { command: resp.command, provider: resp.provider };
}

// [REQ:P0-005a] AI Suggestions
export interface AISuggestResponse {
  commands: string[];
  provider: string;
}

export async function generateAISuggestions(prompt: string, context?: string): Promise<AISuggestResponse> {
  const resp = await aiClient.suggest({ prompt, context: context ?? "" }, { headers: { "X-Idempotency-Key": newAIRequestID() } });
  return { commands: resp.commands, provider: resp.provider };
}

// [REQ:P1-003a] AI Provider Config
export interface ProviderConfig {
  name: string;
  enabled: boolean;
  priority: number;
  timeout_sec: number;
  max_retries: number;
  key_configured?: boolean;
  key_source?: string;
}

export interface ProviderHealth {
  name: string;
  available: boolean;
  last_check?: string;
  last_latency?: string;
  error_count: number;
  success_count: number;
  error_rate: number;
}

export interface AIProviderConfigResponse {
  providers: ProviderConfig[];
  health: ProviderHealth[];
}

type AIConfigClientReq = Parameters<typeof aiClient.getConfig>[0];
type AIConfigClientResp = Awaited<ReturnType<typeof aiClient.getConfig>>;
type AIProviderConfigProto = AIConfigClientResp["providers"][number];
type AIProviderHealthProto = AIConfigClientResp["health"][number];

function decodeProviderConfig(p: AIProviderConfigProto): ProviderConfig {
  return {
    name: p.name,
    enabled: p.enabled,
    priority: p.priority,
    timeout_sec: p.timeoutSec,
    max_retries: p.maxRetries,
    key_configured: p.keyConfigured,
    key_source: p.keySource,
  };
}

function decodeProviderHealth(h: AIProviderHealthProto): ProviderHealth {
  return {
    name: h.name,
    available: h.available,
    last_check: h.lastCheck || undefined,
    last_latency: h.lastLatency || undefined,
    error_count: Number(h.errorCount),
    success_count: Number(h.successCount),
    error_rate: h.errorRate,
  };
}

export async function getAIConfig(): Promise<AIProviderConfigResponse> {
  const req: AIConfigClientReq = {};
  const resp = await aiClient.getConfig(req);
  return {
    providers: resp.providers.map(decodeProviderConfig),
    health: resp.health.map(decodeProviderHealth),
  };
}

export async function updateAIConfig(update: {
  name: string;
  enabled?: boolean;
  priority?: number;
  timeout_sec?: number;
  max_retries?: number;
}): Promise<AIProviderConfigResponse> {
  const req: Parameters<typeof aiClient.updateConfig>[0] = { name: update.name };
  if (update.enabled !== undefined) {
    req.enabled = update.enabled;
    req.hasEnabled = true;
  }
  if (update.priority !== undefined) {
    req.priority = update.priority;
    req.hasPriority = true;
  }
  if (update.timeout_sec !== undefined) {
    req.timeoutSec = update.timeout_sec;
    req.hasTimeoutSec = true;
  }
  if (update.max_retries !== undefined) {
    req.maxRetries = update.max_retries;
    req.hasMaxRetries = true;
  }
  const resp = await aiClient.updateConfig(req);
  return {
    providers: resp.providers.map(decodeProviderConfig),
    health: resp.health.map(decodeProviderHealth),
  };
}

export async function getAIHealth(): Promise<ProviderHealth[]> {
  const resp = await aiClient.getHealth({});
  return resp.health.map(decodeProviderHealth);
}

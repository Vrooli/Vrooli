// DOC: docs/internal/SEAMS.md#ai-connect-client-seam
// Connect-RPC client + wrappers for the AI domain.

import { createClient } from "@connectrpc/connect";
import { AIService } from "@vrooli/proto-types/web-console/v1/ai/ai_pb";

import { transport } from "./client";

export const aiClient = createClient(AIService, transport);

// [REQ:P0-005a] AI Command Generation
export interface AIGenerateResponse {
  command: string;
  provider: string;
}

export async function generateAICommand(prompt: string, context?: string): Promise<AIGenerateResponse> {
  const resp = await aiClient.generate({ prompt, context: context ?? "" });
  return { command: resp.command, provider: resp.provider };
}

// [REQ:P0-005a] AI Suggestions
export interface AISuggestResponse {
  commands: string[];
  provider: string;
}

export async function generateAISuggestions(prompt: string, context?: string): Promise<AISuggestResponse> {
  const resp = await aiClient.suggest({ prompt, context: context ?? "" });
  return { commands: resp.commands, provider: resp.provider };
}

// [REQ:P1-003a] AI Provider Config
export interface ProviderConfig {
  name: string;
  enabled: boolean;
  priority: number;
  timeout_sec: number;
  max_retries: number;
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

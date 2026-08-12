// DOC: docs/internal/SEAMS.md#ai-connect-client-seam
// Connect-RPC client + wrappers for the AI domain.
import { createClient } from "@connectrpc/connect";
import { AIService } from "@vrooli/proto-types/web-console/v1/ai/ai_pb";
import { transport } from "./client";
export const aiClient = createClient(AIService, transport);
export async function generateAICommand(prompt, context) {
    const resp = await aiClient.generate({ prompt, context: context ?? "" });
    return { command: resp.command, provider: resp.provider };
}
export async function generateAISuggestions(prompt, context) {
    const resp = await aiClient.suggest({ prompt, context: context ?? "" });
    return { commands: resp.commands, provider: resp.provider };
}
function decodeProviderConfig(p) {
    return {
        name: p.name,
        enabled: p.enabled,
        priority: p.priority,
        timeout_sec: p.timeoutSec,
        max_retries: p.maxRetries,
    };
}
function decodeProviderHealth(h) {
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
export async function getAIConfig() {
    const req = {};
    const resp = await aiClient.getConfig(req);
    return {
        providers: resp.providers.map(decodeProviderConfig),
        health: resp.health.map(decodeProviderHealth),
    };
}
export async function updateAIConfig(update) {
    const req = { name: update.name };
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
export async function getAIHealth() {
    const resp = await aiClient.getHealth({});
    return resp.health.map(decodeProviderHealth);
}

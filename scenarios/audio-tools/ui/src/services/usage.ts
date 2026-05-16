import { createClient } from "@connectrpc/connect";
import { UsageService } from "@vrooli/proto-types/audio-tools/v1/usage/usage_pb";
import { transport } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";

const client = createClient(UsageService, transport);

export interface UsageRow {
  operationId: string;
  emittedAt: string;
  capability: string;
  operation: string;
  providerTier: string;
  providerId: string;
  modelId: string;
  latencyMs: number;
  creditsCharged: number;
  error: string;
  fallbackReason: string;
}

export async function listRecent(sinceSeconds = 60 * 60 * 24, limit = 50): Promise<Result<UsageRow[]>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.listRecent({
        sinceSeconds: BigInt(sinceSeconds),
        afterEmittedAt: "",
        limit,
      });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return resp.rows.map((r) => ({
      operationId: r.operationId,
      emittedAt: r.emittedAt,
      capability: r.capability,
      operation: r.operation,
      providerTier: r.providerTier,
      providerId: r.providerId,
      modelId: r.modelId,
      latencyMs: r.latencyMs,
      creditsCharged: r.creditsCharged,
      error: r.error,
      fallbackReason: r.fallbackReason,
    }));
  });
}

export interface ProviderDistribution {
  providerTier: string;
  providerId: string;
  count: number;
}

export interface FallbackReason {
  reason: string;
  count: number;
}

export interface UsageSummary {
  since: string;
  until: string;
  operationsTotal: number;
  creditsTotal: number;
  errorCount: number;
  distribution: ProviderDistribution[];
  fallbackReasons: FallbackReason[];
}

export async function getSummary(sinceSeconds = 60 * 60 * 24, capability = ""): Promise<Result<UsageSummary>> {
  return tryCall(async () => {
    let resp;
    try {
      resp = await client.getSummary({
        sinceSeconds: BigInt(sinceSeconds),
        capability,
      });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    const s = resp.summary;
    return {
      since: s?.since ?? "",
      until: s?.until ?? "",
      operationsTotal: Number(s?.operationsTotal ?? 0n),
      creditsTotal: Number(s?.creditsTotal ?? 0n),
      errorCount: Number(s?.errorCount ?? 0n),
      distribution: (s?.distribution ?? []).map((d) => ({
        providerTier: d.providerTier,
        providerId: d.providerId,
        count: Number(d.count),
      })),
      fallbackReasons: (s?.fallbackReasons ?? []).map((f) => ({
        reason: f.reason,
        count: Number(f.count),
      })),
    };
  });
}

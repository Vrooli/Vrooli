import { createClient } from "@connectrpc/connect";
import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { UsageService } from "@vrooli/proto-types/audio-tools/v1/usage/usage_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { transport } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";

const client = createClient(UsageService, transport);

function providerTierLabel(t: ProviderTier): string {
  switch (t) {
    case ProviderTier.LOCAL:
      return "local";
    case ProviderTier.BYOK:
      return "byok";
    case ProviderTier.VROOLI:
      return "vrooli";
    default:
      return "";
  }
}

function timestampToISO(ts: Timestamp | undefined): string {
  if (!ts) return "";
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : Number(ts.seconds ?? 0);
  const nanos = Number(ts.nanos ?? 0);
  return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
}

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
        limit,
      });
    } catch (e) {
      throw normalizeConnectError(e);
    }
    return resp.rows.map((r) => ({
      operationId: r.operationId,
      emittedAt: timestampToISO(r.emittedAt),
      capability: r.capability,
      operation: r.operation,
      providerTier: providerTierLabel(r.providerTier),
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
      since: timestampToISO(s?.since),
      until: timestampToISO(s?.until),
      operationsTotal: Number(s?.operationsTotal ?? 0n),
      creditsTotal: Number(s?.creditsTotal ?? 0n),
      errorCount: Number(s?.errorCount ?? 0n),
      distribution: (s?.distribution ?? []).map((d) => ({
        providerTier: providerTierLabel(d.providerTier),
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

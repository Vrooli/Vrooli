import { createClient } from "@connectrpc/connect";
import { ContinuityService } from "@vrooli/proto-types/web-console/v1/continuity/continuity_pb";

import { transport } from "./client";

export const continuityClient = createClient(ContinuityService, transport);

/** Catalog-link drift counts; answered from indexes without reading bodies. */
export interface ContinuityHealth {
  orphanConversations: number;
  orphanCheckpoints: number;
  orphanWorkspacePanes: number;
  uncatalogedConversations: number;
}

export interface ContinuitySearchMatch {
  eventId: string;
  sessionId: string;
  sequence: number;
  role: string;
  createdAt: string;
  occurredAt?: string;
  excerpt: string;
  lifecycleState: string;
}

export interface ContinuityReceipt {
  operationId: string;
  sessionId: string;
  command: string;
  fromState: string;
  toState: string;
  status: string;
  errorCode: string;
  createdAt: string;
  completedAt?: string;
}

/**
 * Fast drift observation for warnings. The full integrity audit (CLI
 * `continuity integrity`) also hashes every message body; the UI never needs
 * that conservation receipt.
 */
export async function getContinuityHealth(): Promise<ContinuityHealth> {
  const report = await continuityClient.health({});
  return {
    orphanConversations: Number(report.orphanConversations),
    orphanCheckpoints: Number(report.orphanCheckpoints),
    orphanWorkspacePanes: Number(report.orphanWorkspacePanes),
    uncatalogedConversations: Number(report.uncatalogedConversations),
  };
}

export async function getContinuityReceipt(operationId: string): Promise<ContinuityReceipt> {
  const receipt = await continuityClient.getReceipt({ operationId });
  return {
    operationId: receipt.operationId,
    sessionId: receipt.sessionId,
    command: receipt.command,
    fromState: receipt.fromState,
    toState: receipt.toState,
    status: receipt.status,
    errorCode: receipt.errorCode,
    createdAt: receipt.createdAt,
    completedAt: receipt.completedAt || undefined,
  };
}

export async function searchLocalContinuity(
  query: string,
  opts?: { after?: string; agentType?: string; cwd?: string; state?: string; limit?: number; sessionId?: string; agentSessionId?: string; title?: string; topicSummary?: string },
): Promise<{ matches: ContinuitySearchMatch[]; totalMatches: number; distinctSessions: number; truncated: boolean }> {
  // Keep this additive while older installed proto-types copies are refreshed;
  // current generated descriptors encode the catalog filters below.
  const request = {
    query,
    createdAfter: opts?.after ?? "",
    agentType: opts?.agentType ?? "",
    cwd: opts?.cwd ?? "",
    lifecycleState: opts?.state ?? "any",
    limit: opts?.limit ?? 100,
    sessionId: opts?.sessionId ?? "",
    agentSessionId: opts?.agentSessionId ?? "",
    title: opts?.title ?? "",
    topicSummary: opts?.topicSummary ?? "",
  } as Parameters<typeof continuityClient.search>[0];
  const response = await continuityClient.search(request);
  return {
    matches: response.matches.map((match) => ({
      eventId: match.eventId,
      sessionId: match.sessionId,
      sequence: Number(match.sequence),
      role: match.role,
      createdAt: match.createdAt,
      ...((match as typeof match & { occurredAt?: string }).occurredAt
        ? { occurredAt: (match as typeof match & { occurredAt?: string }).occurredAt }
        : {}),
      excerpt: match.excerpt,
      lifecycleState: match.lifecycleState,
    })),
    totalMatches: Number(response.totalMatches),
    distinctSessions: Number(response.distinctSessions),
    truncated: response.truncated,
  };
}

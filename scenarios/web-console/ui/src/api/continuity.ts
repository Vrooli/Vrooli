import { createClient } from "@connectrpc/connect";
import { ContinuityService } from "@vrooli/proto-types/web-console/v1/continuity/continuity_pb";

import { transport } from "./client";

export const continuityClient = createClient(ContinuityService, transport);

export interface ContinuityIntegrity {
  sessions: number;
  conversationSessions: number;
  conversationEvents: number;
  checkpoints: number;
  workspacePanes: number;
  orphanConversations: number;
  orphanCheckpoints: number;
  orphanWorkspacePanes: number;
  uncatalogedConversations: number;
  generation: string;
  eventContentHash: string;
}

export interface ContinuitySearchMatch {
  eventId: string;
  sessionId: string;
  sequence: number;
  role: string;
  createdAt: string;
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

export async function getContinuityIntegrity(): Promise<ContinuityIntegrity> {
  const report = await continuityClient.integrity({});
  return {
    sessions: Number(report.sessions),
    conversationSessions: Number(report.conversationSessions),
    conversationEvents: Number(report.conversationEvents),
    checkpoints: Number(report.checkpoints),
    workspacePanes: Number(report.workspacePanes),
    orphanConversations: Number(report.orphanConversations),
    orphanCheckpoints: Number(report.orphanCheckpoints),
    orphanWorkspacePanes: Number(report.orphanWorkspacePanes),
    uncatalogedConversations: Number(report.uncatalogedConversations),
    generation: report.generation,
    eventContentHash: report.eventContentHash,
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
      excerpt: match.excerpt,
      lifecycleState: match.lifecycleState,
    })),
    totalMatches: Number(response.totalMatches),
    distinctSessions: Number(response.distinctSessions),
    truncated: response.truncated,
  };
}

import { createClient } from "@connectrpc/connect";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { CodeFactsService, type SearchInvariantCandidatesResponse } from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";

import { transport } from "./client";

const factsClient = createClient(CodeFactsService, transport);
const restApiBase = resolveApiBase({ appendSuffix: true });

export interface CandidateReceiptMatchCandidate {
  candidateId: string;
  scenario: string;
  relativePath: string;
  sourceLine: number;
  sourceFileSha256: string;
}

export interface CandidateReceiptMatchResponse {
  status: "EXACT_MATCH" | "UNKNOWN";
  receiptId: string;
  candidate: CandidateReceiptMatchCandidate;
  reason: string;
}

// The owner API is a source-candidate search, not an enforcement verifier.
// Keep the one-scenario/path scope and its ten-result ceiling explicit.
export async function searchInvariantCandidates(
  query: string,
  scenario: string,
  path: string,
  limit = 10,
): Promise<SearchInvariantCandidatesResponse> {
  const response = await factsClient.searchInvariantCandidates({
    query,
    scenario,
    path,
    limit: Math.min(Math.max(limit, 1), 10),
  });
  return response;
}

// The Code Facts owner proxies one receipt lookup to Test Genie. Resolver or
// transport errors remain errors; they are never converted to UNKNOWN.
export async function matchInvariantCandidateReceipt(
  receiptId: string,
  candidate: CandidateReceiptMatchCandidate,
): Promise<CandidateReceiptMatchResponse> {
  const response = await fetch(buildApiUrl("/invariants/candidate-receipt-match", { baseUrl: restApiBase }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    body: JSON.stringify({ receiptId, candidate }),
  });
  if (!response.ok) throw new Error(`Test Genie receipt check unavailable (${response.status})`);
  return await response.json() as CandidateReceiptMatchResponse;
}

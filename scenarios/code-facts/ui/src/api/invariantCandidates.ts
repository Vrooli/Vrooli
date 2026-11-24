import { createClient } from "@connectrpc/connect";
import { CodeFactsService, type SearchInvariantCandidatesResponse } from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";

import { transport } from "./client";

const factsClient = createClient(CodeFactsService, transport);

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

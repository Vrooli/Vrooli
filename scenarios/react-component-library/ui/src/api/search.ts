import { createClient } from "@connectrpc/connect";
import {
  SearchAvailability,
  SearchKind,
  SearchService,
  SemanticLegState,
  type SearchFacet,
  type SearchResponse,
  type SearchResult,
  type SearchStatus,
} from "@vrooli/proto-types/react-component-library/v1/search/search_pb";

import { slowTransport, transport } from "./client";

const client = createClient(SearchService, transport);
const reindexClient = createClient(SearchService, slowTransport);

export { SearchAvailability, SearchKind, SemanticLegState };
export type { SearchFacet, SearchResponse, SearchResult, SearchStatus };

export async function searchWorkbench(
  input: {
    query: string;
    kinds?: SearchKind[];
    limit?: number;
    ask?: boolean;
  },
  options?: { signal?: AbortSignal },
): Promise<SearchResponse> {
  return client.search(
    {
      query: input.query,
      kinds: input.kinds ?? [],
      limit: input.limit ?? 30,
      ask: input.ask ?? false,
    },
    options,
  );
}

export async function suggestWorkbench(query: string, limit = 8): Promise<string[]> {
  const response = await client.suggest({ query, limit });
  return response.suggestions;
}

export async function getSearchStatus(): Promise<SearchStatus> {
  return client.getSearchStatus({});
}

export async function reindexSearch(force = false): Promise<number> {
  const response = await reindexClient.reindex({ force });
  return response.documentCount;
}

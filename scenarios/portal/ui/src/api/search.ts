import { createClient } from "@connectrpc/connect";
import { SearchService, type SuggestResponse } from "@vrooli/proto-types/portal/v1/search/search_pb";

import { transport } from "./client";

const searchClient = createClient(SearchService, transport);

export interface SuggestInput {
  query: string;
  types?: string[];
  limit?: number;
  group?: string;
}

export async function suggest(input: SuggestInput): Promise<SuggestResponse> {
  return searchClient.suggest({
    query: input.query,
    types: input.types ?? [],
    limit: input.limit ?? 5,
    group: input.group ?? "",
  });
}

export type { SuggestResponse };

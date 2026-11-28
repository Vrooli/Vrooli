import { useMutation, useQuery, type UseMutationResult, type UseQueryResult } from "@tanstack/react-query";

import { goCodeGraphClient } from "../../../api/graph";
import type {
  ListFixturesResponse,
  ValidateFixtureResponse,
} from "@vrooli/proto-types/go-code-graph/v1/graph/graph_pb";

export const fixtureKeys = {
  all: () => ["fixtures"] as const,
  list: () => [...fixtureKeys.all(), "list"] as const,
};

/**
 * List the golden determinism fixtures the server ships. The browser can't
 * read bas/fixtures/* directly, so this RPC is the only path to the list.
 */
export function useListFixtures(): UseQueryResult<ListFixturesResponse> {
  return useQuery({
    queryKey: fixtureKeys.list(),
    queryFn: () => goCodeGraphClient.listFixtures({}),
  });
}

/**
 * Validate a named fixture server-side (re-extract + byte-compare against
 * expected-graph.json). A mutation, since it's an explicit user action that
 * shouldn't auto-run or cache stale pass/fail state.
 */
export function useValidateFixture(): UseMutationResult<ValidateFixtureResponse, Error, string> {
  return useMutation({
    mutationFn: (name: string) => goCodeGraphClient.validateFixture({ name }),
  });
}

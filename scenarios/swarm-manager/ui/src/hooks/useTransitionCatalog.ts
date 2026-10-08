/**
 * useTransitionCatalog — read-only access to the server's transition registry.
 *
 * The catalog is small, immutable for the life of a deploy, and already
 * memoised inside `transitionService`. Several components were each issuing
 * their own `["transition-catalog"]` query with slightly different staleTime
 * values; this hook is the one definition.
 *
 * Its main job is answering "what kind of thing is this transition?" so a
 * button can show what it is about to do before it is pressed.
 */

import { useQuery } from "@tanstack/react-query";
import type { TransitionKind } from "@vrooli/proto-types/swarm-manager/v1/domain/transition_pb";
// Imported through the services barrel, not `../services/transition-service`,
// because the barrel is the seam consumers substitute in tests. A direct
// import bypasses that substitution and drags the real Connect transport —
// which resolves the API base at module load — into any suite that renders a
// component using this hook.
import { transitionService } from "../services";

export const TRANSITION_CATALOG_QUERY_KEY = ["transition-catalog"] as const;

export function useTransitionCatalog(enabled = true) {
  return useQuery({
    queryKey: TRANSITION_CATALOG_QUERY_KEY,
    queryFn: () => transitionService.list(),
    // Registry contents change only on deploy.
    staleTime: 5 * 60_000,
    enabled,
  });
}

/**
 * Resolves one transition's declared kind.
 *
 * Returns undefined while loading or when the key is not declared — callers
 * must treat that as "unknown", never as "harmless". `consequenceOf` already
 * does, falling back to a side-effecting classification.
 */
export function useTransitionKind(transitionKey: string | undefined): TransitionKind | undefined {
  const { data } = useTransitionCatalog(Boolean(transitionKey));
  if (!transitionKey) return undefined;
  return data?.find((candidate) => candidate.key === transitionKey)?.kind;
}

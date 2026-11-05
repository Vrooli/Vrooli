/**
 * usePendingDecisionCount — the single number for "how much needs my decision".
 *
 * Every surface that offers to open the decision drawer reads this hook, and it
 * counts exactly what the drawer paginates over (the ranked next-action feed).
 * Before this existed the sidebar badge, the plan board's decide button, and the
 * drawer's own "N of M" each derived their own number from a different source,
 * so the same queue was advertised as three different sizes.
 */

import { useQuery } from "@tanstack/react-query";
import { nextActionService } from "../services/next-action-service";

/** Query key shared with the decision drawer so both read one cache entry. */
export const NEXT_ACTION_FEED_QUERY_KEY = ["next-actions-feed"] as const;

export function usePendingDecisionCount(): number {
  const { data } = useQuery({
    queryKey: NEXT_ACTION_FEED_QUERY_KEY,
    queryFn: () => nextActionService.getFeed(),
    staleTime: 15_000,
  });

  return data?.entries.length ?? 0;
}

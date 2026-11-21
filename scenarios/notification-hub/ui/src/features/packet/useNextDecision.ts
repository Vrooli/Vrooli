import { useQuery } from "@tanstack/react-query";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient } from "../../api/notifications";

/**
 * The next decision waiting for the operator, so a run of decisions can be
 * cleared without going back to the list each time. A decision that is only
 * waiting for its requester's reply to a question comes after those that need
 * the operator now.
 */
export function useNextDecision(currentId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["asks", "next", currentId],
    enabled,
    staleTime: 0,
    retry: false,
    queryFn: async (): Promise<Ask | null> => {
      const { asks } = await conversationsClient.listAsks({ limit: 25, stateFilter: "awaiting" });
      const others = asks.filter((ask) => ask.id !== currentId);
      return others.find((ask) => ask.openContextRequestCount === 0) ?? others[0] ?? null;
    },
  });
}

import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";

import { conversationsClient } from "../../api/notifications";
import { useSession } from "../session/useSession";

const REFRESH_MS = 60_000;

export interface NeedsYou {
  /** Waiting decisions plus pending permission requests. */
  count: number;
  /** False until at least one of the two reads has answered. */
  known: boolean;
}

/**
 * How many decisions are waiting on the owner, for the Needs-you tab and the
 * installed app's badge. Decisions come from the Hub's exact `total_awaiting`,
 * which every page carries, so a one-row read is enough however long the queue
 * is. It counts decisions, not status updates: a status waits for a read, not
 * an answer. Native permission requests are not in that total and are added
 * here.
 */
export function useNeedsYou(): NeedsYou {
  const session = useSession();
  const owner = session.isSuccess && session.data.signedIn;
  const subject = owner ? session.data.subject : "";
  const asks = useQuery({
    queryKey: ["asks", subject, "needs-you-count"],
    queryFn: async ({ signal }) => {
      const page = await conversationsClient.listAsks({ limit: 1, stateFilter: "awaiting" }, { signal });
      return page.totalAwaiting;
    },
    retry: false,
    staleTime: 30_000,
    refetchInterval: REFRESH_MS,
    enabled: owner,
  });
  const consents = useQuery({
    queryKey: ["native-consents", subject, "needs-you-count"],
    queryFn: async ({ signal }) => {
      const page = await conversationsClient.listNativeConsents({ limit: 25 }, { signal });
      return page.consents.filter((consent) => consent.state === "pending").length;
    },
    retry: false,
    staleTime: 30_000,
    refetchInterval: REFRESH_MS,
    enabled: owner,
  });
  const count = owner ? (asks.data ?? 0) + (consents.data ?? 0) : 0;
  useAppBadge(count);
  return { count, known: owner && (asks.data !== undefined || consents.data !== undefined) };
}

/** Mirror the count on the installed app's icon where the platform allows it. */
function useAppBadge(count: number) {
  useEffect(() => {
    // Not every browser has the Badging API; the tab count still works without it.
    if (!("setAppBadge" in navigator)) return;
    try {
      (count > 0 ? navigator.setAppBadge(count) : navigator.clearAppBadge()).catch(() => undefined);
    } catch {
      // Present but refused (for example outside an installed app): the tab count still shows.
    }
  }, [count]);
}

import { useInfiniteQuery } from "@tanstack/react-query";

import { conversationsClient } from "../../api/notifications";

/**
 * Permission requests owned by Agent Manager, read page by page. The queue page
 * owns this read so that one failed-to-load message can speak for both the
 * decisions and the requests; `NativeConsents` only draws what was read.
 */
export function useNativeConsents(query: string, enabled: boolean) {
  return useInfiniteQuery({
    queryKey: ["native-consents", query], initialPageParam: "", enabled,
    queryFn: ({ pageParam, signal }) => conversationsClient.listNativeConsents({ limit: 25, cursor: pageParam, query }, { signal }),
    getNextPageParam: (page) => page.nextCursor || undefined,
    retry: false, staleTime: 0, refetchInterval: 60_000,
  });
}

export type NativeConsentsRead = ReturnType<typeof useNativeConsents>;

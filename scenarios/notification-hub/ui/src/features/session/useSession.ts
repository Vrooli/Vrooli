import { useEffect, useLayoutEffect, useRef } from "react";
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { identityClient } from "../../api/notifications";
import { advanceSessionGeneration } from "../../api/sessionGeneration";
import { ANONYMOUS_SETTINGS_QUERY_ROOTS, OWNER_ONLY_QUERY_ROOTS } from "./ownerQueryRoots";

export const sessionQueryKey = ["owner-session"] as const;

export function clearRecipientContent(queryClient: QueryClient, subject: string) {
  queryClient.removeQueries({ queryKey: ["asks"] });
  queryClient.removeQueries({ queryKey: ["ask"], predicate: (query) => query.queryKey[2] === subject });
  queryClient.removeQueries({ queryKey: ["ask-explainer", subject] });
  for (const root of OWNER_ONLY_QUERY_ROOTS) queryClient.removeQueries({ queryKey: [root] });
  for (const root of ANONYMOUS_SETTINGS_QUERY_ROOTS) void queryClient.resetQueries({ queryKey: [root] });
  queryClient.removeQueries({ queryKey: ["native-consents"] });
  queryClient.removeQueries({ queryKey: ["updates", "linked-asks"] });
}

/**
 * The owner session lives in an HttpOnly cookie, so the page asks the hub who
 * is signed in. GetSession also renews an expired session from the refresh
 * cookie, which keeps a push tap days later signed in.
 */
export function useSession() {
  return useQuery({
    queryKey: sessionQueryKey,
    queryFn: () => identityClient.getSession({}),
    staleTime: 60_000,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  });
}

/** Hide cached recipient content immediately when any protected API call expires. */
export function useSessionExpiryBoundary() {
  const queryClient = useQueryClient();
  const session = useSession();
  const lastSubject = useRef<string | null>(null);
  useLayoutEffect(() => {
    if (session.isPending || session.isError) return;
    const subject = session.data.signedIn ? session.data.subject : null;
    if (lastSubject.current !== subject) {
      if (lastSubject.current) clearRecipientContent(queryClient, lastSubject.current);
      advanceSessionGeneration();
    }
    lastSubject.current = subject;
  }, [queryClient, session.data, session.isError, session.isPending]);
  useEffect(() => {
    const expire = () => {
      const current = queryClient.getQueryData<{ signedIn?: boolean; subject?: string }>(sessionQueryKey);
      advanceSessionGeneration();
      queryClient.setQueryData(sessionQueryKey, {
        signedIn: false,
        subject: "",
        email: "",
      });
      if (current?.subject) clearRecipientContent(queryClient, current.subject);
    };
    window.addEventListener("notification-hub:session-expired", expire);
    return () => window.removeEventListener("notification-hub:session-expired", expire);
  }, [queryClient]);
}

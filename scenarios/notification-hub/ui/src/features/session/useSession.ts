import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { identityClient } from "../../api/notifications";

export const sessionQueryKey = ["owner-session"] as const;

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
  useEffect(() => {
    const expire = () => {
      queryClient.setQueryData(sessionQueryKey, {
        signedIn: false,
        subject: "",
        email: "",
      });
    };
    window.addEventListener("notification-hub:session-expired", expire);
    return () => window.removeEventListener("notification-hub:session-expired", expire);
  }, [queryClient]);
}

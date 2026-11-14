import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  fetchNotificationCatalog,
  sessionNotificationMuted,
  setSessionNotificationMuted,
  NotificationsUnavailableError,
  WEB_CONSOLE_SOURCE,
} from "../api/notifications";

// The settings Notifications section and the per-session toggle read the same
// catalog, so they share a query key: muting a session here refreshes the
// settings list too, and vice versa.
const CATALOG_QUERY_KEY = ["notification-catalog", WEB_CONSOLE_SOURCE] as const;

export interface SessionNotificationMute {
  /** True when a session-scoped override mutes this session. */
  muted: boolean;
  /** False when Notification Hub is unreachable; callers hide the control. */
  available: boolean;
  /** Flip the per-session override. */
  toggle: () => void;
  /** True while a toggle is in flight. */
  isPending: boolean;
}

/**
 * useSessionNotificationMute exposes a single session's notification override.
 * It is the per-session counterpart to the global category preferences in
 * Settings → Notifications: the override wins for this session alone.
 */
export function useSessionNotificationMute(sessionId: string): SessionNotificationMute {
  const queryClient = useQueryClient();

  const catalog = useQuery({
    queryKey: CATALOG_QUERY_KEY,
    queryFn: () => fetchNotificationCatalog(WEB_CONSOLE_SOURCE),
    staleTime: 30_000,
    retry: (failureCount, error) =>
      !(error instanceof NotificationsUnavailableError) && failureCount < 2,
  });

  const muted = catalog.data ? sessionNotificationMuted(catalog.data, sessionId) : false;
  const available = !(catalog.error instanceof NotificationsUnavailableError);

  const mutation = useMutation({
    mutationFn: (next: boolean) => setSessionNotificationMuted(sessionId, next),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: CATALOG_QUERY_KEY });
    },
  });

  return {
    muted,
    available,
    toggle: () => {
      mutation.mutate(!muted);
    },
    isPending: mutation.isPending,
  };
}

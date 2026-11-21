import { useNotificationPreferences } from "@vrooli/api-base/notifications/react";
import {
  sessionNotificationMuted,
  NotificationsUnavailableError,
  notificationPreferencesClient,
  WEB_CONSOLE_SOURCE,
} from "../api/notifications";

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
  const { catalog, save } = useNotificationPreferences(notificationPreferencesClient, [WEB_CONSOLE_SOURCE]);

  const muted = catalog.data ? sessionNotificationMuted(catalog.data, sessionId) : false;
  const available = !(catalog.error instanceof NotificationsUnavailableError);

  return {
    muted,
    available,
    toggle: () => save.mutate({
      source_scenario: WEB_CONSOLE_SOURCE,
      category: "",
      scope_key: sessionId,
      enabled: muted,
      min_urgency: "",
      channel_override: "",
    }),
    isPending: save.isPending,
  };
}

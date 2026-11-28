// Sanctioned REST exception: RESTReasonBrowserSurface.
//
// This is a same-origin facade to notification-hub's REST notification-
// preference API; Connect-RPC does not apply because the upstream surface is
// itself REST (opaque JSON), so this browser-facing surface stays REST too.
// See docs/internal/SEAMS.md for the full REST-exception registry.
//
// Same-origin facade client for the operator's notification preferences.
//
// The browser never talks to notification-hub directly. It calls Web Console's
// own `/api/v1/notifications/*` routes; the Go facade (api/notifications_proxy.go)
// forwards each request to notification-hub carrying this request's Bearer token
// so the hub validates the SAME owner identity Web Console already holds.
//
// These are plain REST JSON endpoints (not Connect-RPC), so this module uses a
// small authenticated fetch helper that mirrors the Authorization + device
// headers the shared Connect transport attaches in ./client.

import { buildApiUrl } from "@vrooli/api-base";
import {
  createNotificationPreferencesClient,
  notificationPreferenceControlGroups,
  NotificationPreferencesError,
  NotificationPreferencesUnavailableError,
  type NotificationCatalog,
  type NotificationCategory,
  type NotificationPreference,
  type NotificationSource,
} from "@vrooli/api-base/notifications";
import { deviceIdentity } from "../lib/deviceIdentity";
import { getAccessToken } from "../lib/auth";
import { API_BASE, makeApiError } from "./client";

export type NotificationUrgency = string;
export type { NotificationCatalog, NotificationCategory, NotificationPreference, NotificationSource };
export { NotificationPreferencesError, NotificationPreferencesUnavailableError as NotificationsUnavailableError };
export { notificationPreferenceControlGroups };

async function authedFetch(path: string, init?: RequestInit): Promise<Response> {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Device-Id", deviceIdentity().id);
  const token = await getAccessToken();
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    ...init,
    headers,
    cache: "no-store",
  });
  if (res.status === 503) {
    throw new NotificationPreferencesUnavailableError();
  }
  if (!res.ok) {
    throw makeApiError("notifications_error", `notifications request failed (${String(res.status)})`, res.status);
  }
  return res;
}

/** This scenario's own source slug — the source web-console sessions notify under. */
export const WEB_CONSOLE_SOURCE = "web-console";

/** Transport keeps Web Console's owner token and device identity at this boundary. */
export const notificationPreferencesClient = createNotificationPreferencesClient((path, init) =>
  authedFetch(`/api/v1${path}`, init),
);

/**
 * Fetch the notification catalog scoped to ONE producing scenario. A scenario's
 * settings page manages only the categories it declared in its own
 * `.vrooli/notifications.json`; the cross-scenario view belongs to Notification
 * Hub's own Settings page.
 *
 * The hub filters server-side via `?source_scenario=`. The response is filtered
 * again here so an older hub that ignores the parameter can never leak other
 * scenarios' categories into this page.
 */
export async function fetchNotificationCatalog(
  sourceScenario: string = WEB_CONSOLE_SOURCE,
): Promise<NotificationCatalog> {
  return notificationPreferencesClient.getCatalog([sourceScenario]);
}

/**
 * Resolve a person-openable link to another scenario's interface. Web Console's
 * Go server derives the right shape from the origin this browser used — a tunnel
 * subdomain when present, a localhost port otherwise — so the UI never hand-
 * builds a host:port. An empty string means the target has no resolvable URL
 * (not installed / no UI port); the caller should omit the link.
 */
export async function fetchScenarioLink(scenario: string): Promise<string> {
  const res = await authedFetch(
    `/api/v1/external-url?scenario=${encodeURIComponent(scenario)}`,
    { method: "GET" },
  );
  const body = (await res.json()) as { url?: string };
  return typeof body.url === "string" ? body.url : "";
}

export async function updateNotificationPreference(
  pref: NotificationPreference,
): Promise<void> {
  await notificationPreferencesClient.updatePreference(pref);
}

/**
 * Write a per-session notification override. A session-scoped preference (one
 * row keyed on scope_key = the session id) wins over the operator's global
 * web-console preference, so a session can be muted while the rest stay on, or
 * unmuted while the rest are off.
 */
export async function setSessionNotificationMuted(
  sessionId: string,
  muted: boolean,
): Promise<void> {
  await updateNotificationPreference({
    source_scenario: WEB_CONSOLE_SOURCE,
    category: "",
    enabled: !muted,
    min_urgency: "",
    channel_override: "",
    scope_key: sessionId,
  });
}

/**
 * Read whether a session is currently muted by a session-scoped override.
 * Absent an override the session inherits the global preference, reported here
 * as not muted (the override simply does not exist yet).
 */
export function sessionNotificationMuted(
  catalog: NotificationCatalog,
  sessionId: string,
): boolean {
  const override = catalog.preferences.find(
    (p) =>
      p.source_scenario === WEB_CONSOLE_SOURCE &&
      p.category === "" &&
      p.scope_key === sessionId,
  );
  return override ? !override.enabled : false;
}

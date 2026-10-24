import { buildApiUrl } from "@vrooli/api-base";

/**
 * Plain-REST client for the notification-preferences surface. These routes sit
 * behind the hub's owner-auth middleware, so the HttpOnly same-origin session
 * cookie must ride along (credentials: "same-origin") and the browser-session
 * header tells sign-in not to echo tokens into a response body — matching the
 * Connect transport in api/notifications.ts.
 *
 * Paths are resolved against the /api/v1 suffix (appendSuffix: true) so a single
 * path segment ("/notifications/catalog") maps to the server route
 * "/api/v1/notifications/catalog".
 */
export interface CatalogCategory {
  category: string;
  display_name: string;
  description: string;
  declared: boolean;
  default_enabled: boolean;
  default_min_urgency: string;
}

export interface CatalogSource {
  source_scenario: string;
  display_name: string;
  icon: string;
  deep_link_base: string;
  declared: boolean;
  categories: CatalogCategory[];
}

export interface NotificationPreference {
  source_scenario: string;
  category: string;
  enabled: boolean;
  min_urgency: string;
  channel_override: string;
}

export interface NotificationCatalog {
  sources: CatalogSource[];
  preferences: NotificationPreference[];
}

export type MinUrgency = "low" | "normal" | "high" | "critical";

export const MIN_URGENCY_CHOICES: readonly MinUrgency[] = ["low", "normal", "high", "critical"];

function sessionInit(init?: RequestInit): RequestInit {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Browser-Session", "1");
  return { ...init, headers, credentials: "same-origin", cache: "no-store" };
}

export async function fetchNotificationCatalog(): Promise<NotificationCatalog> {
  const response = await fetch(buildApiUrl("/notifications/catalog", { appendSuffix: true }), sessionInit());
  if (!response.ok) throw new Error(`Notification catalog request failed (${response.status})`);
  const body = (await response.json()) as Partial<NotificationCatalog>;
  return { sources: body.sources ?? [], preferences: body.preferences ?? [] };
}

export async function updateNotificationPreference(preference: NotificationPreference): Promise<NotificationPreference> {
  const response = await fetch(
    buildApiUrl("/notifications/preferences", { appendSuffix: true }),
    sessionInit({
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(preference),
    }),
  );
  if (!response.ok) throw new Error(`Notification preference update failed (${response.status})`);
  return (await response.json()) as NotificationPreference;
}

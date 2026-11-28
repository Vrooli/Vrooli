import { buildApiUrl } from "@vrooli/api-base";
import {
  createNotificationPreferencesClient,
  notificationPreferenceControlGroups,
  type NotificationCatalog,
  type NotificationCategory,
  type NotificationPreference,
  type NotificationSource,
} from "@vrooli/api-base/notifications";

/** The Hub uses its same-origin owner session; the shared client owns no credentials. */
function sessionRequest(path: string, init?: RequestInit): Promise<Response> {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Browser-Session", "1");
  return fetch(buildApiUrl(path, { appendSuffix: true }), {
    ...init,
    headers,
    credentials: "same-origin",
    cache: "no-store",
  });
}

export const notificationPreferencesClient = createNotificationPreferencesClient(sessionRequest);

export type CatalogCategory = NotificationCategory;
export type CatalogSource = NotificationSource;
export type { NotificationCatalog, NotificationPreference };
export { notificationPreferenceControlGroups };
export type MinUrgency = "low" | "normal" | "high" | "critical";

export const MIN_URGENCY_CHOICES: readonly MinUrgency[] = ["low", "normal", "high", "critical"];

export function fetchNotificationCatalog(sourceScenarios?: readonly string[]): Promise<NotificationCatalog> {
  return notificationPreferencesClient.getCatalog(sourceScenarios);
}

export function updateNotificationPreference(preference: NotificationPreference): Promise<NotificationPreference> {
  return notificationPreferencesClient.updatePreference(preference);
}

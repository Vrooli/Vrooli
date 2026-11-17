/**
 * Typed browser client for Notification Hub's intentional REST exception.
 * Credential and URL policy stay with the caller: the caller-supplied request
 * function must reach a same-origin facade or the Hub and attach its own auth.
 */

export interface NotificationCategory {
  category: string;
  display_name: string;
  description: string;
  declared: boolean;
  default_enabled: boolean;
  default_min_urgency: string;
}

export interface NotificationSource {
  source_scenario: string;
  display_name: string;
  icon: string;
  deep_link_base: string;
  declared: boolean;
  categories: NotificationCategory[];
}

/** `scope_key` is always present on the wire; an empty string denotes global. */
export interface NotificationPreference {
  source_scenario: string;
  category: string;
  scope_key: string;
  enabled: boolean;
  min_urgency: string;
  channel_override: string;
}

/** Preference writes through a source-pinning facade may omit source_scenario. */
export type NotificationPreferenceUpdate = Omit<NotificationPreference, "source_scenario"> & {
  source_scenario?: string;
  scope_key?: string;
};

export interface NotificationCatalog {
  sources: NotificationSource[];
  preferences: NotificationPreference[];
}

export type NotificationRequest = (path: string, init?: RequestInit) => Promise<Response>;

export interface NotificationPreferencesClientOptions {
  /** Maximum wall time for a request and response body. Defaults to 15 seconds and is capped at 60 seconds. */
  timeoutMs?: number;
  /** Maximum bytes accepted from any response. Defaults to 1 MiB. */
  maxResponseBytes?: number;
}

const DEFAULT_TIMEOUT_MS = 15_000;
const MAX_TIMEOUT_MS = 60_000;
const DEFAULT_MAX_RESPONSE_BYTES = 1024 * 1024;

export class NotificationPreferencesError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = "NotificationPreferencesError";
  }
}

export class NotificationPreferencesUnavailableError extends NotificationPreferencesError {
  constructor(message = "Notification Hub is unavailable") {
    super(message, 503);
    this.name = "NotificationPreferencesUnavailableError";
  }
}

function object(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null;
}

function requiredString(value: unknown, field: string): string {
  if (typeof value !== "string") throw new Error(`Notification Hub returned an invalid ${field}`);
  return value;
}

function requiredBoolean(value: unknown, field: string): boolean {
  if (typeof value !== "boolean") throw new Error(`Notification Hub returned an invalid ${field}`);
  return value;
}

function parseCategory(value: unknown): NotificationCategory {
  const item = object(value);
  if (!item) throw new Error("Notification Hub returned an invalid category");
  return {
    category: requiredString(item.category, "category key"),
    display_name: requiredString(item.display_name, "category name"),
    description: requiredString(item.description, "category description"),
    declared: requiredBoolean(item.declared, "category declaration"),
    default_enabled: requiredBoolean(item.default_enabled, "category default"),
    default_min_urgency: requiredString(item.default_min_urgency, "category urgency"),
  };
}

function parseSource(value: unknown): NotificationSource {
  const item = object(value);
  if (!item || !Array.isArray(item.categories)) throw new Error("Notification Hub returned an invalid source");
  return {
    source_scenario: requiredString(item.source_scenario, "source scenario"),
    display_name: requiredString(item.display_name, "source name"),
    icon: requiredString(item.icon, "source icon"),
    deep_link_base: requiredString(item.deep_link_base, "source link"),
    declared: requiredBoolean(item.declared, "source declaration"),
    categories: item.categories.map(parseCategory),
  };
}

export function parseNotificationPreference(value: unknown): NotificationPreference {
  const item = object(value);
  if (!item) throw new Error("Notification Hub returned an invalid preference");
  return {
    source_scenario: requiredString(item.source_scenario, "preference source"),
    category: requiredString(item.category, "preference category"),
    scope_key: requiredString(item.scope_key, "preference scope"),
    enabled: requiredBoolean(item.enabled, "preference enabled state"),
    min_urgency: requiredString(item.min_urgency, "preference urgency"),
    channel_override: requiredString(item.channel_override, "preference channel"),
  };
}

function parsePreferenceUpdate(value: NotificationPreferenceUpdate): NotificationPreferenceUpdate {
  const item = object(value);
  if (!item) throw new Error("Notification preference update is invalid");
  const parsed: NotificationPreferenceUpdate = {
    category: requiredString(item.category, "preference category"),
    scope_key: typeof item.scope_key === "undefined" ? "" : requiredString(item.scope_key, "preference scope"),
    enabled: requiredBoolean(item.enabled, "preference enabled state"),
    min_urgency: requiredString(item.min_urgency, "preference urgency"),
    channel_override: requiredString(item.channel_override, "preference channel"),
  };
  if (typeof item.source_scenario !== "undefined") {
    parsed.source_scenario = requiredString(item.source_scenario, "preference source");
  }
  return parsed;
}

function parseCatalog(value: unknown): NotificationCatalog {
  const item = object(value);
  if (!item || !Array.isArray(item.sources) || !Array.isArray(item.preferences)) {
    throw new Error("Notification Hub returned an invalid catalog");
  }
  return {
    sources: item.sources.map(parseSource),
    preferences: item.preferences.map(parseNotificationPreference),
  };
}

function sourceQuery(sourceScenarios?: readonly string[]): string {
  if (!sourceScenarios?.length) return "";
  const query = new URLSearchParams();
  for (const source of sourceScenarios) {
    const normalized = source.trim();
    if (!normalized) throw new Error("Notification source scope must not be empty");
    query.append("source_scenario", normalized);
  }
  return `?${query.toString()}`;
}

async function boundedResponseText(response: Response, maxBytes: number, signal: AbortSignal): Promise<string> {
  const contentLength = response.headers.get("content-length");
  if (contentLength && /^\d+$/.test(contentLength) && Number(contentLength) > maxBytes) {
    throw new NotificationPreferencesError("Notification response is too large", 502);
  }
  if (!response.body) return "";
  const reader = response.body.getReader();
  const cancelOnAbort = () => { void reader.cancel(); };
  signal.addEventListener("abort", cancelOnAbort, { once: true });
  const chunks: Uint8Array[] = [];
  let bytes = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      bytes += value.byteLength;
      if (bytes > maxBytes) {
        await reader.cancel();
        throw new NotificationPreferencesError("Notification response is too large", 502);
      }
      chunks.push(value);
    }
  } finally {
    signal.removeEventListener("abort", cancelOnAbort);
    reader.releaseLock();
  }
  const body = new Uint8Array(bytes);
  let offset = 0;
  for (const chunk of chunks) {
    body.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(body);
}

async function readJSON(
  request: NotificationRequest,
  path: string,
  init: RequestInit | undefined,
  timeoutMs: number,
  maxResponseBytes: number,
): Promise<unknown> {
  const controller = new AbortController();
  const signals = [controller.signal, init?.signal].filter((signal): signal is AbortSignal => Boolean(signal));
  const signal = signals.length === 1 ? signals[0] : AbortSignal.any(signals);
  let timeout: ReturnType<typeof setTimeout> | undefined;
  try {
    const timedOut = new Promise<never>((_, reject) => {
      timeout = setTimeout(() => {
        controller.abort();
        reject(new NotificationPreferencesUnavailableError("Notification Hub request timed out"));
      }, timeoutMs);
    });
    const response = await Promise.race([
      request(path, { ...init, signal }),
      timedOut,
    ]);
    if (response.status === 503) throw new NotificationPreferencesUnavailableError();
    if (!response.ok) throw new NotificationPreferencesError(`Notification request failed (${response.status})`, response.status);
    let text: string;
    try {
      text = await Promise.race([boundedResponseText(response, maxResponseBytes, signal), timedOut]);
    } catch (error) {
      if (error instanceof NotificationPreferencesError) throw error;
      throw new NotificationPreferencesError("Notification Hub returned an invalid response", response.status);
    }
    try {
      return JSON.parse(text) as unknown;
    } catch {
      throw new NotificationPreferencesError("Notification Hub returned an invalid response", response.status);
    }
  } catch (error) {
    if (error instanceof NotificationPreferencesError) throw error;
    throw new NotificationPreferencesUnavailableError(error instanceof Error ? error.message : undefined);
  } finally {
    if (timeout) clearTimeout(timeout);
  }
}

/** Build the shared typed client while leaving credentials, origin and headers to its caller. */
export function createNotificationPreferencesClient(
  request: NotificationRequest,
  options: NotificationPreferencesClientOptions = {},
) {
  const timeoutMs = Math.min(options.timeoutMs ?? DEFAULT_TIMEOUT_MS, MAX_TIMEOUT_MS);
  const maxResponseBytes = options.maxResponseBytes ?? DEFAULT_MAX_RESPONSE_BYTES;
  if (!Number.isFinite(timeoutMs) || timeoutMs <= 0) throw new Error("Notification timeout must be positive");
  if (!Number.isSafeInteger(maxResponseBytes) || maxResponseBytes <= 0) throw new Error("Notification response limit must be a positive integer");
  const read = (path: string, init?: RequestInit) => readJSON(request, path, init, timeoutMs, maxResponseBytes);
  return {
    async getCatalog(sourceScenarios?: readonly string[]): Promise<NotificationCatalog> {
      const catalog = parseCatalog(await read(`/notifications/catalog${sourceQuery(sourceScenarios)}`));
      if (!sourceScenarios?.length) return catalog;
      const requested = new Set(sourceScenarios);
      const sources = catalog.sources.filter((source) => requested.has(source.source_scenario));
      return {
        sources,
        preferences: catalog.preferences.filter((pref) => requested.has(pref.source_scenario)),
      };
    },
    async getPreferences(sourceScenarios?: readonly string[]): Promise<NotificationPreference[]> {
      const body = object(await read(`/notifications/preferences${sourceQuery(sourceScenarios)}`));
      if (!body || !Array.isArray(body.preferences)) throw new Error("Notification Hub returned invalid preferences");
      const preferences = body.preferences.map(parseNotificationPreference);
      return sourceScenarios?.length
        ? preferences.filter((pref) => sourceScenarios.includes(pref.source_scenario))
        : preferences;
    },
    async updatePreference(preference: NotificationPreferenceUpdate): Promise<NotificationPreference> {
      const parsed = parsePreferenceUpdate(preference);
      return parseNotificationPreference(await read("/notifications/preferences", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(parsed),
      }));
    },
  };
}

export type NotificationPreferencesClient = ReturnType<typeof createNotificationPreferencesClient>;

export const notificationPreferenceKey = (source: string, category: string, scope = "") => `${source}\u0000${category}\u0000${scope}`;

export interface NotificationPreferenceControlRow {
  id: string;
  sourceScenario: string;
  category: string;
  label: string;
  description: string;
  enabled: boolean;
  minUrgency: string;
  onEnabledChange: (enabled: boolean) => void;
  onMinUrgencyChange: (urgency: string) => void;
}

export interface NotificationPreferenceControlGroup {
  id: string;
  testId?: string;
  label: string;
  rows: NotificationPreferenceControlRow[];
}

/** Build the shared, source-scoped controls from global preferences only. */
export function notificationPreferenceControlGroups(
  sources: readonly NotificationSource[],
  preferences: readonly NotificationPreference[],
  allLabel: string,
  onChange: (preference: NotificationPreference) => void,
): NotificationPreferenceControlGroup[] {
  const globalPreferences = preferences.filter((preference) => preference.scope_key === "");
  return sources.map((source) => ({
    id: source.source_scenario,
    label: source.display_name || source.source_scenario,
    rows: categoriesForNotificationSource(source, allLabel).map((category) => {
      const preference = effectiveNotificationPreference(source, category, globalPreferences);
      return {
        id: `${source.source_scenario}-${category.category}`,
        sourceScenario: source.source_scenario,
        category: category.category,
        label: notificationCategoryLabel(category, allLabel),
        description: category.description,
        enabled: preference.enabled,
        minUrgency: preference.min_urgency,
        onEnabledChange: (enabled: boolean) => onChange({ ...preference, enabled }),
        onMinUrgencyChange: (min_urgency: string) => onChange({ ...preference, min_urgency }),
      };
    }),
  }));
}

/** A source without declared categories has one source-wide preference row. */
export function categoriesForNotificationSource(source: NotificationSource, allLabel: string): NotificationCategory[] {
  if (source.categories.length > 0) return source.categories;
  return [{ category: "", display_name: allLabel, description: "", declared: source.declared, default_enabled: true, default_min_urgency: "normal" }];
}

export function notificationCategoryLabel(category: NotificationCategory, allLabel: string): string {
  if (category.display_name) return category.display_name;
  const words = category.category.replace(/[._-]+/g, " ").trim();
  return words ? words.charAt(0).toUpperCase() + words.slice(1) : allLabel;
}

/** Resolve global preference state while leaving per-session scope rows to their owner UI. */
export function effectiveNotificationPreference(
  source: NotificationSource,
  category: NotificationCategory,
  preferences: readonly NotificationPreference[],
): NotificationPreference {
  return preferences.find((preference) =>
    preference.source_scenario === source.source_scenario &&
    preference.category === category.category &&
    preference.scope_key === "",
  ) ?? {
    source_scenario: source.source_scenario,
    category: category.category,
    scope_key: "",
    enabled: category.default_enabled,
    min_urgency: category.default_min_urgency || "normal",
    channel_override: "",
  };
}

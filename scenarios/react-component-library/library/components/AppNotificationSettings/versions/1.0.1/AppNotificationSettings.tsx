/**
 * @libraryId react-component-library:AppNotificationSettings
 * @displayName App notification settings
 * @description A scenario's own notification settings: show its notifications in its installed app instead of Chimesail on this device, plus on/off and minimum urgency for each of its notification types.
 * @version 1.0.1
 * @tags ["notifications","settings"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { forwardRef, useCallback, useEffect, useState, type ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { RadioGroup } from "@vrooli/react-component-library/RadioGroup/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import {
  APP_NOTIFICATION_STRINGS,
  type AppNotificationStrings,
} from "./AppNotificationSettings.strings";

/**
 * Browser side of app notifications: a scenario's own installed app showing
 * that scenario's notifications on this device instead of Chimesail.
 *
 * Everything goes through the scenario's own web server at /vrooli-notify
 * (@vrooli/api-base/server `appNotifications`), which serves the shared push
 * worker and forwards to Notification Hub's AppDeliveryService with the
 * scenario named server-side. Contract:
 * scenarios/notification-hub/docs/concepts/NOTIFICATIONS.md §"App delivery".
 */

export const APP_NOTIFICATIONS_PREFIX = "/vrooli-notify";
const WORKER_URL = `${APP_NOTIFICATIONS_PREFIX}/sw.js`;
const WORKER_SCOPE = `${APP_NOTIFICATIONS_PREFIX}/`;
const PAIRING_KEY = "vrooli-notify:pairing";
const STASH_CACHE = "vrooli-notify";
const STASH_KEY = `${APP_NOTIFICATIONS_PREFIX}/pending-navigation`;
const STASH_MAX_AGE_MS = 2 * 60 * 1000;
const CONFIRM_INTERVAL_MS = 60 * 60 * 1000;

export interface AppSubscriptionState {
  id: string;
  deviceGroup: string;
  deviceLabel: string;
  permission: string;
  confirmedAt: string;
  healthy: boolean;
  unhealthyReason: string;
  standalone: boolean;
}

export interface DeviceChoice {
  group: string;
  label: string;
  lastUsed: string;
  bestMatch: boolean;
  apps: string[];
}

export interface AppNotificationState {
  scenario: string;
  displayName: string;
  vapidPublicKey: string;
  eligible: boolean;
  ineligibleReason: string;
  publicUrl: string;
  subscription: AppSubscriptionState | null;
  devices: DeviceChoice[];
}

/** Where this app should show notifications: paired with a device that has Chimesail, or on its own. */
export type DevicePairing = { group: string } | { standalone: true };

export type AppNotificationSupport = "supported" | "install-first" | "unsupported";

export class AppNotificationsError extends Error {
  constructor(
    message: string,
    readonly code: string = "unknown",
  ) {
    super(message);
    this.name = "AppNotificationsError";
  }
}

/**
 * Whether this page can receive notifications as an app. iPhone and iPad
 * allow web notifications only in an app opened from the Home Screen.
 */
export function appNotificationSupport(): AppNotificationSupport {
  if (typeof window === "undefined" || typeof navigator === "undefined") return "unsupported";
  const ios = /iPhone|iPad|iPod/.test(navigator.userAgent);
  const standalone =
    (navigator as Navigator & { standalone?: boolean }).standalone === true ||
    (typeof window.matchMedia === "function" &&
      window.matchMedia("(display-mode: standalone)").matches);
  if (ios && !standalone) return "install-first";
  if (
    !("serviceWorker" in navigator) ||
    !("PushManager" in window) ||
    typeof Notification === "undefined"
  ) {
    return ios ? "install-first" : "unsupported";
  }
  return "supported";
}

const NOT_ENABLED_MESSAGE = "App notifications are not turned on for this scenario.";

function notEnabled(): AppNotificationsError {
  return new AppNotificationsError(NOT_ENABLED_MESSAGE, "not-enabled");
}

/**
 * A scenario without `"appDelivery": true` does not mount /vrooli-notify, so
 * its server answers with a plain-text 404 or an HTML app shell. Any body that
 * is not JSON from a 404 or a success means the routes are absent; any other
 * non-JSON answer is reported by status, never as a raw parse error.
 */
function parseJSONBody(res: Response, text: string): Record<string, unknown> {
  try {
    const value: unknown = JSON.parse(text);
    if (value !== null && typeof value === "object" && !Array.isArray(value)) {
      return value as Record<string, unknown>;
    }
  } catch {
    // Fall through: not a JSON object.
  }
  if (res.ok || res.status === 404) throw notEnabled();
  throw new AppNotificationsError(
    `Notifications answered ${String(res.status)}.`,
    String(res.status),
  );
}

async function rpc<T>(method: string, body: Record<string, unknown>): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${APP_NOTIFICATIONS_PREFIX}/rpc/${method}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "Connect-Protocol-Version": "1" },
      body: JSON.stringify(body),
      cache: "no-store",
      credentials: "same-origin",
    });
  } catch {
    throw new AppNotificationsError(
      "Notifications could not be reached. Check your connection and try again.",
      "unavailable",
    );
  }
  const text = await res.text();
  if (!text && res.status === 404) throw notEnabled();
  const parsed = text ? parseJSONBody(res, text) : {};
  if (!res.ok) {
    const message =
      typeof parsed.message === "string" && parsed.message
        ? parsed.message
        : `Notifications answered ${String(res.status)}.`;
    throw new AppNotificationsError(
      message,
      typeof parsed.code === "string" ? parsed.code : String(res.status),
    );
  }
  return parsed as T;
}

/** AppState as Connect's JSON form sends it: false, empty and zero values are omitted. */
interface RawAppState {
  scenario?: string;
  displayName?: string;
  vapidPublicKey?: string;
  eligible?: boolean;
  ineligibleReason?: string;
  publicUrl?: string;
  subscription?: Partial<AppSubscriptionState>;
  devices?: Partial<DeviceChoice>[];
}

function normalizeState(raw: RawAppState): AppNotificationState {
  const sub = raw.subscription;
  return {
    scenario: raw.scenario ?? "",
    displayName: raw.displayName ?? raw.scenario ?? "",
    vapidPublicKey: raw.vapidPublicKey ?? "",
    eligible: raw.eligible ?? false,
    ineligibleReason: raw.ineligibleReason ?? "",
    publicUrl: raw.publicUrl ?? "",
    subscription: sub
      ? {
          id: sub.id ?? "",
          deviceGroup: sub.deviceGroup ?? "",
          deviceLabel: sub.deviceLabel ?? "",
          permission: sub.permission ?? "",
          confirmedAt: sub.confirmedAt ?? "",
          healthy: sub.healthy ?? false,
          unhealthyReason: sub.unhealthyReason ?? "",
          standalone: sub.standalone ?? false,
        }
      : null,
    devices: (raw.devices ?? []).map((d) => ({
      group: d.group ?? "",
      label: d.label ?? "",
      lastUsed: d.lastUsed ?? "",
      bestMatch: d.bestMatch ?? false,
      apps: d.apps ?? [],
    })),
  };
}

async function registration(): Promise<ServiceWorkerRegistration | undefined> {
  if (!("serviceWorker" in navigator)) return undefined;
  return navigator.serviceWorker.getRegistration(WORKER_SCOPE);
}

/** This app's current push subscription, if notifications were turned on here. */
export async function currentAppSubscription(): Promise<PushSubscription | null> {
  const reg = await registration();
  return reg ? reg.pushManager.getSubscription() : null;
}

export async function getAppNotificationState(): Promise<AppNotificationState> {
  const sub =
    appNotificationSupport() === "supported"
      ? await currentAppSubscription().catch(() => null)
      : null;
  const raw = await rpc<RawAppState>("GetAppState", {
    endpoint: sub?.endpoint ?? "",
    userAgent: navigator.userAgent,
  });
  return normalizeState(raw);
}

function keyBytes(value: string): Uint8Array<ArrayBuffer> {
  const base64 = (value + "=".repeat((4 - (value.length % 4)) % 4))
    .replace(/-/g, "+")
    .replace(/_/g, "/");
  const binary = atob(base64);
  const bytes = new Uint8Array(new ArrayBuffer(binary.length));
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

async function activeRegistration(): Promise<ServiceWorkerRegistration> {
  // The worker's scope excludes every page, so navigator.serviceWorker.ready
  // would never resolve; wait on this registration's own worker instead.
  const reg = await navigator.serviceWorker.register(WORKER_URL, { scope: WORKER_SCOPE });
  if (reg.active) return reg;
  const worker = reg.installing ?? reg.waiting;
  if (!worker)
    throw new AppNotificationsError(
      "The notification helper did not install. Reload the app and try again.",
      "worker",
    );
  await new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(
        new AppNotificationsError(
          "The notification helper took too long to start. Try again.",
          "worker",
        ),
      );
    }, 20_000);
    worker.addEventListener("statechange", () => {
      if (worker.state === "activated") {
        clearTimeout(timer);
        resolve();
      }
      if (worker.state === "redundant") {
        clearTimeout(timer);
        reject(
          new AppNotificationsError(
            "The notification helper failed to install. Reload the app and try again.",
            "worker",
          ),
        );
      }
    });
  });
  return reg;
}

/**
 * Ask permission and create this app's push subscription. Call it straight
 * from a tap: iPhone only shows the permission prompt for a user gesture.
 */
export async function prepareAppSubscription(vapidPublicKey: string): Promise<PushSubscription> {
  const permission = await Notification.requestPermission();
  if (permission !== "granted") {
    throw new AppNotificationsError(
      permission === "denied"
        ? "Notifications are blocked for this app. Allow them in your device settings, then try again."
        : "Notifications were not allowed. Tap the button again and choose Allow.",
      "permission",
    );
  }
  const reg = await activeRegistration();
  const existing = await reg.pushManager.getSubscription();
  if (existing) return existing;
  if (!vapidPublicKey)
    throw new AppNotificationsError(
      "Notification Hub did not provide its key. Try again shortly.",
      "key",
    );
  return reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: keyBytes(vapidPublicKey),
  });
}

function savePairing(pairing: DevicePairing | null) {
  try {
    if (pairing) localStorage.setItem(PAIRING_KEY, JSON.stringify(pairing));
    else localStorage.removeItem(PAIRING_KEY);
  } catch {
    // Without storage the app still works; it only cannot re-pair by itself.
  }
}

function loadPairing(): DevicePairing | null {
  try {
    const raw = localStorage.getItem(PAIRING_KEY);
    return raw ? (JSON.parse(raw) as DevicePairing) : null;
  } catch {
    return null;
  }
}

async function register(
  subscription: PushSubscription,
  pairing: DevicePairing,
): Promise<AppNotificationState> {
  const json = subscription.toJSON();
  const raw = await rpc<RawAppState>("EnableApp", {
    endpoint: subscription.endpoint,
    p256dh: json.keys?.p256dh ?? "",
    auth: json.keys?.auth ?? "",
    origin: window.location.origin,
    userAgent: navigator.userAgent,
    permission: Notification.permission,
    pairWithGroup: "group" in pairing ? pairing.group : "",
    standalone: "standalone" in pairing,
  });
  savePairing(pairing);
  return normalizeState(raw);
}

/** Pair this app with a device; from then on that device shows this scenario's notifications here. */
export async function enableAppNotifications(
  subscription: PushSubscription,
  pairing: DevicePairing,
): Promise<AppNotificationState> {
  return register(subscription, pairing);
}

/** Stop showing notifications in this app; the device goes back to Chimesail. */
export async function disableAppNotifications(): Promise<void> {
  const sub = await currentAppSubscription().catch(() => null);
  if (sub) {
    await rpc("DisableApp", { endpoint: sub.endpoint });
    await sub.unsubscribe().catch(() => false);
  }
  savePairing(null);
}

/** Tell the hub this app is still installed and whether it may still notify. */
export async function confirmAppNotifications(): Promise<boolean> {
  const reg = await registration();
  if (!reg) return false;
  const pairing = loadPairing();
  let sub = await reg.pushManager.getSubscription();
  if (!sub) {
    // The browser dropped the subscription (iOS does this occasionally). If
    // the operator had turned notifications on here, quietly make a new one.
    if (!pairing || Notification.permission !== "granted") return false;
    const state = await getAppNotificationState();
    if (!state.vapidPublicKey) return false;
    sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: keyBytes(state.vapidPublicKey),
    });
    await register(sub, pairing);
    return true;
  }
  const result = await rpc<{ registered?: boolean; healthy?: boolean }>("ConfirmApp", {
    endpoint: sub.endpoint,
    permission: Notification.permission,
    userAgent: navigator.userAgent,
  });
  if (!result.registered && pairing && Notification.permission === "granted") {
    await register(sub, pairing);
    return true;
  }
  return result.healthy ?? false;
}

export interface InitAppNotificationsOptions {
  /**
   * Open an in-app URL a notification tap asked for. Return true when handled
   * (for example by the app's router); otherwise the page navigates to it.
   */
  onNavigate?: (url: string) => boolean;
}

/**
 * Call once when the app starts. It re-confirms the app with Notification Hub
 * (at most hourly) and opens the screen a notification tap asked for, which
 * iPhone does not always do by itself when the app was already running.
 * Returns a cleanup function. Does nothing until notifications were turned on.
 */
export function initAppNotifications(options: InitAppNotificationsOptions = {}): () => void {
  if (
    typeof window === "undefined" ||
    typeof navigator === "undefined" ||
    !("serviceWorker" in navigator)
  )
    return () => {};
  let lastConfirm = 0;

  const navigate = (url: unknown) => {
    if (typeof url !== "string") return;
    let target: URL;
    try {
      target = new URL(url, window.location.origin);
    } catch {
      return;
    }
    if (target.origin !== window.location.origin || target.href === window.location.href) return;
    if (options.onNavigate?.(target.href)) return;
    window.location.assign(target.href);
  };

  const consumeStash = async () => {
    if (typeof caches === "undefined") return;
    const cache = await caches.open(STASH_CACHE);
    const hit = await cache.match(STASH_KEY);
    if (!hit) return;
    await cache.delete(STASH_KEY);
    const stash = (await hit.json()) as { url?: string; at?: number };
    if (typeof stash.at === "number" && Date.now() - stash.at < STASH_MAX_AGE_MS)
      navigate(stash.url);
  };

  const confirm = () => {
    if (Date.now() - lastConfirm < CONFIRM_INTERVAL_MS) return;
    lastConfirm = Date.now();
    void confirmAppNotifications().catch(() => {
      lastConfirm = 0;
    });
  };

  const onMessage = (event: MessageEvent) => {
    const data = event.data as { type?: string; url?: unknown } | null;
    if (data?.type === "vrooli-notify:navigate") navigate(data.url);
  };
  const onVisible = () => {
    if (document.visibilityState !== "visible") return;
    void consumeStash().catch(() => undefined);
    confirm();
  };

  navigator.serviceWorker.addEventListener("message", onMessage);
  document.addEventListener("visibilitychange", onVisible);
  onVisible();
  return () => {
    navigator.serviceWorker.removeEventListener("message", onMessage);
    document.removeEventListener("visibilitychange", onVisible);
  };
}

export interface NotificationCategoryView {
  category: string;
  displayName: string;
  description: string;
  enabled: boolean;
  minUrgency: string;
}

interface CatalogBody {
  sources?: {
    source_scenario: string;
    categories?: {
      category: string;
      display_name: string;
      description: string;
      default_enabled: boolean;
      default_min_urgency: string;
    }[];
  }[];
  preferences?: {
    source_scenario: string;
    category: string;
    scope_key?: string;
    enabled: boolean;
    min_urgency: string;
  }[];
}

/** This scenario's notification types and the operator's current choice for each. */
export async function fetchAppNotificationCategories(): Promise<NotificationCategoryView[]> {
  const res = await fetch(`${APP_NOTIFICATIONS_PREFIX}/catalog`, {
    cache: "no-store",
    credentials: "same-origin",
  });
  const text = await res.text();
  if (res.status === 404 && !text) throw notEnabled();
  let body: CatalogBody;
  try {
    body = parseJSONBody(res, text);
  } catch (error) {
    if (error instanceof AppNotificationsError && error.code === "not-enabled") throw error;
    throw new AppNotificationsError("Notification types could not be loaded.", String(res.status));
  }
  if (!res.ok)
    throw new AppNotificationsError("Notification types could not be loaded.", String(res.status));
  const source = body.sources?.[0];
  if (!source) return [];
  return (source.categories ?? []).map((c) => {
    const pref = body.preferences?.find((p) => p.category === c.category && !p.scope_key);
    return {
      category: c.category,
      displayName: c.display_name || c.category,
      description: c.description,
      enabled: pref ? pref.enabled : c.default_enabled,
      minUrgency: (pref ? pref.min_urgency : c.default_min_urgency) || "normal",
    };
  });
}

export async function saveAppNotificationCategory(view: NotificationCategoryView): Promise<void> {
  const res = await fetch(`${APP_NOTIFICATIONS_PREFIX}/preferences`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    credentials: "same-origin",
    body: JSON.stringify({
      category: view.category,
      enabled: view.enabled,
      min_urgency: view.minUrgency,
      channel_override: "",
      scope_key: "",
    }),
  });
  if (!res.ok)
    throw new AppNotificationsError("That change was not saved. Try again.", String(res.status));
}

/**
 * Everything the settings panel asks of the browser and the scenario's
 * /vrooli-notify routes. The default is the real browser client; previews and
 * tests pass a deterministic one instead.
 */
export interface AppNotificationClient {
  support(): AppNotificationSupport;
  getState(): Promise<AppNotificationState>;
  prepareSubscription(vapidPublicKey: string): Promise<PushSubscription>;
  enable(subscription: PushSubscription, pairing: DevicePairing): Promise<AppNotificationState>;
  disable(): Promise<void>;
  confirm(): Promise<boolean>;
  fetchCategories(): Promise<NotificationCategoryView[]>;
  saveCategory(view: NotificationCategoryView): Promise<void>;
}

/** The real client: this page's service worker, push and Notification APIs, and its /vrooli-notify routes. */
export const browserAppNotificationClient: AppNotificationClient = {
  support: appNotificationSupport,
  getState: getAppNotificationState,
  prepareSubscription: prepareAppSubscription,
  enable: enableAppNotifications,
  disable: disableAppNotifications,
  confirm: confirmAppNotifications,
  fetchCategories: fetchAppNotificationCategories,
  saveCategory: saveAppNotificationCategory,
};

export const appNotificationSettingsStyles = `
[data-rcl-app-notify] { display: grid; gap: var(--space-sm, 0.75rem); min-inline-size: 0; }
[data-rcl-app-notify-title] { margin: 0; color: var(--color-foreground); font: var(--text-body, inherit); font-weight: 700; }
[data-rcl-app-notify-text] { margin: 0; color: var(--color-muted-foreground); font: var(--text-caption, inherit); max-inline-size: 65ch; }
[data-rcl-app-notify-stack] { display: grid; gap: var(--space-sm, 0.75rem); min-inline-size: 0; }
[data-rcl-app-notify-row], [data-rcl-app-notify-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs, 0.5rem); }
[data-rcl-app-notify-message] { margin: 0; color: var(--color-danger); font: var(--text-caption, inherit); }
[data-rcl-app-notify-category] { display: grid; grid-template-columns: minmax(0, 1fr) minmax(8rem, 10rem); gap: var(--space-sm, 0.75rem); align-items: start; padding-block: var(--space-xs, 0.5rem); border-block-start: var(--border-hairline, 1px) solid var(--color-border); }
@media (max-width: 480px) { [data-rcl-app-notify-category] { grid-template-columns: minmax(0, 1fr); } }
`;

export type { AppNotificationStrings };
// One import gives a scenario the panel, its English strings and the start-up hook.
export { APP_NOTIFICATION_STRINGS };

export interface AppNotificationSettingsProps {
  /** Show this scenario's notification types with on/off and minimum urgency (default true). */
  showCategories?: boolean;
  strings?: Partial<AppNotificationStrings>;
  className?: string;
  /**
   * Where the panel reads and changes app notifications. Defaults to this
   * page's browser APIs and its /vrooli-notify routes; read once at mount.
   */
  client?: AppNotificationClient;
}

function fill(template: string, values: Record<string, string>): string {
  return template.replace(/\{(\w+)\}/g, (_, key: string) => values[key] ?? "");
}

type Phase =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "not-enabled" }
  | { kind: "ready"; state: AppNotificationState }
  | {
      kind: "choosing";
      state: AppNotificationState;
      subscription: PushSubscription;
      choice: string;
    };

const STANDALONE = "__standalone__";

/**
 * A scenario's own notification settings: whether its installed app shows its
 * notifications on this device instead of Chimesail, and (optionally) which of
 * its notification types reach the operator. Needs the scenario web server's
 * app notifications (`"appDelivery": true` in .vrooli/notifications.json).
 */
export const AppNotificationSettings = forwardRef<HTMLElement, AppNotificationSettingsProps>(
  function AppNotificationSettings({ showCategories = true, strings, className, client }, ref) {
    useLibraryStyleSheet(
      "react-component-library:AppNotificationSettings",
      "1.0.1",
      appNotificationSettingsStyles,
    );
    const s: AppNotificationStrings = { ...APP_NOTIFICATION_STRINGS, ...strings };
    const [api] = useState<AppNotificationClient>(() => client ?? browserAppNotificationClient);
    const support = api.support();
    const [phase, setPhase] = useState<Phase>({ kind: "loading" });
    const [busy, setBusy] = useState(false);
    const [message, setMessage] = useState("");
    const [categories, setCategories] = useState<NotificationCategoryView[] | null>(null);
    const [categoriesError, setCategoriesError] = useState(false);

    const load = useCallback(async () => {
      setPhase({ kind: "loading" });
      try {
        setPhase({ kind: "ready", state: await api.getState() });
      } catch (error) {
        if (error instanceof AppNotificationsError && error.code === "not-enabled") {
          setPhase({ kind: "not-enabled" });
          return;
        }
        setPhase({ kind: "error", message: error instanceof Error ? error.message : s.loadError });
      }
    }, [s.loadError, api]);

    useEffect(() => {
      void load();
    }, [load]);

    useEffect(() => {
      if (!showCategories) return;
      api.fetchCategories().then(setCategories, () => {
        setCategoriesError(true);
      });
    }, [showCategories, api]);

    // A scenario without "appDelivery": true has no /vrooli-notify routes; the
    // panel stays invisible until it opts in.
    if (phase.kind === "not-enabled") return null;

    const app = phase.kind === "ready" || phase.kind === "choosing" ? phase.state.displayName : "";

    const turnOn = async (state: AppNotificationState) => {
      setBusy(true);
      setMessage("");
      try {
        const subscription = await api.prepareSubscription(state.vapidPublicKey);
        const best = state.devices.find((d) => d.bestMatch) ?? state.devices[0];
        setPhase({ kind: "choosing", state, subscription, choice: best ? best.group : STANDALONE });
      } catch (error) {
        setMessage(error instanceof Error ? error.message : String(error));
      } finally {
        setBusy(false);
      }
    };

    const confirmChoice = async () => {
      if (phase.kind !== "choosing") return;
      setBusy(true);
      setMessage("");
      try {
        const pairing =
          phase.choice === STANDALONE ? { standalone: true as const } : { group: phase.choice };
        setPhase({ kind: "ready", state: await api.enable(phase.subscription, pairing) });
      } catch (error) {
        setMessage(error instanceof Error ? error.message : String(error));
      } finally {
        setBusy(false);
      }
    };

    const turnOff = async () => {
      setBusy(true);
      setMessage("");
      try {
        await api.disable();
        await load();
      } catch (error) {
        setMessage(error instanceof Error ? error.message : String(error));
      } finally {
        setBusy(false);
      }
    };

    const checkAgain = async () => {
      setBusy(true);
      try {
        await api.confirm();
        await load();
      } finally {
        setBusy(false);
      }
    };

    const saveCategory = async (next: NotificationCategoryView) => {
      const previous = categories;
      setCategories(
        (rows) => rows?.map((row) => (row.category === next.category ? next : row)) ?? null,
      );
      try {
        await api.saveCategory(next);
      } catch {
        setCategories(previous);
        setMessage(s.saveError);
      }
    };

    let body: ReactNode;
    if (support === "install-first") {
      body = <p data-rcl-app-notify-text>{s.installFirst}</p>;
    } else if (support === "unsupported") {
      body = <p data-rcl-app-notify-text>{s.unsupported}</p>;
    } else if (phase.kind === "loading") {
      body = (
        <p data-rcl-app-notify-text role="status">
          {s.loading}
        </p>
      );
    } else if (phase.kind === "error") {
      body = (
        <div data-rcl-app-notify-row>
          <p data-rcl-app-notify-text>{phase.message || s.loadError}</p>
          <Button variant="secondary" onClick={() => void load()}>
            {s.retry}
          </Button>
        </div>
      );
    } else if (phase.kind === "choosing") {
      body = (
        <div data-rcl-app-notify-stack>
          <RadioGroup
            label={s.chooseTitle}
            description={fill(s.chooseDescription, { app })}
            variant="card"
            value={phase.choice}
            onValueChange={(choice) => {
              setPhase({ ...phase, choice });
            }}
            disabled={busy}
            options={[
              ...phase.state.devices.map((d) => ({
                value: d.group,
                label: d.label,
                description:
                  d.apps.length > 0 ? fill(s.alsoHere, { apps: d.apps.join(", ") }) : undefined,
                badge: d.bestMatch ? s.bestMatch : undefined,
                testId: `app-notify-device-${d.group}`,
              })),
              {
                value: STANDALONE,
                label: s.standaloneLabel,
                description: fill(s.standaloneDescription, { app }),
                testId: "app-notify-device-standalone",
              },
            ]}
          />
          <div data-rcl-app-notify-actions>
            <Button
              variant="primary"
              disabled={busy}
              onClick={() => void confirmChoice()}
              data-testid="app-notify-confirm"
            >
              {busy ? s.turningOn : s.confirm}
            </Button>
            <Button variant="ghost" disabled={busy} onClick={() => void load()}>
              {s.cancel}
            </Button>
          </div>
        </div>
      );
    } else {
      const { state } = phase;
      const sub = state.subscription;
      if (!state.eligible && !sub) {
        body = (
          <p data-rcl-app-notify-text>{fill(s.ineligible, { reason: state.ineligibleReason })}</p>
        );
      } else if (!sub) {
        body = (
          <div data-rcl-app-notify-stack>
            <p data-rcl-app-notify-text>{fill(s.offDescription, { app })}</p>
            <div data-rcl-app-notify-actions>
              <Button
                variant="primary"
                disabled={busy}
                onClick={() => void turnOn(state)}
                data-testid="app-notify-turn-on"
              >
                {busy ? s.turningOn : s.turnOn}
              </Button>
            </div>
          </div>
        );
      } else {
        const description = sub.standalone
          ? fill(s.onStandaloneDescription, { app })
          : fill(s.onDescription, { app, device: sub.deviceLabel });
        body = (
          <div data-rcl-app-notify-stack>
            <p data-rcl-app-notify-text data-testid="app-notify-status">
              {sub.healthy ? description : sub.unhealthyReason}
            </p>
            <div data-rcl-app-notify-actions>
              {!sub.healthy && (
                <Button variant="secondary" disabled={busy} onClick={() => void checkAgain()}>
                  {s.checkAgain}
                </Button>
              )}
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => void turnOff()}
                data-testid="app-notify-turn-off"
              >
                {busy ? s.turningOff : s.turnOff}
              </Button>
            </div>
          </div>
        );
      }
    }

    const urgencyOptions = [
      { value: "low", label: s.urgencyLow },
      { value: "normal", label: s.urgencyNormal },
      { value: "high", label: s.urgencyHigh },
      { value: "critical", label: s.urgencyCritical },
    ];

    return (
      <section ref={ref} data-rcl-app-notify className={className} data-testid="app-notifications">
        <h3 data-rcl-app-notify-title>{s.title}</h3>
        {body}
        {message && (
          <p data-rcl-app-notify-message role="alert">
            {message}
          </p>
        )}
        {showCategories && (
          <div data-rcl-app-notify-stack>
            <h3 data-rcl-app-notify-title>{s.categoriesTitle}</h3>
            <p data-rcl-app-notify-text>{fill(s.categoriesDescription, { app: app || "these" })}</p>
            {categoriesError && <p data-rcl-app-notify-text>{s.categoriesError}</p>}
            {categories?.map((row) => (
              <div key={row.category} data-rcl-app-notify-category>
                <Switch
                  checked={row.enabled}
                  onCheckedChange={(enabled) => void saveCategory({ ...row, enabled })}
                  label={row.displayName}
                  description={row.description}
                />
                <Select
                  aria-label={`${s.minimumUrgency}: ${row.displayName}`}
                  value={row.minUrgency}
                  disabled={!row.enabled}
                  options={urgencyOptions}
                  onValueChange={(minUrgency) => void saveCategory({ ...row, minUrgency })}
                />
              </div>
            ))}
          </div>
        )}
      </section>
    );
  },
);

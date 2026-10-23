import { Button } from "@vrooli/react-component-library/Button/2";
import { Avatar } from "@vrooli/react-component-library/Avatar/1";
import { Bell, BellOff, RefreshCw } from "lucide-react";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { usePreferenceStorageUnavailable, useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { recipientsClient, registerBrowserPushSubscription } from "../api/notifications";
import { forceServiceWorkerUpdate, readConnectionStatus, type ConnectionStatus } from "../api/serviceWorker";
import { fetchEventIntegrationConfig, updateEventIntegrationConfig, type EventIntegrationConfig } from "../api/integrations";
import { fetchNotificationCatalog, updateNotificationPreference, MIN_URGENCY_CHOICES, type CatalogCategory, type CatalogSource, type NotificationPreference } from "../api/preferences";
import { useEffect, useState } from "react";
import { getInjectedConfig } from "@vrooli/api-base";
import { SessionPanel, useSession } from "../features/session/SessionGate";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const preferenceStorageUnavailable = usePreferenceStorageUnavailable();
  const session = useSession();
  const queryClient = useQueryClient();
  const signedIn = Boolean(session.data?.signedIn);
  // Device metadata and the public VAPID key are recipient-scoped reads; only
  // subscription registration requires the signed-in owner session.
  const devices = useQuery({ queryKey: ["recipient-devices"], queryFn: () => recipientsClient.listDevices({}) });
  const eventConfig = useQuery({ queryKey: ["event-integration-config"], queryFn: fetchEventIntegrationConfig });
  const [eventForm, setEventForm] = useState<EventIntegrationConfig>({ events_api_base: "", webhook_url: "", pattern: "incident.*", sensitivity_by_severity: { critical: "critical", warning: "sensitive", informational: "public" } });
  const eventUpdate = useMutation({ mutationFn: updateEventIntegrationConfig, onSuccess: (config) => setEventForm(config) });
  const [pushMessage, setPushMessage] = useState("");

  // Catalog and recipient preference views are ordinary configured-recipient
  // reads. Preference changes remain behind the owner identity check.
  const catalog = useQuery({ queryKey: ["notification-catalog"], queryFn: fetchNotificationCatalog });
  const preferenceUpdate = useMutation({ mutationFn: updateNotificationPreference, onSuccess: () => void catalog.refetch() });
  const prefIndex = new Map<string, NotificationPreference>();
  for (const p of catalog.data?.preferences ?? []) prefIndex.set(`${p.source_scenario}\u0000${p.category}`, p);
  const effectivePref = (sourceScenario: string, category: string, defaultEnabled: boolean, defaultMinUrgency: string): NotificationPreference => {
    const stored = prefIndex.get(`${sourceScenario}\u0000${category}`);
    if (stored) return stored;
    return { source_scenario: sourceScenario, category, enabled: defaultEnabled, min_urgency: defaultMinUrgency || "low", channel_override: "" };
  };
  const savePref = (next: NotificationPreference) => preferenceUpdate.mutate(next);

  useEffect(() => {
    if (eventConfig.data) setEventForm(eventConfig.data);
  }, [eventConfig.data]);

  const enablePush = async () => {
    if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
      // iOS exposes Web Push only to Home Screen apps, never to a Safari tab.
      setPushMessage("This browser does not offer push here. On iPhone, tap Share → Add to Home Screen, open Notification Hub from the Home Screen icon, then try again.");
      return;
    }
    // Read the key at tap time: the cached device query may still be loading
    // or retrying, and an empty cache must not read as "not configured".
    let publicKey = devices.data?.vapidPublicKey;
    let keyError = "";
    if (!publicKey && signedIn) {
      try {
        publicKey = (await recipientsClient.listDevices({})).vapidPublicKey;
      } catch (error) {
        keyError = error instanceof Error ? error.message : String(error);
      }
    }
    publicKey = publicKey
      || getInjectedConfig()?.VAPID_PUBLIC_KEY as string | undefined
      || import.meta.env.VITE_VAPID_PUBLIC_KEY as string | undefined;
    if (!publicKey) {
      if (keyError) setPushMessage(`Could not load this hub's push key: ${keyError}`);
      else if (!signedIn) setPushMessage("Sign in before enabling notifications.");
      else setPushMessage("This hub has no push key configured.");
      return;
    }
    let registrationAccepted = false;
    try {
      await registerBrowserPushSubscription(base64UrlToBytes(publicKey) as unknown as BufferSource);
      registrationAccepted = true;
      await devices.refetch();
      // The panel may still have an older browser read in flight. Cancel it
      // before taking the post-registration snapshot so fetchQuery cannot
      // reuse that pre-registration promise or let it overwrite fresh state.
      await queryClient.cancelQueries({ queryKey: ["connection-status"], exact: true });
      const connection = await queryClient.fetchQuery<ConnectionStatus>({
        queryKey: ["connection-status"],
        queryFn: readConnectionStatus,
        staleTime: 0,
      });
      setPushMessage(connection.subscribed
        ? "This browser is registered for Web Push."
        : "The Hub accepted the registration, but this browser still reports no active push subscription. Check again or try enabling notifications once more.");
    } catch (error) {
      if (!registrationAccepted) {
        setPushMessage(error instanceof Error ? error.message : "Push registration failed.");
      } else {
        setPushMessage(error instanceof Error
          ? `Registration succeeded, but this browser's subscription status could not be verified: ${error.message}`
          : "Registration succeeded, but this browser's subscription status could not be verified.");
      }
    }
  };

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex flex-wrap gap-2">
          {THEME_CHOICES.map((c) => (
            <Button
              key={c}
              type="button"
              variant={choice === c ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={choice === c}
              onClick={() => setTheme(c)}
              data-testid={selectors.settingsPage.themeOption({ choice: c })}
            >
              {t(strings.theme.choice[c])}
            </Button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex flex-wrap gap-2">
          {SUPPORTED_LOCALES.map((lng) => (
            <Button
              key={lng}
              type="button"
              variant={currentLocale === lng ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={currentLocale === lng}
              onClick={() => void setLocale(lng)}
              data-testid={selectors.settingsPage.localeOption({ code: lng })}
            >
              {getLocaleConfig(lng).nativeLabel}
            </Button>
          ))}
        </div>
      </div>

      {preferenceStorageUnavailable && (
        <aside role="status" aria-live="polite" aria-atomic="true" className="border-s-4 border-app-warning ps-4 py-3">
          <p className="font-semibold">{t(strings.preferenceStorage.title)}</p>
          <p>{t(strings.preferenceStorage.body)}</p>
          <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.preferenceStorage.noAccountChange)}</p>
        </aside>
      )}

      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">Notification devices</h3>
        <p className="text-sm text-app-muted-foreground">Register this browser, inspect paired channels, and keep sensitive content governed by channel approval.</p>
        <SessionPanel />
        {signedIn && <Button type="button" variant="primary" onClick={() => void enablePush()}>Enable browser notifications</Button>}
        {pushMessage && <p role="status" className="text-sm text-app-muted-foreground">{pushMessage}</p>}
        <ConnectionStatusPanel />
        <ul className="space-y-2 text-sm" aria-label="Registered notification devices">
          {devices.data?.devices.map((device) => <li key={device.id} className="rounded border border-app-border px-3 py-2">{device.name} <span className="text-app-muted-foreground">{device.channels.join(", ") || "no channels"}</span></li>)}
        </ul>
      </div>

      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">Event integration</h3>
        <p className="text-sm text-app-muted-foreground">Update the live vrooli-events subscription without restarting notification-hub.</p>
        <label className="flex flex-col gap-1 text-sm">Events API base
          <input className="rounded border border-app-border bg-app-background px-3 py-2" value={eventForm.events_api_base} onChange={(event) => setEventForm({ ...eventForm, events_api_base: event.target.value })} />
        </label>
        <label className="flex flex-col gap-1 text-sm">Webhook URL
          <input className="rounded border border-app-border bg-app-background px-3 py-2" value={eventForm.webhook_url} onChange={(event) => setEventForm({ ...eventForm, webhook_url: event.target.value })} />
        </label>
        <label className="flex flex-col gap-1 text-sm">Event pattern
          <input className="rounded border border-app-border bg-app-background px-3 py-2" value={eventForm.pattern} onChange={(event) => setEventForm({ ...eventForm, pattern: event.target.value })} />
        </label>
        <label className="flex flex-col gap-1 text-sm">Critical sensitivity
          <input className="rounded border border-app-border bg-app-background px-3 py-2" value={eventForm.sensitivity_by_severity?.critical ?? ""} onChange={(event) => setEventForm({ ...eventForm, sensitivity_by_severity: { ...eventForm.sensitivity_by_severity, critical: event.target.value } })} />
        </label>
        <label className="flex flex-col gap-1 text-sm">Warning sensitivity
          <input className="rounded border border-app-border bg-app-background px-3 py-2" value={eventForm.sensitivity_by_severity?.warning ?? ""} onChange={(event) => setEventForm({ ...eventForm, sensitivity_by_severity: { ...eventForm.sensitivity_by_severity, warning: event.target.value } })} />
        </label>
        <label className="flex flex-col gap-1 text-sm">Informational sensitivity
          <input className="rounded border border-app-border bg-app-background px-3 py-2" value={eventForm.sensitivity_by_severity?.informational ?? ""} onChange={(event) => setEventForm({ ...eventForm, sensitivity_by_severity: { ...eventForm.sensitivity_by_severity, informational: event.target.value } })} />
        </label>
        <Button type="button" variant="secondary" disabled={eventUpdate.isPending} onClick={() => eventUpdate.mutate(eventForm)}>Save event integration</Button>
        {eventConfig.isError && <p aria-live="polite" className="text-sm text-app-muted-foreground">Event integration settings are unavailable.</p>}
        {eventUpdate.isError && <p aria-live="polite" className="text-sm text-app-muted-foreground">Event integration update failed.</p>}
      </div>

      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">Notification preferences</h3>
        <p className="text-sm text-app-muted-foreground">Choose which scenarios may notify you and the minimum urgency per category.</p>
        {!signedIn && <p className="text-sm text-app-muted-foreground">Sign in to change notification preferences.</p>}

        {catalog.isError ? (
          <NotificationsMessage
            testId="notifications-error"
            icon={<BellOff className="h-6 w-6" aria-hidden="true" />}
            title="Can't reach notifications right now"
            body="Your preferences are safe. This is usually a brief hiccup reaching the hub — try again in a moment."
          >
            <SecondaryButton
              testId="notifications-retry"
              onClick={() => void catalog.refetch()}
              icon={<RefreshCw className={`h-4 w-4 ${catalog.isFetching ? "animate-spin" : ""}`} aria-hidden="true" />}
            >
              Retry
            </SecondaryButton>
          </NotificationsMessage>
        ) : catalog.isLoading ? (
          <NotificationsSkeleton />
        ) : (catalog.data?.sources.length ?? 0) === 0 ? (
          <NotificationsMessage
            testId="notifications-empty"
            icon={<Bell className="h-6 w-6" aria-hidden="true" />}
            title="No notifications to manage yet"
            body="Once an app you use starts sending you notifications, it'll appear here so you can fine-tune what reaches you."
          >
            <SecondaryButton
              testId="notifications-refresh"
              onClick={() => void catalog.refetch()}
              icon={<RefreshCw className={`h-4 w-4 ${catalog.isFetching ? "animate-spin" : ""}`} aria-hidden="true" />}
            >
              Refresh
            </SecondaryButton>
          </NotificationsMessage>
        ) : (
          catalog.data?.sources.map((source: CatalogSource) => {
            const sourceName = source.display_name || source.source_scenario;
            return (
              <div key={source.source_scenario} data-testid={`notification-source-${source.source_scenario}`} className="flex flex-col gap-2 rounded-lg border border-app-border bg-app-surface-raised px-3 py-3">
                <div className="flex items-center gap-2.5">
                  <Avatar name={sourceName} size="xs" shape="rounded" />
                  <h4 className="text-sm font-semibold text-app-foreground">{sourceName}</h4>
                </div>
                {categoriesFor(source).map((category) => (
                  <PreferenceRow
                    key={category.category}
                    label={category.display_name || category.category || `All ${sourceName} notifications`}
                    description={category.description || undefined}
                    pref={effectivePref(source.source_scenario, category.category, category.default_enabled, category.default_min_urgency)}
                    disabled={preferenceUpdate.isPending || !signedIn}
                    onChange={savePref}
                  />
                ))}
              </div>
            );
          })
        )}
        {signedIn && preferenceUpdate.isError && <p aria-live="polite" className="text-sm text-app-muted-foreground">Notification preference update failed.</p>}
      </div>
    </section>
  );
}

/**
 * Connection status: makes the push pipeline observable so a device that looks
 * "enabled" but isn't actually connected — or is running a stale service worker
 * iOS refused to swap — shows the gap instead of hiding it. "Update now" forces
 * a waiting worker to activate and reloads.
 */
function ConnectionStatusPanel() {
  const status = useQuery({
    queryKey: ["connection-status"],
    queryFn: readConnectionStatus,
    // Reopening the home-screen app refocuses the window; re-read so the panel
    // reflects a worker that activated (or didn't) since it was last seen.
    refetchOnWindowFocus: true,
  });
  const update = useMutation({
    mutationFn: forceServiceWorkerUpdate,
    onSuccess: () => void status.refetch(),
  });
  const data = status.data;

  const worker: StatusTone = !data ? "muted" : !data.controlled ? "bad" : data.stale ? "warn" : "good";
  const subscription: StatusTone = !data ? "muted" : data.subscribed ? "good" : "bad";
  const permissionTone: StatusTone = !data
    ? "muted"
    : data.permission === "granted"
      ? "good"
      : data.permission === "denied"
        ? "bad"
        : "warn";

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-app-border bg-app-surface-raised px-3 py-3" data-testid="connection-status">
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-sm font-semibold text-app-foreground">Connection status</h4>
        <SecondaryButton
          testId="connection-status-refresh"
          onClick={() => void status.refetch()}
          icon={<RefreshCw className={`h-4 w-4 ${status.isFetching ? "animate-spin" : ""}`} aria-hidden="true" />}
        >
          Check again
        </SecondaryButton>
      </div>

      {!data?.supported && data && (
        <p className="text-xs text-app-muted-foreground" data-testid="connection-unsupported">
          Push isn&apos;t available in this browser context. On iPhone, add Notification Hub to your Home Screen and open it from there.
        </p>
      )}

      <StatusRow tone={permissionTone} label="Notification permission" value={data ? data.permission : "checking…"} testId="connection-permission" />
      <StatusRow tone={subscription} label="This device" value={data ? (data.subscribed ? `subscribed (${data.endpointHost})` : "not subscribed") : "checking…"} testId="connection-subscription" />
      <StatusRow
        tone={worker}
        label="Service worker"
        value={
          !data
            ? "checking…"
            : !data.controlled
              ? "no active worker"
              : data.stale
                ? `stale — running ${data.runningVersion ?? "?"}, latest ${data.availableVersion ?? "?"}`
                : `up to date (${data.runningVersion ?? data.availableVersion ?? "?"})`
        }
        testId="connection-worker"
      />

      {data?.stale && (
        <div className="flex flex-col gap-1">
          <Button type="button" variant="primary" disabled={update.isPending} onClick={() => update.mutate()} data-testid="connection-update">
            {update.isPending ? "Updating…" : "Update now"}
          </Button>
          <p className="text-xs text-app-muted-foreground">
            A newer version is deployed. If this persists on iPhone, fully close the app (swipe it away) and reopen it.
          </p>
        </div>
      )}
    </div>
  );
}

type StatusTone = "good" | "warn" | "bad" | "muted";

function StatusRow({ tone, label, value, testId }: { tone: StatusTone; label: string; value: string; testId: string }) {
  const dot: Record<StatusTone, string> = {
    good: "bg-green-500",
    warn: "bg-amber-500",
    bad: "bg-red-500",
    muted: "bg-app-surface-muted",
  };
  return (
    <div className="flex items-center justify-between gap-3 text-sm" data-testid={testId}>
      <span className="flex items-center gap-2 text-app-muted-foreground">
        <span className={`h-2 w-2 shrink-0 rounded-full ${dot[tone]}`} aria-hidden="true" />
        {label}
      </span>
      <span className="min-w-0 truncate text-app-foreground" data-testid={`${testId}-value`}>{value}</span>
    </div>
  );
}

function PreferenceRow({ label, description, pref, disabled, onChange }: {
  label: string;
  description?: string;
  pref: NotificationPreference;
  disabled: boolean;
  onChange: (next: NotificationPreference) => void;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-t border-app-border pt-2.5 first:border-t-0 first:pt-0">
      <div className="flex min-w-0 flex-col">
        <span className="text-sm text-app-foreground">{label}</span>
        {description && <span className="text-xs text-app-muted-foreground">{description}</span>}
      </div>
      <div className="flex items-center gap-3">
        <label className="flex items-center gap-1 text-sm">
          <input
            type="checkbox"
            checked={pref.enabled}
            disabled={disabled}
            onChange={(event) => onChange({ ...pref, enabled: event.target.checked })}
          />
          Enabled
        </label>
        <label className="flex items-center gap-1 text-sm">Min urgency
          <select
            className="rounded border border-app-border bg-app-background px-2 py-1"
            value={pref.min_urgency || "low"}
            disabled={disabled}
            onChange={(event) => onChange({ ...pref, min_urgency: event.target.value })}
          >
            {MIN_URGENCY_CHOICES.map((choice) => <option key={choice} value={choice}>{choice}</option>)}
          </select>
        </label>
      </div>
    </div>
  );
}

/**
 * A source may declare an empty category list — it only offers a coarse,
 * source-wide on/off. We synthesise a single pseudo-category with the empty
 * string id so the row maps onto the source-wide preference the hub stores
 * (matching web-console's NotificationsSection).
 */
function categoriesFor(source: CatalogSource): CatalogCategory[] {
  if (source.categories.length > 0) return source.categories;
  return [
    {
      category: "",
      display_name: "All notifications",
      description: "",
      declared: source.declared,
      default_enabled: true,
      default_min_urgency: "low",
    },
  ];
}

/** A loading skeleton shaped like the populated list, so the section never flashes blank. */
function NotificationsSkeleton() {
  return (
    <div className="grid gap-4" data-testid="notifications-loading" role="status" aria-live="polite" aria-label="Loading notification preferences">
      {[0, 1].map((group) => (
        <div key={group} className="flex flex-col gap-3 rounded-lg border border-app-border bg-app-surface-raised px-3 py-3">
          <div className="flex items-center gap-2.5">
            <div className="h-6 w-6 shrink-0 animate-pulse rounded-md bg-app-surface-muted" />
            <div className="h-3 w-32 animate-pulse rounded bg-app-surface-muted" />
          </div>
          {[0, 1].map((row) => (
            <div key={row} className="flex items-center justify-between gap-4">
              <div className="grid flex-1 gap-2">
                <div className="h-2.5 w-40 animate-pulse rounded bg-app-surface-muted" />
                <div className="h-2 w-3/4 animate-pulse rounded bg-app-surface-muted" />
              </div>
              <div className="h-5 w-16 shrink-0 animate-pulse rounded-full bg-app-surface-muted" />
            </div>
          ))}
        </div>
      ))}
      <span className="sr-only">Loading notification preferences</span>
    </div>
  );
}

/** A composed, centred message used for both the empty and error states. */
function NotificationsMessage({ icon, title, body, children, testId }: {
  icon: React.ReactNode;
  title: string;
  body: string;
  children?: React.ReactNode;
  testId: string;
}) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-lg border border-app-border bg-app-surface-raised px-6 py-10 text-center" role="status" data-testid={testId}>
      <span className="grid h-12 w-12 place-items-center rounded-2xl bg-app-primary/10 text-app-primary">{icon}</span>
      <div className="grid gap-1">
        <p className="font-semibold text-app-foreground">{title}</p>
        <p className="mx-auto max-w-sm text-sm text-app-muted-foreground">{body}</p>
      </div>
      {children && <div className="mt-1 flex flex-col items-center gap-2">{children}</div>}
    </div>
  );
}

function SecondaryButton({ children, onClick, testId, icon }: {
  children: React.ReactNode;
  onClick: () => void;
  testId?: string;
  icon?: React.ReactNode;
}) {
  return (
    <button
      type="button"
      data-testid={testId}
      onClick={onClick}
      className="inline-flex items-center gap-2 rounded-lg border border-app-border bg-transparent px-3.5 py-1.5 text-sm font-medium text-app-foreground transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
    >
      {icon}
      {children}
    </button>
  );
}

function base64UrlToBytes(value: string): Uint8Array {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const decoded = atob(normalized);
  return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}

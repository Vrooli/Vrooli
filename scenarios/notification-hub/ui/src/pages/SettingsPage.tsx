import { Button } from "@vrooli/react-component-library/Button/2";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useMutation, useQuery } from "@tanstack/react-query";
import { recipientsClient, registerBrowserPushSubscription } from "../api/notifications";
import { fetchEventIntegrationConfig, updateEventIntegrationConfig, type EventIntegrationConfig } from "../api/integrations";
import { useEffect, useState } from "react";
import { getInjectedConfig } from "@vrooli/api-base";

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
  const devices = useQuery({ queryKey: ["recipient-devices"], queryFn: () => recipientsClient.listDevices({}) });
  const eventConfig = useQuery({ queryKey: ["event-integration-config"], queryFn: fetchEventIntegrationConfig });
  const [eventForm, setEventForm] = useState<EventIntegrationConfig>({ events_api_base: "", webhook_url: "", pattern: "incident.*", sensitivity_by_severity: { critical: "critical", warning: "sensitive", informational: "public" } });
  const eventUpdate = useMutation({ mutationFn: updateEventIntegrationConfig, onSuccess: (config) => setEventForm(config) });
  const [pushMessage, setPushMessage] = useState("");

  useEffect(() => {
    if (eventConfig.data) setEventForm(eventConfig.data);
  }, [eventConfig.data]);

	  const enablePush = async () => {
    const publicKey = devices.data?.vapidPublicKey
      || getInjectedConfig()?.VAPID_PUBLIC_KEY as string | undefined
      || import.meta.env.VITE_VAPID_PUBLIC_KEY as string | undefined;
    if (!publicKey || !("serviceWorker" in navigator) || !("PushManager" in window)) {
      setPushMessage("Push is not configured for this origin yet.");
      return;
    }
    try {
      await registerBrowserPushSubscription(base64UrlToBytes(publicKey) as unknown as BufferSource);
      await devices.refetch();
      setPushMessage("This browser is registered for Web Push.");
    } catch (error) {
      setPushMessage(error instanceof Error ? error.message : "Push registration failed.");
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

      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">Notification devices</h3>
        <p className="text-sm text-app-muted-foreground">Register this browser, inspect paired channels, and keep sensitive content governed by channel approval.</p>
        <Button type="button" variant="primary" onClick={() => void enablePush()}>Enable browser notifications</Button>
        {pushMessage && <p role="status" className="text-sm text-app-muted-foreground">{pushMessage}</p>}
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
    </section>
  );
}

function base64UrlToBytes(value: string): Uint8Array {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const decoded = atob(normalized);
  return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}

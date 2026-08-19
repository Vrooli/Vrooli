import { Button } from "../components/ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useQuery } from "@tanstack/react-query";
import { recipientsClient, registerBrowserPushSubscription } from "../api/notifications";
import { useState } from "react";

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
  const [pushMessage, setPushMessage] = useState("");

	  const enablePush = async () => {
	    const publicKey = devices.data?.vapidPublicKey || (import.meta.env.VITE_VAPID_PUBLIC_KEY as string | undefined);
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
    </section>
  );
}

function base64UrlToBytes(value: string): Uint8Array {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const decoded = atob(normalized);
  return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}

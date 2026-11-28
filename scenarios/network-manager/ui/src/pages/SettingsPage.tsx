import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { fetchPrivacySettings } from "../api/network";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useQuery } from "@tanstack/react-query";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const THEME_LABELS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} satisfies Record<ThemeChoice, string>;

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const { data } = useQuery({
    queryKey: ["network", "privacy"],
    queryFn: fetchPrivacySettings,
  });

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.settings.description)}</p>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex gap-2">
          {THEME_CHOICES.map((c) => (
            <button
              key={c}
              type="button"
              role="radio"
              aria-checked={choice === c}
              onClick={() => setTheme(c)}
              data-testid={selectors.settingsPage.themeOption({ choice: c })}
              className={
                choice === c
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {t(THEME_LABELS[c])}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex gap-2">
          {SUPPORTED_LOCALES.map((lng) => (
            <button
              key={lng}
              type="button"
              role="radio"
              aria-checked={currentLocale === lng}
              onClick={() => void setLocale(lng)}
              data-testid={selectors.settingsPage.localeOption({ code: lng })}
              className={
                currentLocale === lng
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {getLocaleConfig(lng).nativeLabel}
            </button>
          ))}
        </div>
      </div>

      <section data-testid={selectors.network.privacySummary} className="rounded-panel border border-app-border bg-app-surface p-4">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.retentionHeading)}
        </h3>
        <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-3">
          <div>
            <dt className="text-app-muted-foreground">{t(strings.pages.settings.queryLogDays)}</dt>
            <dd className="font-semibold">{data?.retention?.queryLogDays ?? 0}</dd>
          </div>
          <div>
            <dt className="text-app-muted-foreground">{t(strings.pages.settings.snapshotDays)}</dt>
            <dd className="font-semibold">{data?.retention?.snapshotDays ?? 0}</dd>
          </div>
          <div>
            <dt className="text-app-muted-foreground">{t(strings.pages.settings.experimentDays)}</dt>
            <dd className="font-semibold">{data?.retention?.experimentDays ?? 0}</dd>
          </div>
        </dl>

        <h3 className="mt-5 text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.visibilityHeading)}
        </h3>
        <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-3">
          <div>
            <dt className="text-app-muted-foreground">{t(strings.pages.settings.showQueryDomains)}</dt>
            <dd className="font-semibold">
              {data?.visibility?.showQueryDomains ? t(strings.network.enabled) : t(strings.network.disabled)}
            </dd>
          </div>
          <div>
            <dt className="text-app-muted-foreground">{t(strings.pages.settings.showDeviceHistory)}</dt>
            <dd className="font-semibold">
              {data?.visibility?.showDeviceHistory ? t(strings.network.enabled) : t(strings.network.disabled)}
            </dd>
          </div>
          <div>
            <dt className="text-app-muted-foreground">{t(strings.pages.settings.householdMode)}</dt>
            <dd className="font-semibold">
              {data?.visibility?.householdMode ? t(strings.network.enabled) : t(strings.network.disabled)}
            </dd>
          </div>
        </dl>
      </section>
    </section>
  );
}

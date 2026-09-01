import { Button } from "@vrooli/react-component-library/Button/2";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme } from "../theme/ThemeProvider";

const THEME_CHOICES = [
  { choice: "light" as const, label: strings.theme.choice.light },
  { choice: "dark" as const, label: strings.theme.choice.dark },
  { choice: "system" as const, label: strings.theme.choice.system },
];

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>

      <div data-testid="settings-blast-radius" className="rounded-lg border p-4 text-sm text-app-muted-foreground">
        {t(strings.pages.settings.blastRadius)}
      </div>

      <section
        data-testid="settings-descriptor-region"
        data-experience-surface="descriptor-region"
        data-experience-state="ready"
        className="rounded-lg border p-4"
      >
        <h3 className="font-semibold">{t(strings.pages.settings.descriptorHeading)}</h3>
        <p data-testid="settings-descriptor-source" className="mt-2 text-sm text-app-muted-foreground">
          {t(strings.console.channels.descriptorLoaded)}
        </p>
        <p data-testid="settings-descriptor-status" role="status" className="mt-2 text-sm">
          {t(strings.pages.settings.descriptorStatus)}
        </p>
      </section>

      <section
        data-testid="settings-metering-region"
        data-experience-surface="metering-region"
        data-experience-state="ready"
        className="flex flex-col gap-3 rounded-lg border p-4"
      >
        <h3 data-testid="settings-metering" className="font-semibold">{t(strings.pages.settings.meteringHeading)}</h3>
        <label className="flex flex-col gap-1 text-sm" htmlFor="settings-byok">
          {t(strings.pages.settings.byokLabel)}
          <input id="settings-byok" data-testid="settings-byok" type="password" className="min-h-11 rounded border p-2" />
        </label>
        <p data-testid="settings-byok-note" className="text-sm text-app-muted-foreground">
          {t(strings.pages.settings.byokNote)}
        </p>
      </section>

      <div data-testid="settings-quiet-hours" role="group" className="rounded-lg border p-4">
        <h3 className="font-semibold">{t(strings.pages.settings.quietHoursHeading)}</h3>
        <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.settings.quietHoursDescription)}</p>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div data-testid="settings-theme" role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex flex-wrap gap-2">
          {THEME_CHOICES.map(({ choice: themeChoice, label }) => (
            <Button
              key={themeChoice}
              type="button"
              variant={choice === themeChoice ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={choice === themeChoice}
              onClick={() => setTheme(themeChoice)}
              data-testid={selectors.settingsPage.themeOption({ choice: themeChoice })}
            >
              {t(label)}
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

      <Button data-testid="settings-save" type="button" variant="primary" size="sm" className="self-start">
        {t(strings.pages.settings.save)}
      </Button>
    </section>
  );
}

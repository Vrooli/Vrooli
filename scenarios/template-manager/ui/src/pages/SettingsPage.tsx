import { Button } from "../components/ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { type ThemeChoice } from "../theme/ThemeProvider";
import { useTheme } from "../theme/useTheme";

type ThemeLabelKey = typeof strings.theme.choice[keyof typeof strings.theme.choice];

const THEME_CHOICES = [
  { choice: "light", labelKey: strings.theme.choice.light },
  { choice: "dark", labelKey: strings.theme.choice.dark },
  { choice: "system", labelKey: strings.theme.choice.system },
] as const satisfies ReadonlyArray<{ choice: ThemeChoice; labelKey: ThemeLabelKey }>;

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

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex flex-wrap gap-2">
          {THEME_CHOICES.map(({ choice: optionChoice, labelKey }) => (
            <Button
              key={optionChoice}
              type="button"
              variant={choice === optionChoice ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={choice === optionChoice}
              onClick={() => setTheme(optionChoice)}
              data-testid={selectors.settingsPage.themeOption({ choice: optionChoice })}
            >
              {t(labelKey)}
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
    </section>
  );
}

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Button } from "../components/ui/button";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/themeContext";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

function themeChoiceLabel(choice: ThemeChoice) {
  switch (choice) {
    case "light":
      return strings.theme.choice.light;
    case "dark":
      return strings.theme.choice.dark;
    case "system":
      return strings.theme.choice.system;
  }
}

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
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex gap-2">
          {THEME_CHOICES.map((c) => (
            <Button
              key={c}
              type="button"
              role="radio"
              aria-checked={choice === c}
              variant={choice === c ? "default" : "outline"}
              size="sm"
              onClick={() => setTheme(c)}
              data-testid={selectors.settingsPage.themeOption({ choice: c })}
            >
              {t(themeChoiceLabel(c))}
            </Button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex gap-2">
          {SUPPORTED_LOCALES.map((lng) => (
            <Button
              key={lng}
              type="button"
              role="radio"
              aria-checked={currentLocale === lng}
              variant={currentLocale === lng ? "default" : "outline"}
              size="sm"
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

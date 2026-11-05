import { useNavigate } from "react-router-dom";
import { Button } from "@vrooli/react-component-library/Button/2";
import { AuthSection } from "@vrooli/react-component-library/AuthSection/1.0.5";
import { PermissionState } from "@vrooli/react-component-library/PermissionState/1.0.8";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const THEME_LABELS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} as const satisfies Record<ThemeChoice, string>;

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
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
              onClick={() => setTheme(c)}
              size="sm"
              shape="square"
              variant={choice === c ? "primary" : "secondary"}
              data-testid={selectors.settingsPage.themeOption({ choice: c })}
            >
                {t(THEME_LABELS[c])}
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
              onClick={() => void setLocale(lng)}
              size="sm"
              shape="square"
              variant={currentLocale === lng ? "primary" : "secondary"}
              data-testid={selectors.settingsPage.localeOption({ code: lng })}
            >
              {getLocaleConfig(lng).nativeLabel}
            </Button>
          ))}
        </div>
      </div>

      <section aria-labelledby="security-heading" className="flex flex-col gap-4 rounded-panel border border-app-border bg-app-surface p-4">
        <div>
          <h3 id="security-heading" className="text-lg font-semibold">{t(strings.security.heading)}</h3>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.security.description)}</p>
        </div>
        <AuthSection
          signedIn={false}
          onSignIn={() => navigate("/auth/login")}
          onSignOut={() => navigate("/auth/login")}
        />
        <div aria-label={t("feedback.permission-state.permission-required")}>
          <p className="sr-only">{t("feedback.permission-state.request-access-to-continue")}</p>
          <PermissionState action={() => navigate("/auth/login")} />
          <span className="sr-only">{t("feedback.permission-state.request-access")}</span>
          <h4 className="sr-only">{t(strings.security.permissionHeading)}</h4>
          <p className="sr-only">{t(strings.security.permissionDescription)}</p>
        </div>
      </section>
    </section>
  );
}

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useUiPreferences } from "../hooks/useUiPreferences";

const THEME_OPTIONS = [
  { value: "light", labelKey: strings.theme.choice.light },
  { value: "dark", labelKey: strings.theme.choice.dark },
  { value: "system", labelKey: strings.theme.choice.system },
] as const satisfies ReadonlyArray<{ value: ThemeChoice; labelKey: string }>;

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const { preferences, updatePreference } = useUiPreferences();

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
          {THEME_OPTIONS.map(({ value, labelKey }) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={choice === value}
              onClick={() => setTheme(value)}
              data-testid={selectors.settingsPage.themeOption({ choice: value })}
              className={
                choice === value
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {t(labelKey)}
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

      <div data-testid={selectors.features.settingsExt.root} className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.preferencesHeading)}
        </h3>

        <fieldset className="flex flex-col gap-1">
          <legend className="text-sm font-medium">{t(strings.pages.settings.densityLabel)}</legend>
          <div role="radiogroup" aria-label={t(strings.pages.settings.densityLabel)} className="flex gap-2">
            {(["comfortable", "dense"] as const).map((d) => (
              <button
                key={d}
                type="button"
                role="radio"
                aria-checked={preferences.density === d}
                data-testid={`${selectors.features.settingsExt.densityToggle}-${d}`}
                onClick={() => updatePreference("density", d)}
                className={
                  preferences.density === d
                    ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                    : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
                }
              >
                {t(d === "comfortable" ? strings.pages.settings.densityComfortable : strings.pages.settings.densityDense)}
              </button>
            ))}
          </div>
        </fieldset>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={preferences.reducedMotion}
            data-testid={selectors.features.settingsExt.reducedMotionToggle}
            onChange={(e) => updatePreference("reducedMotion", e.target.checked)}
          />
          <span className="font-medium">{t(strings.pages.settings.reducedMotionLabel)}</span>
        </label>
        <span className="text-xs text-app-muted-foreground">
          {t(strings.pages.settings.reducedMotionHint)}
        </span>

        <fieldset className="flex flex-col gap-1">
          <legend className="text-sm font-medium">{t(strings.pages.settings.handednessLabel)}</legend>
          <div role="radiogroup" aria-label={t(strings.pages.settings.handednessLabel)} className="flex gap-2">
            {(["left", "right"] as const).map((h) => (
              <button
                key={h}
                type="button"
                role="radio"
                aria-checked={preferences.handedness === h}
                data-testid={`${selectors.features.settingsExt.handednessToggle}-${h}`}
                onClick={() => updatePreference("handedness", h)}
                className={
                  preferences.handedness === h
                    ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                    : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
                }
              >
                {t(h === "left" ? strings.pages.settings.handednessLeft : strings.pages.settings.handednessRight)}
              </button>
            ))}
          </div>
        </fieldset>

        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.pages.settings.defaultScenarioLabel)}</span>
          <input
            type="text"
            value={preferences.defaultScenario}
            data-testid={selectors.features.settingsExt.defaultScenarioInput}
            onChange={(e) => updatePreference("defaultScenario", e.target.value)}
            className="rounded-control border border-app-border bg-app-surface px-2 py-1"
          />
          <span className="text-xs text-app-muted-foreground">
            {t(strings.pages.settings.defaultScenarioHint)}
          </span>
        </label>

        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.pages.settings.defaultDomainLabel)}</span>
          <input
            type="text"
            value={preferences.defaultDomainFilter}
            data-testid={selectors.features.settingsExt.defaultDomainInput}
            onChange={(e) => updatePreference("defaultDomainFilter", e.target.value)}
            className="rounded-control border border-app-border bg-app-surface px-2 py-1"
          />
          <span className="text-xs text-app-muted-foreground">
            {t(strings.pages.settings.defaultDomainHint)}
          </span>
        </label>
      </div>
    </section>
  );
}

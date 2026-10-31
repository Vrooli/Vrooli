import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const THEME_LABEL_KEYS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} as const satisfies Record<ThemeChoice, string>;

export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-2">
        <p className="text-xs font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.eyebrow)}
        </p>
        <h2 id="settings-heading" className="text-2xl font-semibold">
          {t(strings.pages.settings.title)}
        </h2>
        <p className="max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.settings.description)}
        </p>
      </header>

      <div className="grid gap-5 lg:grid-cols-[1fr_1fr]">
        <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="policy-heading">
          <h3 id="policy-heading" className="font-semibold">{t(strings.pages.settings.policyHeading)}</h3>
          <div className="mt-4 grid gap-3">
            <div className="flex items-start justify-between gap-4 rounded-control border border-app-border bg-app-surface-muted p-3">
              <span>
                <label htmlFor="privacy-default" className="block text-sm font-medium">{t(strings.pages.settings.privacyDefault)}</label>
                <span className="mt-1 block text-sm text-app-muted-foreground">{t(strings.pages.settings.privacyDefaultDetail)}</span>
              </span>
              <select id="privacy-default" className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm" defaultValue="internal">
                <option value="internal">{t(strings.pages.settings.privacyInternal)}</option>
                <option value="confidential">{t(strings.pages.settings.privacyConfidential)}</option>
                <option value="secret">{t(strings.pages.settings.privacySecret)}</option>
              </select>
            </div>
            <div className="flex items-start justify-between gap-4 rounded-control border border-app-border bg-app-surface-muted p-3">
              <span>
                <label htmlFor="evidence-retention" className="block text-sm font-medium">{t(strings.pages.settings.evidenceRetention)}</label>
                <span className="mt-1 block text-sm text-app-muted-foreground">{t(strings.pages.settings.evidenceRetentionDetail)}</span>
              </span>
              <select id="evidence-retention" className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm" defaultValue="metadata">
                <option value="metadata">{t(strings.pages.settings.metadataOnly)}</option>
                <option value="short">{t(strings.pages.settings.shortRetention)}</option>
              </select>
            </div>
            <div className="flex items-start justify-between gap-4 rounded-control border border-app-border bg-app-surface-muted p-3">
              <span>
                <label htmlFor="smoke-before-execute" className="block text-sm font-medium">{t(strings.pages.settings.smokeBeforeExecute)}</label>
                <span className="mt-1 block text-sm text-app-muted-foreground">{t(strings.pages.settings.smokeBeforeExecuteDetail)}</span>
              </span>
              <input id="smoke-before-execute" type="checkbox" defaultChecked className="mt-1 size-5" />
            </div>
          </div>
        </section>

        <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="display-heading">
          <h3 id="display-heading" className="font-semibold">{t(strings.pages.settings.displayHeading)}</h3>
          <div className="mt-4 grid gap-4">
            <div className="flex flex-col gap-2">
              <h4 className="text-sm font-semibold uppercase text-app-muted-foreground">
                {t(strings.pages.settings.themeHeading)}
              </h4>
              <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex flex-wrap gap-2">
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
                        ? "rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground"
                        : "rounded-control border border-app-border px-3 py-2 text-sm text-app-foreground hover:bg-app-surface-muted"
                    }
                  >
                    {t(THEME_LABEL_KEYS[c])}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-2">
              <h4 className="text-sm font-semibold uppercase text-app-muted-foreground">
                {t(strings.pages.settings.localeHeading)}
              </h4>
              <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex flex-wrap gap-2">
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
                        ? "rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground"
                        : "rounded-control border border-app-border px-3 py-2 text-sm text-app-foreground hover:bg-app-surface-muted"
                    }
                  >
                    {getLocaleConfig(lng).nativeLabel}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </section>
      </div>
    </section>
  );
}

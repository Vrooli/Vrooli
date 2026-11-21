import { Check } from "lucide-react";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../../i18n";
import { usePreferenceStorageUnavailable } from "../../theme/preferenceStorage";
import { THEME_CHOICES, THEME_LABEL } from "../../theme/themeChoices";
import { useTheme } from "../../theme/themeContext";
import { Notice } from "./Notice";
import { SettingsScreen } from "./SettingsScreen";

// The same inset track and raised selected segment as the section tabs, as a radio group.
const CHOICE =
  "flex min-h-11 cursor-pointer items-center justify-center rounded-control px-3 text-sm font-medium text-app-muted-foreground transition-colors duration-quick hover:text-app-foreground has-[:checked]:bg-app-surface has-[:checked]:text-app-foreground has-[:checked]:shadow-raised has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2";

/** Theme and language. Both belong to this browser, so no sign-in is needed. */
export function AppearancePage() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
  const storageUnavailable = usePreferenceStorageUnavailable();
  const locale = getCurrentLocale();
  return (
    <SettingsScreen title={t(strings.settings.section.appearance)} description={t(strings.settings.appearance.description)} backToSettings>
      <fieldset className="flex flex-col gap-2">
        <legend className="mb-2 text-base font-semibold">{t(strings.settings.appearance.theme)}</legend>
        <div className="grid grid-cols-3 gap-1 rounded-control border border-app-border bg-app-surface-muted p-1">
          {THEME_CHOICES.map((theme) => (
            <label key={theme} className={CHOICE} data-testid={selectors.settingsPage.themeOption({ choice: theme })}>
              <input
                type="radio"
                name="settings-theme"
                className="sr-only"
                value={theme}
                checked={choice === theme}
                onChange={() => setTheme(theme)}
              />
              {t(THEME_LABEL[theme])}
            </label>
          ))}
        </div>
      </fieldset>
      <fieldset className="flex flex-col gap-2">
        <legend className="mb-2 text-base font-semibold">{t(strings.settings.appearance.language)}</legend>
        <div className="divide-y divide-app-border overflow-hidden rounded-panel border border-app-border bg-app-surface">
          {SUPPORTED_LOCALES.map((code) => (
            <label key={code} data-testid={selectors.settingsPage.localeOption({ code })} className="flex min-h-12 cursor-pointer items-center justify-between gap-3 px-4 py-2 has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:-outline-offset-2">
              <input
                type="radio"
                name="settings-language"
                className="sr-only"
                value={code}
                checked={locale === code}
                onChange={() => void setLocale(code)}
              />
              <span>{getLocaleConfig(code).nativeLabel}</span>
              {locale === code && <Check size={18} className="text-app-primary" aria-hidden="true" />}
            </label>
          ))}
        </div>
      </fieldset>
      <p className="text-sm text-app-muted-foreground">{t(strings.settings.appearance.browserOnly)}</p>
      {storageUnavailable && (
        <Notice tone="warning" title={t(strings.preferenceStorage.title)}>{t(strings.preferenceStorage.body)}</Notice>
      )}
    </SettingsScreen>
  );
}

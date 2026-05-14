/**
 * Locale switcher — button group binding the i18n pipeline (setLocale →
 * applyDocumentLocale → html lang/dir) to a user-visible control.
 *
 * Buttons render their *native* labels (English / 日本語 / العربية), not
 * translated copy, so a user who has accidentally landed in a locale they
 * can't read can still find their way back.
 */
import { useTranslation } from "react-i18next";
import { cn } from "../lib/classnames";
import {
  LOCALE_CODES,
  type Locale,
  getCurrentLocale,
  getLocaleConfig,
  setLocale,
} from "../i18n";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

// The selectors registry uses widened index-signature types that confuse
// `noUncheckedIndexedAccess`. Aliasing once narrows for every consumer
// below; same workaround as src/__tests__/routing.test.ts.
const localeSelectors = selectors.locale as unknown as {
  switcher: string;
  toggle: (params: { code: string }) => string;
};

export default function LocaleSwitcher() {
  const { t, i18n } = useTranslation();
  // Subscribe to i18n.language so aria-pressed re-renders when locale flips.
  void i18n.language;
  const active = getCurrentLocale();

  return (
    <div
      role="group"
      data-testid={localeSelectors.switcher}
      aria-label={t(strings.settings.workspaceSection.localeSwitcherAria)}
      className="inline-flex items-center gap-1 rounded-lg border border-wc-default bg-wc-surface-base p-0.5"
    >
      {LOCALE_CODES.map((code) => {
        const config = getLocaleConfig(code);
        const isActive = code === active;
        return (
          <button
            key={code}
            type="button"
            data-testid={localeSelectors.toggle({ code })}
            aria-pressed={isActive}
            aria-label={t(strings.settings.workspaceSection.localeToggleAria, {
              language: config.nativeLabel,
            })}
            onClick={() => {
              void setLocale(code as Locale);
            }}
            className={cn(
              "rounded-md px-2 py-1 text-xs font-medium transition-colors",
              isActive
                ? "bg-wc-accent text-black"
                : "text-wc-text-secondary hover:bg-wc-surface-input hover:text-wc-text-primary",
            )}
          >
            {config.nativeLabel}
          </button>
        );
      })}
    </div>
  );
}

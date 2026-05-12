import type { ReactNode } from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";

type Props = {
  children: ReactNode;
};

/**
 * PLACEHOLDER SHELL — REPLACE WHEN BUILDING THE REAL UX.
 *
 * This shell is a minimum-viable layout so the template boots green
 * and demonstrates the i18n/a11y/design-token seams. It is NOT a
 * reasonable end-state for your scenario. When you design the real
 * product UI:
 *   - Replace this centered single-panel layout with the real shell
 *     (navigation, routing, side panes, headers — whatever fits).
 *   - Replace the eyebrow / title / description with real product
 *     surfaces.
 *   - Replace the inline locale buttons with whatever fits the real
 *     settings/preferences surface (header menu, settings page,
 *     keyboard shortcut, etc.). KEEP the underlying i18n behavior:
 *     `useTranslation`, `setLocale`, `SUPPORTED_LOCALES` are durable
 *     seams, not placeholder choices.
 *   - Build a real settings page covering every preference the
 *     scenario actually needs (theme, font scale, locale, a11y,
 *     account, notifications, etc.) — do not constrain it to
 *     whatever specific examples appear in `DESIGN.md`; the design
 *     governs how settings look and behave, not which ones exist.
 *
 * Lives in `components/` (not `features/`) because the shell is
 * cross-cutting layout, not domain content.
 */
export function AppShell({ children }: Props) {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-app-background p-6 text-app-foreground">
      <div className="w-full max-w-xl rounded-panel border border-app-border bg-app-surface p-8 shadow-lg">
        <div className="flex items-center justify-between gap-4">
          <p
            data-testid={selectors.app.eyebrow}
            className="text-sm uppercase text-app-muted-foreground"
          >
            {t(strings.app.eyebrow)}
          </p>
          <div
            role="group"
            aria-label={t(strings.locale.switcherLabel)}
            data-testid={selectors.locale.switcher}
            className="flex items-center gap-1 rounded-control border border-app-border bg-app-surface-muted p-1 text-xs"
          >
            {SUPPORTED_LOCALES.map((lng) => (
              <button
                key={lng}
                type="button"
                data-testid={selectors.locale.toggle({ code: lng })}
                onClick={() => void setLocale(lng)}
                aria-pressed={currentLocale === lng}
                className={
                  currentLocale === lng
                    ? "rounded-control bg-app-primary px-3 py-1 font-medium text-app-primary-foreground"
                    : "rounded-control px-3 py-1 text-app-muted-foreground hover:text-app-foreground"
                }
              >
                {getLocaleConfig(lng).nativeLabel}
              </button>
            ))}
          </div>
        </div>
        <h1
          data-testid={selectors.app.title}
          className="mt-3 text-3xl font-semibold"
        >
          {t(strings.app.title)}
        </h1>
        <p
          data-testid={selectors.app.description}
          className="mt-2 text-app-muted-foreground"
        >
          {t(strings.app.description)}
        </p>
        {children}
      </div>
    </div>
  );
}

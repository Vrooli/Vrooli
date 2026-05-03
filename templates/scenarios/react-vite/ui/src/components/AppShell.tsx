import type { ReactNode } from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";

type Props = {
  children: ReactNode;
};

/**
 * AppShell renders the outer page layout — header, locale switcher,
 * and the card-stack scaffolding — and slots feature content into the
 * default <main> region via children.
 *
 * Lives in `components/` (not `features/`) because the shell is
 * cross-cutting layout, not domain content. Per react-coherence:
 * `features/<name>/` owns domain UI; `components/` owns reusable
 * layout primitives.
 */
export function AppShell({ children }: Props) {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-xl rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl backdrop-blur">
        <div className="flex items-center justify-between gap-4">
          <p
            data-testid={selectors.app.eyebrow}
            className="text-sm uppercase tracking-[0.2em] text-slate-400"
          >
            {t(strings.app.eyebrow)}
          </p>
          <div
            role="group"
            aria-label={t(strings.locale.switcherLabel)}
            data-testid={selectors.locale.switcher}
            className="flex items-center gap-1 rounded-full border border-white/10 bg-black/20 p-1 text-xs"
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
                    ? "rounded-full bg-white/15 px-3 py-1 font-medium text-white"
                    : "rounded-full px-3 py-1 text-slate-300 hover:text-white"
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
          className="mt-2 text-slate-300"
        >
          {t(strings.app.description)}
        </p>
        {children}
      </div>
    </div>
  );
}

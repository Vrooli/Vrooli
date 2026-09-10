import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../../i18n";
import { useTheme, type ThemeChoice } from "../../theme/ThemeProvider";
import { OwnerSignIn } from "./OwnerSignIn";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

function SignInUtility() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  return <div className="flex flex-wrap items-center gap-3">
    <div role="group" aria-label={t(strings.locale.switcherLabel)} data-testid={selectors.locale.switcher} className="flex gap-1">
      {SUPPORTED_LOCALES.map((code) => <button key={code} type="button" data-testid={selectors.locale.toggle({ code })} onClick={() => void setLocale(code)} aria-pressed={currentLocale === code}>{getLocaleConfig(code).nativeLabel}</button>)}
    </div>
    <label data-testid={selectors.theme.switcher}>
      <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
      <select value={choice} onChange={(event) => setTheme(event.target.value as ThemeChoice)} data-testid={selectors.theme.select} aria-label={t(strings.theme.switcherLabel)}>
        {THEME_CHOICES.map((theme) => <option key={theme} value={theme}>{t(strings.theme.choice[theme])}</option>)}
      </select>
    </label>
  </div>;
}

/**
 * Unauthenticated gate surface. The bridge console is owner-gated end-to-end
 * (every fleet RPC requires an owner JWT), so with no owner token we present the
 * sign-in / create-account form as the whole surface instead of a dashboard that
 * could only render "please sign in" errors. The library shell keeps theme and
 * locale reachable before sign-in.
 */
export function SignInScreen() {
  const { t } = useTranslation();
  return (
    <LibraryAppShell density="sidebar" mobileNav="tabs" mainMode="scroll"
      brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
      items={[]}
      utility={<SignInUtility />}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      testId={selectors.layout.shell}
    >
      <div data-testid={selectors.session.signInScreen} className="flex min-h-full items-center justify-center p-6">
        <div className="flex w-full max-w-md flex-col gap-4">
          <div className="flex flex-col gap-1">
            <h1 className="text-2xl font-semibold">{t(strings.session.screenTitle)}</h1>
            <p className="text-sm text-app-muted-foreground">{t(strings.session.screenIntro)}</p>
          </div>
          <p
            data-testid={selectors.session.firstTimeNote}
            className="rounded-panel border border-app-primary/30 bg-app-primary/10 p-3 text-sm text-app-foreground"
          >
            {t(strings.session.firstTimeNote)}
          </p>
          <OwnerSignIn />
        </div>
      </div>
    </LibraryAppShell>
  );
}

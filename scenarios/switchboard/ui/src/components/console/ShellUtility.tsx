import { LogIn, LogOut, Settings } from "lucide-react";
import { Link } from "react-router-dom";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useSession } from "../../features/session/SessionProvider";
import { useAttention } from "../../features/health/useAttention";
import { useTranslation } from "../../i18n";
import { useTheme, type ThemeChoice } from "../../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/** Scenario controls supplied to the library's responsive utility slot. */
export function ShellUtility() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
  const { session, requireSession, signOut } = useSession();
  const attention = useAttention();
  return (
    <div className="flex w-full min-w-0 items-center justify-between gap-1 md:flex-col">
      <button type="button" data-testid="shell-session" aria-label={t(session ? strings.console.session.signOut : strings.console.session.signIn)}
        title={session?.email ?? session?.subject ?? t(strings.console.session.signIn)}
        onClick={() => session ? signOut() : void requireSession()}
        className="grid min-h-11 min-w-11 place-items-center rounded-control border border-app-border text-app-foreground hover:bg-app-surface-muted">
        {session ? <LogOut aria-hidden="true" className="h-4 w-4" /> : <LogIn aria-hidden="true" className="h-4 w-4" />}
      </button>
      <label data-testid={selectors.theme.switcher} className="min-w-0 max-w-full">
        <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
        <select value={choice} onChange={(event) => setTheme(event.target.value as ThemeChoice)} data-testid={selectors.theme.select}
          aria-label={t(strings.theme.switcherLabel)} className="min-h-11 max-w-full rounded-control border border-app-border bg-app-surface px-1 text-xs text-app-foreground">
          {THEME_CHOICES.map((value) => <option key={value} value={value}>{t(strings.theme.choice[value])}</option>)}
        </select>
      </label>
      <Link to="/settings" data-testid="shell-settings" aria-label={t(strings.layout.nav.settings)} className="grid min-h-11 min-w-11 place-items-center rounded-control border border-app-border text-app-foreground md:hidden">
        <Settings aria-hidden="true" className="h-4 w-4" />
      </Link>
      <span role="status" title={t(attention.apiOk === false ? strings.console.shell.apiUnreachable : strings.console.shell.apiConnected)}>
        <span aria-hidden="true" className={["inline-block h-2 w-2 rounded-full", attention.apiOk === false ? "bg-app-danger" : attention.apiOk ? "bg-app-success" : "bg-app-border"].join(" ")} />
        <span className="sr-only">{t(attention.apiOk === false ? strings.console.shell.apiUnreachable : strings.console.shell.apiConnected)}</span>
      </span>
    </div>
  );
}

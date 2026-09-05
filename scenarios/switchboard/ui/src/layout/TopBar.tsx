import { BellRing, LogIn, LogOut, Settings } from "lucide-react";
import { Link, NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useAttention } from "./useAttention";
import { useSession } from "../features/session/SessionProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Compact top bar: brand, an attention pill when a decision is waiting, the
 * theme control, and (below `md`) the Settings entry the bottom nav omits.
 */
export function TopBar() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
  const attention = useAttention();
  const { session, requireSession, signOut } = useSession();

  return (
    <header
      data-testid={selectors.layout.topBar}
      className="flex min-h-14 shrink-0 items-center justify-between gap-3 border-b border-app-border bg-app-surface px-3 py-1.5 md:px-4"
    >
      <Link to="/" className="flex min-h-11 min-w-0 items-center gap-2.5 rounded-control text-app-foreground">
        <span aria-hidden="true" className="grid h-7 w-7 place-items-center rounded-control bg-app-shell text-white">
          <svg viewBox="0 0 20 20" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
            <path d="M3 6h9M3 10h14M3 14h6" />
            <circle cx="15" cy="6" r="1.6" fill="currentColor" stroke="none" />
            <circle cx="12" cy="14" r="1.6" fill="currentColor" stroke="none" />
          </svg>
        </span>
        <h1 data-testid={selectors.app.title} className="truncate text-base font-semibold tracking-tight">
          {t(strings.app.title)}
        </h1>
      </Link>
      <div className="flex items-center gap-2">
        {attention.pending > 0 ? (
          <Link
            to="/"
            data-testid="topbar-attention"
            className="inline-flex min-h-11 items-center gap-1.5 rounded-pill border border-app-warning/50 bg-app-warning/10 px-3 text-xs font-semibold text-app-warning md:min-h-9"
          >
            <BellRing aria-hidden="true" className="h-3.5 w-3.5" />
            {t(strings.console.attention.pendingCount, { count: attention.pending })}
          </Link>
        ) : null}
        {session ? (
          <button
            type="button"
            data-testid="topbar-session"
            onClick={signOut}
            title={t(strings.console.session.signOut)}
            className="hidden min-h-11 items-center gap-1.5 rounded-control border border-app-border px-2.5 text-xs text-app-foreground hover:bg-app-surface-muted sm:inline-flex md:min-h-9"
          >
            <span className="h-2 w-2 rounded-full bg-app-success" aria-hidden="true" />
            <span className="max-w-[12rem] truncate">{session.email ?? session.subject}</span>
            <LogOut aria-hidden="true" className="h-3.5 w-3.5 text-app-muted-foreground" />
          </button>
        ) : (
          <button
            type="button"
            data-testid="topbar-session"
            onClick={() => void requireSession()}
            className="inline-flex min-h-11 min-w-11 items-center justify-center gap-1.5 rounded-control border border-app-border px-2.5 text-xs font-medium text-app-foreground hover:bg-app-surface-muted md:min-h-9 md:min-w-0"
          >
            <LogIn aria-hidden="true" className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">{t(strings.console.session.signIn)}</span>
          </button>
        )}
        <label data-testid={selectors.theme.switcher} className="flex items-center text-xs text-app-muted-foreground">
          <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
          <select
            value={choice}
            onChange={(e) => setTheme(e.target.value as ThemeChoice)}
            data-testid={selectors.theme.select}
            aria-label={t(strings.theme.switcherLabel)}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-2 text-xs text-app-foreground md:min-h-9"
          >
            {THEME_CHOICES.map((c) => (
              <option key={c} value={c}>
                {t(strings.theme.choice[c])}
              </option>
            ))}
          </select>
        </label>
        <NavLink
          to="/settings"
          aria-label={t(strings.layout.nav.settings)}
          data-testid="topbar-settings"
          className={({ isActive }) =>
            [
              "grid h-11 w-11 place-items-center rounded-control border border-app-border md:hidden",
              isActive ? "bg-app-primary/10 text-app-primary" : "text-app-muted-foreground",
            ].join(" ")
          }
        >
          <Settings aria-hidden="true" className="h-4 w-4" />
        </NavLink>
      </div>
    </header>
  );
}

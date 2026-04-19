import { type ReactNode } from "react";
import { ThemeProvider, type ThemeKey } from "./ThemeProvider";

interface DashboardLayoutProps {
  themeKey: ThemeKey;
  title: string;
  children: ReactNode;
  aside?: ReactNode;
}

/**
 * Shared layout chrome used by every dashboard page.
 * - Wraps children in a <ThemeProvider> so CSS variables cascade.
 * - Sticks a header at the top with the dashboard title.
 * - Renders an optional <aside> column (typically a MetricList).
 */
export function DashboardLayout({
  themeKey,
  title,
  children,
  aside,
}: DashboardLayoutProps) {
  const bodyClass = aside ? "cc-body" : "cc-body cc-body-single";
  return (
    <ThemeProvider themeKey={themeKey}>
      <div data-theme={themeKey} className="cc-layout">
        <header className="cc-header">
          <h1>{title}</h1>
        </header>
        <div className={bodyClass}>
          <main>{children}</main>
          {aside ? (
            <aside className="cc-aside">
              <details className="cc-aside-drawer" open data-testid="metrics-drawer">
                <summary className="cc-aside-summary">Metrics</summary>
                <div className="cc-aside-body">{aside}</div>
              </details>
            </aside>
          ) : null}
        </div>
      </div>
    </ThemeProvider>
  );
}

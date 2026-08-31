/**
 * @libraryId react-component-library:AppShell
 * @displayName AppShell
 * @description
 * @version 1.0.6
 * @tags ["navigation","layout","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/**
 * @vrooliComponentSource react-component-library:AppShell
 * @deps {"react":"^18"}
 */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";

export interface AppShellProps {
  navigation?: ReactNode;
  header?: ReactNode;
  children?: ReactNode;
  title?: string;
  navigationLabel?: string;
  /**
   * `managed` lets a composed navigation asset own its drawer, resize handle,
   * and responsive presentation while this shell still owns the landmarks and
   * page geometry. The default `rail` presentation is intentionally opinionated
   * and needs no integration glue.
   */
  navigationMode?: "rail" | "managed";
  headerMode?: "visible" | "hidden";
  mainMode?: "padded" | "flush";
  className?: string;
  navigationClassName?: string;
  headerClassName?: string;
  mainClassName?: string;
}

const appShellStyles = `
[data-rcl-app-shell] { display: grid; grid-template-columns: minmax(15rem, 20rem) minmax(0, 1fr); min-block-size: 100dvh; min-inline-size: 0; background: var(--color-background); color: var(--color-foreground); }
[data-rcl-app-shell] [data-rcl-app-shell-nav] { min-block-size: 0; min-inline-size: 0; overflow: auto; border-inline-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); padding: var(--space-sm); }
[data-rcl-app-shell] [data-rcl-app-shell-content] { display: flex; min-block-size: 0; min-inline-size: 0; flex-direction: column; }
[data-rcl-app-shell] [data-rcl-app-shell-header] { display: flex; min-block-size: var(--space-2xl); min-inline-size: 0; flex-shrink: 0; align-items: center; border-block-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); padding-inline: var(--space-md); }
[data-rcl-app-shell] [data-rcl-app-shell-main] { min-block-size: 0; min-inline-size: 0; flex: 1; overflow: auto; padding: clamp(var(--space-sm), 3vw, var(--space-xl)); }
[data-rcl-app-shell] [data-rcl-app-shell-nav] nav { display: grid; gap: var(--space-3xs); }
[data-rcl-app-shell] [data-rcl-app-shell-nav] a { display: flex; min-block-size: var(--tap-target-min); align-items: center; border-radius: var(--radius-control); color: var(--color-muted-foreground); padding-inline: var(--space-xs); font: var(--text-body-sm); text-decoration: none; transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
[data-rcl-app-shell] [data-rcl-app-shell-nav] a:hover, [data-rcl-app-shell] [data-rcl-app-shell-nav] a[aria-current="page"] { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-app-shell] [data-rcl-app-shell-nav] a[aria-current="page"] { font-weight: 650; box-shadow: inset var(--space-3xs) 0 var(--color-primary); }

@media (max-width: 48rem) { [data-rcl-app-shell] { grid-template-columns: 1fr; grid-template-rows: auto minmax(0, 1fr); } [data-rcl-app-shell] [data-rcl-app-shell-nav] { border-block-end: var(--border-hairline) solid var(--color-border); border-inline-end: 0; padding: var(--space-xs); } [data-rcl-app-shell] [data-rcl-app-shell-nav] nav { display: flex; flex-wrap: wrap; max-inline-size: 100%; overflow: hidden; scrollbar-width: none; } [data-rcl-app-shell] [data-rcl-app-shell-nav] a { flex: 1 1 auto; min-inline-size: 0; max-inline-size: 100%; overflow-wrap: anywhere; white-space: normal; } [data-rcl-app-shell] [data-rcl-app-shell-header] { padding-inline: var(--space-sm); } [data-rcl-app-shell] [data-rcl-app-shell-main] { padding: var(--space-sm); } }
[data-rcl-app-shell][data-navigation-mode="managed"] { grid-template-columns: minmax(0, auto) minmax(0, 1fr); }
[data-rcl-app-shell][data-navigation-mode="managed"] [data-rcl-app-shell-nav] { block-size: 100dvh; max-block-size: 100dvh; min-block-size: 0; overflow: hidden; padding: 0; border: 0; background: transparent; }
[data-rcl-app-shell][data-navigation-mode="managed"] [data-rcl-app-shell-content] { min-inline-size: 0; }
[data-rcl-app-shell][data-header-mode="hidden"] [data-rcl-app-shell-header] { display: none; }
[data-rcl-app-shell][data-main-mode="flush"] [data-rcl-app-shell-main] { padding: 0; }
@media (max-width: 48rem) { [data-rcl-app-shell][data-navigation-mode="managed"] [data-rcl-app-shell-nav] { block-size: auto; max-block-size: none; min-block-size: 0; border: 0; padding: 0; overflow: visible; } }


`;

function DefaultNavigation() {
  const libraryStrings = useStrings();
  return (
    <nav
      aria-label={libraryStrings(
        "navigation.app-shell.application-navigation",
        "Application navigation",
      )}
    >
      <a data-testid="navigation.app-shell" href="/" aria-current="page">
        {libraryStrings("navigation.app-shell.workspace", "Workspace")}
      </a>
      <a data-testid="navigation.app-shell" href="/settings">
        {libraryStrings("navigation.app-shell.settings", "Settings")}
      </a>
    </nav>
  );
}

export const AppShell = withClassName(function AppShell({
  navigation,
  header,
  children,
  title,
  navigationLabel = "Application navigation",
  navigationMode = "rail",
  headerMode = "visible",
  mainMode = "padded",
  className,
  navigationClassName,
  headerClassName,
  mainClassName,
}: AppShellProps) {
  const libraryStrings = useStrings();
  title =
    title ?? libraryStrings("navigation.app-shell.application-workspace", "Application workspace");
  const strings = useStrings();
  return (
    <div
      data-rcl-app-shell
      data-navigation-mode={navigationMode}
      data-header-mode={headerMode}
      data-main-mode={mainMode}
      data-testid="app-shell"
      className={className}
    >
      {/* prettier-ignore */}
      <StyleSheet name="appshell-1-0-6-1" css={appShellStyles} />
      <a
        className="rcl-app-shell-skip"
        data-testid="navigation.app-shell-skip"
        href="#app-shell-main"
      >
        {strings("navigation.app-shell.skip-to-content", "Skip to content")}
      </a>
      <div
        data-rcl-app-shell-nav
        data-testid="app-shell-navigation"
        role="complementary"
        aria-label={navigationLabel}
        className={navigationClassName}
      >
        {navigation ?? <DefaultNavigation />}
      </div>
      <div data-rcl-app-shell-content>
        {header ? (
          <header
            data-rcl-app-shell-header
            data-testid="app-shell-header"
            className={headerClassName}
          >
            {header}
          </header>
        ) : (
          <header
            data-rcl-app-shell-header
            data-testid="app-shell-header"
            className={headerClassName}
          >
            <h1>{title}</h1>
          </header>
        )}
        <main
          id="app-shell-main"
          data-testid="app-shell-main"
          data-rcl-app-shell-main
          role="main"
          aria-label={title}
          className={mainClassName}
        >
          {children}
        </main>
      </div>
    </div>
  );
});

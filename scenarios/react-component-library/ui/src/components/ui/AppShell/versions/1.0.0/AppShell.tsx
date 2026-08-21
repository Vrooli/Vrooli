/**
 * @vrooliComponentSource navigation.app-shell
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 2c80bd09-6377-45af-9f5d-83bce616b8d9
 * @vrooliComponentAppliedAt 2026-08-17T23:22:40Z
 * @vrooliComponentSourceSha256 2cf737c46aa9ffc910951b082c485b238b3597a69f03ff1498df58e31cc95eef
 * @vrooliComponentDriftHash dbb90ad3457f949a57bd9c8bcc664dcf72aadae2d30fc1f8d967b2c3af378652
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import { useComponentStyles } from "../../../../../hooks/useComponentStyles";

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
[data-rcl-app-shell] .rcl-app-shell-skip { position: fixed; inset-block-start: var(--space-sm); inset-inline-start: var(--space-sm); z-index: var(--layer-toast); inline-size: var(--tap-target-min); block-size: var(--tap-target-min); overflow: hidden; clip-path: inset(50%); white-space: nowrap; border: 0; padding: 0; border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); font: var(--text-label); text-decoration: none; }
[data-rcl-app-shell] .rcl-app-shell-skip:focus-visible { inline-size: auto; block-size: auto; overflow: visible; clip-path: none; white-space: normal; padding: var(--space-xs) var(--space-sm); outline: var(--focus-ring-width) solid var(--color-focus-ring); outline-offset: var(--focus-ring-offset); }
@media (max-width: 48rem) { [data-rcl-app-shell] { grid-template-columns: 1fr; grid-template-rows: auto minmax(0, 1fr); } [data-rcl-app-shell] [data-rcl-app-shell-nav] { border-block-end: var(--border-hairline) solid var(--color-border); border-inline-end: 0; padding: var(--space-xs); } [data-rcl-app-shell] [data-rcl-app-shell-nav] nav { display: flex; flex-wrap: wrap; max-inline-size: 100%; overflow: hidden; scrollbar-width: none; } [data-rcl-app-shell] [data-rcl-app-shell-nav] a { flex: 1 1 auto; min-inline-size: 0; max-inline-size: 100%; overflow-wrap: anywhere; white-space: normal; } [data-rcl-app-shell] [data-rcl-app-shell-header] { padding-inline: var(--space-sm); } [data-rcl-app-shell] [data-rcl-app-shell-main] { padding: var(--space-sm); } }
[data-rcl-app-shell][data-navigation-mode="managed"] { grid-template-columns: minmax(0, auto) minmax(0, 1fr); }
[data-rcl-app-shell][data-navigation-mode="managed"] [data-rcl-app-shell-nav] { block-size: 100dvh; max-block-size: 100dvh; min-block-size: 0; overflow: hidden; padding: 0; border: 0; background: transparent; }
[data-rcl-app-shell][data-navigation-mode="managed"] [data-rcl-app-shell-content] { min-inline-size: 0; }
[data-rcl-app-shell][data-header-mode="hidden"] [data-rcl-app-shell-header] { display: none; }
[data-rcl-app-shell][data-main-mode="flush"] [data-rcl-app-shell-main] { padding: 0; }
@media (max-width: 48rem) { [data-rcl-app-shell][data-navigation-mode="managed"] [data-rcl-app-shell-nav] { block-size: auto; max-block-size: none; min-block-size: 0; border: 0; padding: 0; overflow: visible; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-app-shell] *, [data-rcl-app-shell] *::before, [data-rcl-app-shell] *::after { transition: none !important; } }
@media (forced-colors: active) { [data-rcl-app-shell] [data-rcl-app-shell-nav] a[aria-current="page"] { background: Highlight; color: HighlightText; } }
`;

function defaultNavigation() {
  return (
    <nav aria-label="Application navigation">
      <a href="/" aria-current="page">
        Workspace
      </a>
      <a href="/settings">Settings</a>
    </nav>
  );
}

export function AppShell({
  navigation,
  header,
  children,
  title = "Application workspace",
  navigationLabel = "Application navigation",
  navigationMode = "rail",
  headerMode = "visible",
  mainMode = "padded",
  className,
  navigationClassName,
  headerClassName,
  mainClassName,
}: AppShellProps) {
  useComponentStyles("rcl-app-shell", appShellStyles);

  return (
    <div
      data-rcl-app-shell
      data-navigation-mode={navigationMode}
      data-header-mode={headerMode}
      data-main-mode={mainMode}
      data-testid="app-shell"
      className={className}
    >
      <a className="rcl-app-shell-skip" data-testid="app-shell-skip" href="#app-shell-main">
        Skip to content
      </a>
      <div
        data-rcl-app-shell-nav
        data-testid="app-shell-navigation"
        role="complementary"
        aria-label={navigationLabel}
        className={navigationClassName}
      >
        {navigation ?? defaultNavigation()}
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
}

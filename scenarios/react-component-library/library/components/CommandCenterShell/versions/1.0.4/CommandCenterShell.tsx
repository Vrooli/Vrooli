/**
 * @libraryId react-component-library:CommandCenterShell
 * @displayName Command Center Shell
 * @version 1.0.4
 * @tags ["layout","command-center","dashboard","responsive","experience"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import type { ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
export const commandCenterShellStyles = `
[data-rcl-command-center] { min-block-size: 100%; display: grid; grid-template-columns: minmax(0, 1fr); gap: var(--space-md); padding: var(--space-md); background: var(--color-background); color: var(--color-foreground); }
[data-rcl-command-center] .rcl-command-center__navigation { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); padding: var(--space-sm); }
[data-rcl-command-center] .rcl-command-center__navigation > * { min-inline-size: 0; }
[data-rcl-command-center] .rcl-command-center__navigation a { display: flex; min-block-size: var(--tap-target-min); align-items: center; border-radius: var(--radius-control); color: var(--color-muted-foreground); padding-inline: var(--space-xs); text-decoration: none; font: 600 var(--text-body-sm-size)/var(--text-body-sm-line) var(--font-sans); transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
[data-rcl-command-center] .rcl-command-center__navigation a:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-command-center] .rcl-command-center__navigation a:focus-visible, [data-rcl-command-center] button:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
[data-rcl-command-center] .rcl-command-center__body { min-inline-size: 0; display: grid; align-content: start; gap: var(--space-md); }
[data-rcl-command-center] .rcl-command-center__header { display: flex; min-inline-size: 0; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-sm); }
[data-rcl-command-center] .rcl-command-center__title { min-inline-size: 0; margin: 0; color: var(--color-foreground); font-size: var(--text-title-size); font-weight: 700; line-height: var(--text-title-line); letter-spacing: -0.015em; }
[data-rcl-command-center] .rcl-command-center__controls { display: flex; min-inline-size: 0; flex-wrap: wrap; gap: var(--space-2xs); }
[data-rcl-command-center] .rcl-command-center__controls button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); cursor: pointer; padding-inline: var(--space-sm); font: 600 var(--text-body-sm-size)/var(--text-body-sm-line) var(--font-sans); transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-command-center] .rcl-command-center__controls button:hover { border-color: var(--color-primary); background: var(--color-surface-muted); }
[data-rcl-command-center] .rcl-command-center__controls button:active { transform: translateY(1px); }
[data-rcl-command-center] .rcl-command-center__metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 10rem), 1fr)); gap: var(--space-sm); margin: 0; }
[data-rcl-command-center] .rcl-command-center__metric { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); padding: var(--space-sm); }
[data-rcl-command-center] .rcl-command-center__metric-label { color: var(--color-muted-foreground); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-command-center] .rcl-command-center__metric-value { margin: var(--space-3xs) 0 0; color: var(--color-foreground); font-size: var(--text-title-size); font-weight: 700; line-height: var(--text-title-line); letter-spacing: -0.015em; }
[data-rcl-command-center] .rcl-command-center__metric-detail { margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-command-center] .rcl-command-center__primary { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); padding: var(--space-md); }
[data-rcl-command-center] .rcl-command-center__primary > :not([role="status"]) { max-inline-size: 100%; }
@media (min-width: 48rem) { [data-rcl-command-center] { grid-template-columns: minmax(12rem, 15rem) minmax(0, 1fr); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-command-center] *, [data-rcl-command-center] *::before, [data-rcl-command-center] *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; } }
@media (forced-colors: active) { [data-rcl-command-center] .rcl-command-center__navigation, [data-rcl-command-center] .rcl-command-center__metric, [data-rcl-command-center] .rcl-command-center__primary, [data-rcl-command-center] .rcl-command-center__controls button { border-color: CanvasText; background: Canvas; color: CanvasText; } }
`;
export interface CommandCenterMetric {
  label: string;
  value: string;
  detail?: string;
}

export interface CommandCenterShellProps {
  title: string;
  navigation: ReactNode;
  metrics: CommandCenterMetric[];
  controls?: ReactNode;
  children?: ReactNode;
  regionId?: string;
  regionState?: ExperienceSurfaceState;
  loading?: ReactNode;
  empty?: ReactNode;
  partial?: ReactNode;
  error?: ReactNode;
  onRetry?: () => void;
  className?: string;
}

// CommandCenterShell intentionally composes durable regions instead of drawing
// a decorative frame: callers provide real navigation, controls, and primary
// content while AsyncPanel supplies observable lifecycle semantics.
export const CommandCenterShell = withClassName(function CommandCenterShell({
  title,
  navigation,
  metrics,
  controls,
  children,
  regionId = "command-center-primary",
  regionState = "ready",
  loading,
  empty,
  partial,
  error,
  onRetry,
  className,
}: CommandCenterShellProps) {
  return (
    <main
      data-rcl-command-center
      className={["rcl-command-center", className].filter(Boolean).join(" ")}
    >
      <StyleSheet name="command-center-shell-1-0-1" css={commandCenterShellStyles} />
      <nav aria-label={`${title} navigation`} className="rcl-command-center__navigation">
        {navigation}
      </nav>
      <section className="rcl-command-center__body">
        <header className="rcl-command-center__header">
          <h1 className="rcl-command-center__title">{title}</h1>
          {controls ? (
            <div aria-label={`${title} controls`} className="rcl-command-center__controls">
              {controls}
            </div>
          ) : null}
        </header>
        <dl className="rcl-command-center__metrics">
          {metrics.map((metric) => (
            <div key={metric.label} className="rcl-command-center__metric">
              <dt className="rcl-command-center__metric-label">{metric.label}</dt>
              <dd className="rcl-command-center__metric-value">{metric.value}</dd>
              {metric.detail ? (
                <p className="rcl-command-center__metric-detail">{metric.detail}</p>
              ) : null}
            </div>
          ))}
        </dl>
        <AsyncPanel
          surfaceId={regionId}
          state={regionState}
          loading={loading}
          empty={empty}
          partial={partial}
          error={error}
          onRetry={onRetry}
          className="rcl-command-center__primary"
        >
          {children}
        </AsyncPanel>
      </section>
    </main>
  );
});

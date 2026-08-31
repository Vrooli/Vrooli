/**
 * @libraryId react-component-library:CommandCenterShell
 * @displayName Command Center Shell
 * @description A responsive operational-workspace composition with navigation, summary figures, controls, and a lifecycle-aware primary region.
 * @version 1.0.2
 * @tags ["layout","command-center","dashboard","responsive","experience"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import type { ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
import { commandCenterShellStyles } from "./styles";

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

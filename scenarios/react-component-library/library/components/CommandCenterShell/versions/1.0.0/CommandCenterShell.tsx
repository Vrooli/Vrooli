/**
 * @libraryId react-component-library:CommandCenterShell
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18","react-component-library:ClassMerge":"1.0.0"}
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "../../../AsyncPanel/versions/1.0.0/AsyncPanel";
import { cn } from "../../../../foundations/ClassMerge/versions/1.0.0/ClassMerge";
import type { ExperienceSurfaceState } from "../../../ExperienceSurface/versions/1.0.0/ExperienceSurface";
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
export function CommandCenterShell({
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
      className={cn("rcl-command-center", className)}
    >
      <style
        data-rcl-command-center-styles
        dangerouslySetInnerHTML={{ __html: commandCenterShellStyles }}
      />
      <nav
        aria-label={`${title} navigation`}
        className="rcl-command-center__navigation"
      >
        {navigation}
      </nav>
      <section className="rcl-command-center__body">
        <header className="rcl-command-center__header">
          <h1 className="rcl-command-center__title">{title}</h1>
          {controls ? (
            <div
              aria-label={`${title} controls`}
              className="rcl-command-center__controls"
            >
              {controls}
            </div>
          ) : null}
        </header>
        <dl className="rcl-command-center__metrics">
          {metrics.map((metric) => (
            <div key={metric.label} className="rcl-command-center__metric">
              <dt className="rcl-command-center__metric-label">
                {metric.label}
              </dt>
              <dd className="rcl-command-center__metric-value">
                {metric.value}
              </dd>
              {metric.detail ? (
                <p className="rcl-command-center__metric-detail">
                  {metric.detail}
                </p>
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
}

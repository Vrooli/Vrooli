/**
 * @vrooliComponentSource react-component-library:CommandCenterShell
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 7d593e9b-1216-4a69-b029-848dc20ab26f
 * @vrooliComponentAppliedAt 2026-08-10T20:01:09Z
 * @vrooliComponentSourceSha256 7e0c48c458f6789470d86f2f8f722b704cba4dffcaf3728c6b2bde0fd5b37705
 * @vrooliComponentDriftHash 45abc42ff375a79fe55bb74785369ccf30782089569241b936dd1cbf939cd28a
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "../../../AsyncPanel/versions/1.0.0/AsyncPanel";
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
      className={["rcl-command-center", className].filter(Boolean).join(" ")}
    >
      <style
        data-rcl-command-center-styles
        dangerouslySetInnerHTML={{ __html: commandCenterShellStyles }}
      />
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
}

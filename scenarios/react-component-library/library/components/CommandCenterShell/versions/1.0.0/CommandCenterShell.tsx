/**
 * @libraryId react-component-library:CommandCenterShell
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "../../../AsyncPanel/versions/1.0.0/AsyncPanel";
import type { ExperienceSurfaceState } from "../../../ExperienceSurface/versions/1.0.0/ExperienceSurface";

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
      className={[
        "grid min-h-full gap-4 p-4 lg:grid-cols-[15rem_minmax(0,1fr)]",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <nav
        aria-label={`${title} navigation`}
        className="rounded-control border border-app-border bg-app-surface p-3"
      >
        {navigation}
      </nav>
      <section className="min-w-0 space-y-4">
        <header className="flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-xl font-semibold text-app-foreground">{title}</h1>
          {controls ? (
            <div
              aria-label={`${title} controls`}
              className="flex flex-wrap gap-2"
            >
              {controls}
            </div>
          ) : null}
        </header>
        <dl className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {metrics.map((metric) => (
            <div
              key={metric.label}
              className="rounded-control border border-app-border bg-app-surface p-3"
            >
              <dt className="text-sm text-app-muted-foreground">
                {metric.label}
              </dt>
              <dd className="mt-1 text-2xl font-semibold text-app-foreground">
                {metric.value}
              </dd>
              {metric.detail ? (
                <p className="mt-1 text-sm text-app-muted-foreground">
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
          className="rounded-control border border-app-border bg-app-surface p-4"
        >
          {children}
        </AsyncPanel>
      </section>
    </main>
  );
}

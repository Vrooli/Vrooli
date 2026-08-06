/**
 * @libraryId react-component-library:AssetDetailShell
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "../../../AsyncPanel/versions/1.0.0/AsyncPanel";
import type { ExperienceSurfaceState } from "../../../ExperienceSurface/versions/1.0.0/ExperienceSurface";

export interface AssetDetailShellProps {
  title: string;
  preview: ReactNode;
  metadata: ReactNode;
  activity?: ReactNode;
  activityState?: ExperienceSurfaceState;
  activitySurfaceId?: string;
  activityLoading?: ReactNode;
  activityEmpty?: ReactNode;
  activityPartial?: ReactNode;
  activityError?: ReactNode;
  onRetryActivity?: () => void;
  actions?: ReactNode;
  className?: string;
}

// AssetDetailShell keeps the primary asset and its metadata independently
// usable while supporting activity reports a real lifecycle boundary.
export function AssetDetailShell({
  title,
  preview,
  metadata,
  activity,
  activityState = "ready",
  activitySurfaceId = "asset-activity",
  activityLoading,
  activityEmpty,
  activityPartial,
  activityError,
  onRetryActivity,
  actions,
  className,
}: AssetDetailShellProps) {
  return (
    <main
      className={[
        "mx-auto grid max-w-7xl gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_20rem]",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <section className="min-w-0 space-y-4">
        <header className="flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-xl font-semibold text-app-foreground">{title}</h1>
          {actions ? (
            <div
              aria-label={`${title} actions`}
              className="flex flex-wrap gap-2"
            >
              {actions}
            </div>
          ) : null}
        </header>
        <section
          aria-label={`${title} preview`}
          className="overflow-hidden rounded-control border border-app-border bg-app-surface"
        >
          {preview}
        </section>
        <AsyncPanel
          surfaceId={activitySurfaceId}
          state={activityState}
          loading={activityLoading}
          empty={activityEmpty}
          partial={activityPartial}
          error={activityError}
          onRetry={onRetryActivity}
          className="rounded-control border border-app-border bg-app-surface p-4"
        >
          {activity}
        </AsyncPanel>
      </section>
      <aside
        aria-label={`${title} metadata`}
        className="rounded-control border border-app-border bg-app-surface p-4"
      >
        {metadata}
      </aside>
    </main>
  );
}

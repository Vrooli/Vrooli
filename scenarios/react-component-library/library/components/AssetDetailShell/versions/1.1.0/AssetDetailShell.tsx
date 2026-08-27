/**
 * @libraryId react-component-library:AssetDetailShell
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "../../../AsyncPanel/versions/1.0.2/AsyncPanel";
import type { ExperienceSurfaceState } from "../../../ExperienceSurface/versions/1.0.2/ExperienceSurface";
import { assetDetailShellStyles } from "./styles";

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
    <>
      <style
        data-rcl-asset-detail-shell-styles
        dangerouslySetInnerHTML={{ __html: assetDetailShellStyles }}
      />
      <main data-rcl-asset-detail className={className}>
        <section data-rcl-asset-detail-primary>
          <header data-rcl-asset-detail-header>
            <h1 data-rcl-asset-detail-title>{title}</h1>
            {actions ? (
              <div
                data-rcl-asset-detail-actions
                aria-label={`${title} actions`}
              >
                {actions}
              </div>
            ) : null}
          </header>
          <section
            aria-label={`${title} preview`}
            data-rcl-asset-detail-preview
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
            className="rcl-asset-detail-activity"
          >
            {activity}
          </AsyncPanel>
        </section>
        <aside aria-label={`${title} metadata`} data-rcl-asset-detail-metadata>
          {metadata}
        </aside>
      </main>
    </>
  );
}

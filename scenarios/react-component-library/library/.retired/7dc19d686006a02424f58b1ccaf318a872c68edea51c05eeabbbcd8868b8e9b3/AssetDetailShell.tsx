/**
 * @libraryId react-component-library:AssetDetailShell
 * @displayName Asset Detail Shell
 * @description A responsive asset-detail composition with preview, metadata, and lifecycle-aware supporting activity.
 * @version 1.1.4
 * @tags ["layout","asset-detail","preview","metadata","experience"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import type { ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
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

export const AssetDetailShell = withClassName(function AssetDetailShell({
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
      <StyleSheet
        name="asset-detail-shell-1-1-1"
        css={assetDetailShellStyles}
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
});

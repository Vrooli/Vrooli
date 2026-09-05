/**
 * @libraryId react-component-library:AssetDetailShell
 * @displayName Asset Detail Shell
 * @version 1.1.6
 * @tags ["layout","asset-detail","preview","metadata","experience"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import type { ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
export const assetDetailShellStyles = `
[data-rcl-asset-detail] { display: grid; inline-size: min(100%, calc(var(--space-xl) * 32)); min-inline-size: 0; margin-inline: auto; grid-template-columns: minmax(0, 1fr); gap: var(--space-md); padding: var(--space-md); color: var(--color-foreground); }
[data-rcl-asset-detail-primary] { display: grid; min-inline-size: 0; gap: var(--space-md); }
[data-rcl-asset-detail-header] { display: flex; min-inline-size: 0; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-xs); }
[data-rcl-asset-detail-title] { min-inline-size: 0; margin: 0; color: var(--color-foreground); font: var(--text-title); letter-spacing: -0.02em; overflow-wrap: anywhere; }
[data-rcl-asset-detail-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs); }
[data-rcl-asset-detail-actions] > button:not([data-rcl-control]), [data-rcl-asset-detail-actions] > a:not([data-rcl-control]) { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding-inline: var(--space-sm); font: var(--text-label); text-decoration: none; cursor: pointer; }
[data-rcl-asset-detail-preview], [data-rcl-asset-detail-metadata], [data-rcl-asset-detail-activity], .rcl-asset-detail-activity { min-inline-size: 0; align-self: start; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); }
[data-rcl-asset-detail-preview] { min-block-size: calc(var(--space-xl) * 6); display: grid; place-items: center; overflow: hidden; background: radial-gradient(circle at 20% 10%, color-mix(in srgb, var(--color-primary) 13%, transparent), transparent 45%), var(--color-surface); }
[data-rcl-asset-detail-preview] img { display: block; max-inline-size: 100%; block-size: auto; }
[data-rcl-asset-detail-preview] > * { max-inline-size: 100%; }
[data-rcl-asset-detail-metadata] { padding: var(--space-md); }
[data-rcl-asset-detail-activity], .rcl-asset-detail-activity { padding: var(--space-md); }
[data-rcl-asset-detail] :focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@media (min-width: 60rem) { [data-rcl-asset-detail] { grid-template-columns: minmax(0, 1fr) minmax(0, min(var(--sidebar-width), 36%)); } }
@media (max-width: 36rem) { [data-rcl-asset-detail] { padding: var(--space-sm); } [data-rcl-asset-detail-header] { align-items: flex-start; } [data-rcl-asset-detail-actions] { inline-size: 100%; } [data-rcl-asset-detail-actions] > button:not([data-rcl-control]), [data-rcl-asset-detail-actions] > a:not([data-rcl-control]) { flex: 1 1 auto; } }
@media (forced-colors: active) { [data-rcl-asset-detail-preview], [data-rcl-asset-detail-metadata], [data-rcl-asset-detail-activity] { border-color: CanvasText; background: Canvas; box-shadow: none; } }
`;
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
      <StyleSheet name="asset-detail-shell-1-1-1" css={assetDetailShellStyles} />
      <main data-rcl-asset-detail className={className}>
        <section data-rcl-asset-detail-primary>
          <header data-rcl-asset-detail-header>
            <h1 data-rcl-asset-detail-title>{title}</h1>
            {actions ? (
              <div data-rcl-asset-detail-actions aria-label={`${title} actions`}>
                {actions}
              </div>
            ) : null}
          </header>
          <section aria-label={`${title} preview`} data-rcl-asset-detail-preview>
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

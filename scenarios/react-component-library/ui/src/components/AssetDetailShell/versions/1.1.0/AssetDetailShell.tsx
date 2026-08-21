/**
 * @vrooliComponentSource react-component-library:AssetDetailShell
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption a1876380-85df-4ed9-a8b1-ce6008e168a0
 * @vrooliComponentAppliedAt 2026-08-11T00:47:57Z
 * @vrooliComponentSourceSha256 7c43442072660b4378066d12d672dc1d190209a13cb80fc268bb2180a85127db
 * @vrooliComponentDriftHash c1276bed5e9dc293e0ee236bc00335fba35a4b4bfbe7318e50f4453f20831b5f
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "./AsyncPanel";
import type { ExperienceSurfaceState } from "../../../ExperienceSurface/versions/1.0.0/ExperienceSurface";
import { assetDetailShellStyles } from "./styles";
import { useComponentStyles } from "../../../../hooks/useComponentStyles";

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
  useComponentStyles("rcl-asset-detail-shell", assetDetailShellStyles);

  return (
    <>
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
}

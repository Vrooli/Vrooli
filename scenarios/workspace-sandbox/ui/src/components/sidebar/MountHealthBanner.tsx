/**
 * MountHealthBanner consolidates repeated mount-health failures across
 * multiple sandboxes into a single banner with a "Restart all" CTA.
 * Used by the Active tab; History items have no live mount.
 */

import { useState } from "react";
import { AlertCircle, Info, Loader2, RefreshCw } from "lucide-react";

import type { Sandbox } from "../../lib/api";
import { MOUNT_INFO_TEXT } from "./SandboxItem";
import { useBannerData } from "./useBannerData";

interface MountHealthBannerProps {
  sandboxes: Sandbox[];
  onRestartUnhealthy?: () => void;
  restartingIds?: Set<string>;
}

export function MountHealthBanner({
  sandboxes,
  onRestartUnhealthy,
  restartingIds,
}: MountHealthBannerProps) {
  const { banners } = useBannerData(sandboxes);
  const [infoOpen, setInfoOpen] = useState(false);

  if (banners.length === 0) return null;

  return (
    <>
      {banners.map((banner) => (
        <div
          key={banner.message}
          className="mb-3 px-3 py-2 rounded-lg bg-amber-950/30 border border-amber-800/50"
          data-testid="mount-warning-banner"
        >
          <div className="flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-amber-400 flex-shrink-0 mt-0.5" />
            <div className="min-w-0 flex-1">
              <p className="text-xs text-amber-300">{banner.message}</p>
              <p className="text-[10px] text-amber-400/70 mt-0.5">
                Affects {banner.count} sandboxes
              </p>
            </div>
            <div className="flex items-center gap-1 flex-shrink-0">
              <button
                className="p-1 rounded hover:bg-amber-900/40 transition-colors text-amber-400"
                onClick={() => setInfoOpen((v) => !v)}
                data-testid="banner-info-button"
                aria-label="Mount warning details"
              >
                <Info className="h-3.5 w-3.5" />
              </button>
              {onRestartUnhealthy && (
                <button
                  className="flex items-center gap-1 px-2 py-1 rounded text-[11px] font-medium text-amber-300 bg-amber-900/40 hover:bg-amber-900/60 transition-colors disabled:opacity-50"
                  onClick={onRestartUnhealthy}
                  disabled={!!restartingIds && restartingIds.size > 0}
                  data-testid="banner-restart-all-button"
                >
                  {restartingIds && restartingIds.size > 0 ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <RefreshCw className="h-3 w-3" />
                  )}
                  Restart all
                </button>
              )}
            </div>
          </div>
          {infoOpen && (
            <p
              className="mt-2 text-[10px] text-amber-300/70 leading-relaxed pl-6"
              data-testid="banner-info-text"
            >
              {MOUNT_INFO_TEXT}
            </p>
          )}
        </div>
      ))}
    </>
  );
}

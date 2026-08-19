import { useEffect, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { chromeTheme } from "../../lib/chromeTheme";
import Banner from "./Banner";
import { BANNER_THEME_COLOR } from "./arbitrate";
import type { BannerDamping } from "./damping";
import { useBannerPresentation } from "./useBannerPresentation";
import type { MaybeBanner } from "./types";

export interface BannerRegionProps {
  /** Every possible notice; falsy entries are inactive conditions. */
  readonly banners: readonly MaybeBanner[];
  /**
   * Region-wide appearance/disappearance timing, merged over the per-tone
   * defaults. For hosts whose chrome wants a different temperament overall;
   * a single noisy condition should carry its own override instead.
   */
  readonly damping?: Partial<BannerDamping>;
}

/**
 * The single arbitrated home for top-chrome notices.
 *
 * Renders the highest-priority banner in full and collapses the rest behind
 * one summary row, so N simultaneous conditions cost one banner's height plus
 * a line instead of N banners shoving the workspace down. The region's own
 * height is capped and scrolls internally, so an unexpected pile-up can never
 * push the terminal off-screen.
 *
 * Appearance and disappearance are damped — see `damping.ts`. The region owes
 * its reader a stable surface even when a source is spamming state changes,
 * and it is the only place that can honour that across all notices at once.
 *
 * It also publishes the active tone to `chromeTheme`, which tints the
 * Android/PWA status bar to match.
 */
export default function BannerRegion({ banners, damping }: BannerRegionProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const { presented, dismiss } = useBannerPresentation(banners, damping);

  const primary = presented[0];
  const overflow = presented.slice(1);

  const tone = primary?.tone ?? null;
  useEffect(() => {
    chromeTheme.setBannerOverride(tone ? BANNER_THEME_COLOR[tone] : null);
    return () => { chromeTheme.setBannerOverride(null); };
  }, [tone]);

  // Collapse again once the pile-up clears, so the next one starts closed.
  useEffect(() => {
    if (overflow.length === 0 && expanded) setExpanded(false);
  }, [overflow.length, expanded]);

  if (!primary) return null;

  return (
    <div
      data-wc-banner-region=""
      data-testid="banner-region"
      className="wc-stable-theme"
      role="region"
      aria-label={t(strings.banners.regionLabel)}
    >
      <Banner banner={primary} onDismiss={dismiss} />

      {overflow.length > 0 ? (
        <>
          <button
            type="button"
            data-wc-banner-overflow-toggle=""
            data-testid="banner-overflow-toggle"
            aria-expanded={expanded}
            onClick={() => { setExpanded((open) => !open); }}
          >
            <span>
              {expanded
                ? t(strings.banners.showLess)
                : t(strings.banners.moreNotices, { count: overflow.length })}
            </span>
            {expanded ? (
              <ChevronUp className="h-3 w-3" aria-hidden="true" />
            ) : (
              <ChevronDown className="h-3 w-3" aria-hidden="true" />
            )}
          </button>

          {expanded
            ? overflow.map((banner) => (
                <Banner key={banner.id} banner={banner} onDismiss={dismiss} compact />
              ))
            : null}
        </>
      ) : null}
    </div>
  );
}

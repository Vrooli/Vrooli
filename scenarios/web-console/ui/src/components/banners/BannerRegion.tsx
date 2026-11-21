import { useTranslation } from "react-i18next";
import { BannerRegion as LibraryBannerRegion } from "@vrooli/react-component-library/Banner";
import type { BannerDamping } from "@vrooli/react-component-library/Banner";
import { strings } from "../../consts/strings";
import type { MaybeBanner } from "./types";

export interface BannerRegionProps {
  /** Every possible notice; falsy entries are inactive conditions. */
  readonly banners: readonly MaybeBanner[];
  /**
   * Region-wide appearance/disappearance timing, merged over the per-tone
   * defaults. A single noisy condition should carry its own override instead.
   */
  readonly damping?: Partial<BannerDamping>;
}

/**
 * web-console's top-chrome notice region.
 *
 * Arbitration, damping, collapse/expand, and status-bar tinting are the
 * library's — this file is only the app's configuration of them: its locale
 * bridge and its stable test ids.
 *
 * The region is the one owner of the notch tint, and it derives that from the
 * *presented* set rather than from the raw conditions. That distinction is
 * load-bearing: a condition outlives the reader dismissing its notice, so a
 * second reader computing the tint from conditions leaves the status bar
 * coloured for a banner that is no longer on screen.
 */
export default function BannerRegion({ banners, damping }: BannerRegionProps) {
  const { t } = useTranslation();
  return (
    <LibraryBannerRegion
      banners={banners}
      damping={damping}
      ariaLabel={t(strings.banners.regionLabel)}
      // The library's default concatenates the count with an English phrase.
      // This app has real pluralisation, so it renders the label itself.
      overflowLabel={(count, expanded) =>
        expanded ? t(strings.banners.showLess) : t(strings.banners.moreNotices, { count })
      }
      testId="banner-region"
      className="wc-stable-theme fixed inset-x-0 top-0 z-[var(--layer-banner)] pt-[env(safe-area-inset-top)]"
    />
  );
}

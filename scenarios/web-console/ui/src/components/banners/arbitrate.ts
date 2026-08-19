import type { BannerDescriptor, BannerTone, MaybeBanner } from "./types";

export interface BannerArbitration {
  /** The one banner shown in full, or null when nothing is active. */
  readonly primary: BannerDescriptor | null;
  /** Everything else, in priority order, collapsed behind a summary row. */
  readonly overflow: readonly BannerDescriptor[];
  /** primary + overflow, for callers that need the whole active set. */
  readonly active: readonly BannerDescriptor[];
}

/**
 * Decide what the top chrome shows. Pure, so the "do banners stack?" question
 * is answered by a unit test rather than by looking at the running app.
 *
 * Rules, in order:
 *   1. Falsy entries are inactive conditions and drop out.
 *   2. Duplicate ids collapse — the highest-priority instance wins, so a
 *      condition raised from two places cannot render twice.
 *   3. Sort by priority descending; ties break on id so the order is stable
 *      across renders and does not flicker when unrelated state changes.
 *   4. The head renders in full; the tail collapses.
 */
export function arbitrateBanners(banners: readonly MaybeBanner[]): BannerArbitration {
  const byId = new Map<string, BannerDescriptor>();
  for (const banner of banners) {
    if (!banner) continue;
    const existing = byId.get(banner.id);
    if (existing && existing.priority >= banner.priority) continue;
    byId.set(banner.id, banner);
  }

  const active = [...byId.values()].sort(
    (a, b) => b.priority - a.priority || a.id.localeCompare(b.id),
  );

  return {
    primary: active[0] ?? null,
    overflow: active.slice(1),
    active,
  };
}

/**
 * The class that tints the iOS/PWA safe-area strip above the region.
 *
 * This used to be a single ternary in Workspace hard-wired to one banner (the
 * voice fallback notice), so ten other notices left the notch showing the
 * previous surface colour. Deriving it from the arbitration means whatever is
 * on top owns the strip, always.
 */
export function bannerFillClassName(primary: BannerDescriptor | null): string | undefined {
  if (!primary) return undefined;
  return TONE_FILL[primary.tone];
}

const TONE_FILL: Record<BannerTone, string> = {
  danger: "wc-banner-fill-danger",
  warning: "wc-banner-fill-warning",
  info: "wc-banner-fill-info",
};

/**
 * The `<meta name="theme-color">` value matching a tone, for the Android/PWA
 * status bar. Opaque on purpose — the OS bar cannot composite alpha.
 */
export const BANNER_THEME_COLOR: Record<BannerTone, string> = {
  danger: "#3f0d0d",
  warning: "#3d2a06",
  info: "#0c2a3d",
};

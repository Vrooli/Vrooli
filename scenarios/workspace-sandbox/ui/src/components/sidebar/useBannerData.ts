/**
 * Build the mount-health banner list from a set of sandboxes.
 *
 * When the same mount-health message is shared by ≥2 sandboxes we
 * surface a single consolidated banner with a "Restart all" affordance
 * and suppress the inline warning on each individual row. The set of
 * "consolidated" messages is returned alongside the banner list so the
 * row renderer can suppress the inline warning to avoid duplication.
 */

import { useMemo } from "react";

import type { Sandbox } from "../../lib/api";

export interface BannerData {
  banners: Array<{ message: string; count: number }>;
  consolidatedMessages: Set<string>;
}

export function useBannerData(sandboxes: Sandbox[]): BannerData {
  return useMemo(() => {
    const messageCounts = new Map<string, number>();
    for (const sb of sandboxes) {
      if (sb.mountHealth && !sb.mountHealth.healthy) {
        const msg = sb.mountHealth.hint || sb.mountHealth.error || "Mount unhealthy";
        messageCounts.set(msg, (messageCounts.get(msg) || 0) + 1);
      }
    }

    const consolidated = new Set<string>();
    const bannerList: BannerData["banners"] = [];
    for (const [msg, count] of messageCounts) {
      if (count >= 2) {
        consolidated.add(msg);
        bannerList.push({ message: msg, count });
      }
    }
    return { banners: bannerList, consolidatedMessages: consolidated };
  }, [sandboxes]);
}

import { describe, expect, it } from "vitest";
import { arbitrateBanners, bannerFillClassName } from "../arbitrate";
import { BANNER_PRIORITY, type BannerDescriptor } from "../types";

function banner(over: Partial<BannerDescriptor> & Pick<BannerDescriptor, "id">): BannerDescriptor {
  return {
    testId: `${over.id}-banner`,
    tone: "info",
    priority: 10,
    title: over.id,
    ...over,
  };
}

describe("banner arbitration", () => {
  it("shows nothing when no condition holds", () => {
    const { primary, overflow } = arbitrateBanners([null, false, undefined, "", 0]);
    expect(primary).toBeNull();
    expect(overflow).toHaveLength(0);
  });

  it("shows exactly one banner in full no matter how many conditions hold", () => {
    const { primary, overflow } = arbitrateBanners([
      banner({ id: "a", priority: 10 }),
      banner({ id: "b", priority: 90 }),
      banner({ id: "c", priority: 50 }),
      banner({ id: "d", priority: 20 }),
    ]);
    expect(primary?.id).toBe("b");
    expect(overflow.map((entry) => entry.id)).toEqual(["c", "d", "a"]);
  });

  it("collapses a duplicate id to its highest-priority instance", () => {
    // The same underlying fault raised from two places must not render twice.
    const { primary, overflow, active } = arbitrateBanners([
      banner({ id: "audio-unavailable", priority: 20 }),
      banner({ id: "audio-unavailable", priority: 70 }),
    ]);
    expect(active).toHaveLength(1);
    expect(primary?.priority).toBe(70);
    expect(overflow).toHaveLength(0);
  });

  it("orders ties by id so the region does not reshuffle between renders", () => {
    const first = arbitrateBanners([
      banner({ id: "zebra", priority: 40 }),
      banner({ id: "alpha", priority: 40 }),
    ]);
    const second = arbitrateBanners([
      banner({ id: "alpha", priority: 40 }),
      banner({ id: "zebra", priority: 40 }),
    ]);
    expect(first.active.map((entry) => entry.id)).toEqual(["alpha", "zebra"]);
    expect(second.active.map((entry) => entry.id)).toEqual(first.active.map((entry) => entry.id));
  });

  it("ranks a lost connection above every voice notice", () => {
    const { primary } = arbitrateBanners([
      banner({ id: "voice-transcribing", priority: BANNER_PRIORITY.voiceTranscribing }),
      banner({ id: "voice-rejection", priority: BANNER_PRIORITY.voiceRejection }),
      banner({ id: "connection-lost", priority: BANNER_PRIORITY.connectionLost }),
    ]);
    expect(primary?.id).toBe("connection-lost");
  });

  it("tints the safe-area strip from the banner on top", () => {
    expect(bannerFillClassName(null)).toBeUndefined();
    expect(bannerFillClassName(banner({ id: "a", tone: "danger" }))).toBe("wc-banner-fill-danger");
    expect(bannerFillClassName(banner({ id: "a", tone: "warning" }))).toBe("wc-banner-fill-warning");
    expect(bannerFillClassName(banner({ id: "a", tone: "info" }))).toBe("wc-banner-fill-info");
  });
});

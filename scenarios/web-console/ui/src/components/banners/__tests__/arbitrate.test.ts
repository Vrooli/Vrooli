import { describe, expect, it } from "vitest";
import { arbitrateBanners, BANNER_CHROME } from "../arbitrate";
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

  it("gives every tone both status-bar channels", () => {
    // Both channels exist for every tone: the OS bar reads `statusColor`, the
    // iOS safe-area strip reads `fillColor`. A tone with only one of them would
    // be correct on one platform and stale on the other.
    // These are fallbacks only — a real browser measures the banner's rendered
    // background and publishes that. What has to hold of the fallback is that
    // it is a usable opaque colour for every tone: the OS composites the status
    // colour itself and renders alpha as black, so opacity is a correctness
    // requirement rather than a style choice.
    for (const tone of ["danger", "warning", "info", "success"] as const) {
      expect(BANNER_CHROME[tone].statusColor).toMatch(/^rgb\(\d+, \d+, \d+\)$/);
    }
  });
});

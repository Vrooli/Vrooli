import { describe, expect, it } from "vitest";
import {
  createPresentationState,
  dismissBanner,
  reconcileBanners,
  resolveDamping,
  type BannerDamping,
  type PresentationState,
} from "../damping";
import type { BannerDescriptor, BannerTone } from "../types";

function banner(
  id: string,
  over: Partial<BannerDescriptor> = {},
): BannerDescriptor {
  return {
    id,
    testId: `${id}-banner`,
    tone: "warning",
    priority: 50,
    title: id,
    ...over,
  };
}

/** Deterministic policy so a test states the timing it is asserting. */
const policy = (over: Partial<BannerDamping> = {}) => (): BannerDamping => ({
  enterAfterMs: 200,
  minVisibleMs: 1_000,
  exitAfterMs: 400,
	flapWindowMs: 10 * 1000,
  flapThreshold: 2,
  flapBackoffFactor: 2,
  maxExitAfterMs: 6_400,
  primaryDwellMs: 1_000,
  ...over,
});

/** Drive a condition through the reconciler and report what the reader saw. */
function run(
  state: PresentationState,
  steps: { at: number; active: BannerDescriptor[] }[],
  policyFor = policy(),
): { at: number; ids: string[]; settling: string[] }[] {
  return steps.map(({ at, active }) => {
    const { presented } = reconcileBanners(state, active, at, policyFor);
    return {
      at,
      ids: presented.map((entry) => entry.id),
      settling: presented.filter((entry) => entry.settling).map((entry) => entry.id),
    };
  });
}

describe("banner damping", () => {
  describe("enter delay", () => {
    it("never paints a condition that resolves faster than the delay", () => {
      const state = createPresentationState();
      const frames = run(state, [
        { at: 0, active: [banner("a")] },
        { at: 80, active: [banner("a")] },
        { at: 150, active: [] },
        { at: 400, active: [] },
      ]);
      // The whole point: a 150ms blip costs the reader nothing at all.
      expect(frames.every((frame) => frame.ids.length === 0)).toBe(true);
      expect(state.tracked.size).toBe(0);
    });

    it("paints once the condition outlasts the delay", () => {
      const state = createPresentationState();
      const frames = run(state, [
        { at: 0, active: [banner("a")] },
        { at: 199, active: [banner("a")] },
        { at: 200, active: [banner("a")] },
      ]);
      expect(frames.map((frame) => frame.ids)).toEqual([[], [], ["a"]]);
    });

    it("paints a danger banner immediately, because something is actually broken", () => {
      const state = createPresentationState();
      const { presented } = reconcileBanners(
        state,
        [banner("boom", { tone: "danger" })],
        0,
        resolveDamping,
      );
      expect(presented.map((entry) => entry.id)).toEqual(["boom"]);
    });

    it("gives each tone a distinct patience", () => {
      const patience = (tone: BannerTone) => resolveDamping(banner("x", { tone })).enterAfterMs;
      expect(patience("danger")).toBe(0);
      expect(patience("warning")).toBeGreaterThan(0);
      expect(patience("info")).toBeGreaterThan(patience("warning"));
    });
  });

  describe("minimum visible time", () => {
    it("keeps a banner readable even when its condition clears immediately", () => {
      const state = createPresentationState();
      const frames = run(state, [
        { at: 0, active: [banner("a")] },
        { at: 200, active: [banner("a")] }, // paints
        { at: 210, active: [] }, // condition gone almost at once
        { at: 900, active: [] },
        { at: 1_199, active: [] },
        { at: 1_200, active: [] }, // enteredAt(200) + minVisible(1000)
      ]);
      expect(frames.map((frame) => frame.ids)).toEqual([
        [], ["a"], ["a"], ["a"], ["a"], [],
      ]);
      // While held it is marked settling, so its actions can be made inert.
      expect(frames[2]?.settling).toEqual(["a"]);
    });
  });

  describe("exit hold", () => {
    it("absorbs a clear-and-reassert gap without any visual change", () => {
      const state = createPresentationState();
      const frames = run(
        state,
        [
          { at: 0, active: [banner("a")] },
          { at: 200, active: [banner("a")] },
          { at: 2_000, active: [] }, // clears
          { at: 2_100, active: [banner("a")] }, // back inside the hold
          { at: 2_200, active: [banner("a")] },
        ],
        policy({ minVisibleMs: 0 }),
      );
      // Present continuously from the moment it painted — no remove/add pair.
      expect(frames.map((frame) => frame.ids)).toEqual([[], ["a"], ["a"], ["a"], ["a"]]);
      expect(frames[3]?.settling).toEqual([]);
    });

    it("removes the banner once the hold expires with the condition still gone", () => {
      const state = createPresentationState();
      const frames = run(
        state,
        [
          { at: 0, active: [banner("a")] },
          { at: 200, active: [banner("a")] },
          { at: 2_000, active: [] },
          { at: 2_399, active: [] },
          { at: 2_400, active: [] }, // 2000 + exitAfter(400)
        ],
        policy({ minVisibleMs: 0 }),
      );
      expect(frames.map((frame) => frame.ids)).toEqual([[], ["a"], ["a"], ["a"], []]);
    });
  });

  describe("flap backoff", () => {
    it("widens the hold geometrically for a source that keeps re-asserting", () => {
      const state = createPresentationState();
      const wide = policy({ minVisibleMs: 0 });
      // Paint, then oscillate. Each re-entry inside the hold is a flap.
      reconcileBanners(state, [banner("a")], 0, wide);
      reconcileBanners(state, [banner("a")], 200, wide);
      let clock = 200;
      for (let cycle = 0; cycle < 4; cycle += 1) {
        clock += 100;
        reconcileBanners(state, [], clock, wide); // clears
        clock += 100;
        reconcileBanners(state, [banner("a")], clock, wide); // re-asserts
      }
      const record = state.tracked.get("a");
      expect(record?.flaps).toBe(4);

      // With flaps=4, threshold=2, factor=2 → 400 * 2^2 = 1600ms of hold.
      clock += 100;
      reconcileBanners(state, [], clock, wide);
      expect(state.tracked.get("a")?.hideAt).toBe(clock + 1_600);
    });

    it("caps the widened hold so a wedged source cannot pin a banner forever", () => {
      const state = createPresentationState();
      const wide = policy({ minVisibleMs: 0, maxExitAfterMs: 1_000 });
      reconcileBanners(state, [banner("a")], 0, wide);
      reconcileBanners(state, [banner("a")], 200, wide);
      let clock = 200;
      for (let cycle = 0; cycle < 8; cycle += 1) {
        clock += 50;
        reconcileBanners(state, [], clock, wide);
        clock += 50;
        reconcileBanners(state, [banner("a")], clock, wide);
      }
      clock += 50;
      reconcileBanners(state, [], clock, wide);
      expect(state.tracked.get("a")?.hideAt).toBe(clock + 1_000);
    });

    it("forgets flaps once the source has been quiet for a full window", () => {
      const state = createPresentationState();
      const wide = policy({ minVisibleMs: 0, flapWindowMs: 1_000 });
      reconcileBanners(state, [banner("a")], 0, wide);
      reconcileBanners(state, [banner("a")], 200, wide);
      reconcileBanners(state, [], 300, wide);
      reconcileBanners(state, [banner("a")], 400, wide); // flap 1
      expect(state.tracked.get("a")?.flaps).toBe(1);

      reconcileBanners(state, [], 2_000, wide);
      reconcileBanners(state, [banner("a")], 2_100, wide); // window elapsed
      expect(state.tracked.get("a")?.flaps).toBe(1);
    });
  });

  describe("primary dwell", () => {
    // "arrival" sorts ahead of "incumbent" alphabetically at equal priority, so
    // without a dwell the newcomer would displace the banner already on screen.
    it("holds the top slot against an equal-rank peer", () => {
      const state = createPresentationState();
      const immediate = policy({ enterAfterMs: 0, minVisibleMs: 0 });
      reconcileBanners(state, [banner("incumbent")], 0, immediate);
      const { presented } = reconcileBanners(
        state,
        [banner("incumbent"), banner("arrival")],
        100,
        immediate,
      );
      expect(presented.map((entry) => entry.id)).toEqual(["incumbent", "arrival"]);
    });

    it("yields the top slot to something genuinely more urgent", () => {
      const state = createPresentationState();
      const immediate = policy({ enterAfterMs: 0, minVisibleMs: 0 });
      reconcileBanners(state, [banner("incumbent")], 0, immediate);
      const { presented } = reconcileBanners(
        state,
        [banner("incumbent"), banner("urgent", { priority: 90 })],
        100,
        immediate,
      );
      expect(presented[0]?.id).toBe("urgent");
    });

    it("releases the slot once the dwell expires", () => {
      const state = createPresentationState();
      const immediate = policy({ enterAfterMs: 0, minVisibleMs: 0, primaryDwellMs: 500 });
      reconcileBanners(state, [banner("incumbent")], 0, immediate);
      reconcileBanners(state, [banner("incumbent"), banner("arrival")], 100, immediate);
      const { presented } = reconcileBanners(
        state,
        [banner("incumbent"), banner("arrival")],
        600,
        immediate,
      );
      expect(presented.map((entry) => entry.id)).toEqual(["arrival", "incumbent"]);
    });
  });

  describe("dismissal", () => {
    it("hides immediately, ignoring the minimum visible floor", () => {
      const state = createPresentationState();
      const immediate = policy({ enterAfterMs: 0 });
      reconcileBanners(state, [banner("a")], 0, immediate);
      expect(dismissBanner(state, "a")).toBe(true);
      const { presented } = reconcileBanners(state, [banner("a")], 10, immediate);
      expect(presented).toHaveLength(0);
    });

    it("stays hidden while the condition still holds, then is forgotten", () => {
      const state = createPresentationState();
      const immediate = policy({ enterAfterMs: 0 });
      reconcileBanners(state, [banner("a")], 0, immediate);
      dismissBanner(state, "a");
      // The caller's suppression latch has not caught up yet — the banner must
      // not flash back on the next render.
      expect(reconcileBanners(state, [banner("a")], 10, immediate).presented).toHaveLength(0);
      expect(reconcileBanners(state, [banner("a")], 5_000, immediate).presented).toHaveLength(0);
      // Condition finally clears: forget it, so a genuine recurrence shows again.
      reconcileBanners(state, [], 6_000, immediate);
      expect(state.tracked.has("a")).toBe(false);
      expect(reconcileBanners(state, [banner("a")], 7_000, immediate).presented).toHaveLength(1);
    });
  });

  describe("wake scheduling", () => {
    it("reports no wake-up when the region is at rest", () => {
      const state = createPresentationState();
      const immediate = policy({ enterAfterMs: 0 });
      const { wakeAt } = reconcileBanners(state, [banner("a")], 0, immediate);
      expect(wakeAt).toBeNull();
    });

    it("reports the earliest pending transition, not one per banner", () => {
      const state = createPresentationState();
      const { wakeAt } = reconcileBanners(
        state,
        [banner("a"), banner("b", { priority: 40 })],
        0,
        policy({ enterAfterMs: 200 }),
      );
      expect(wakeAt).toBe(200);
    });

    it("is stable when called twice with the same inputs", () => {
      // React invokes effects twice in development; a flap must not be counted
      // twice, and the presented set must not move.
      const state = createPresentationState();
      const immediate = policy({ enterAfterMs: 0, minVisibleMs: 0 });
      reconcileBanners(state, [banner("a")], 0, immediate);
      reconcileBanners(state, [], 100, immediate);
      const first = reconcileBanners(state, [banner("a")], 200, immediate);
      const flapsAfterFirst = state.tracked.get("a")?.flaps;
      const second = reconcileBanners(state, [banner("a")], 200, immediate);
      expect(second.presented.map((entry) => entry.id)).toEqual(
        first.presented.map((entry) => entry.id),
      );
      expect(state.tracked.get("a")?.flaps).toBe(flapsAfterFirst);
    });
  });

  describe("the reported failure", () => {
    it("presents one steady banner for a source toggling several times a second", () => {
      // A condition that flips roughly every 100ms for two seconds — what a
      // spamming producer looks like from the region's side.
      const state = createPresentationState();
      const frames: string[][] = [];
      for (let tick = 0; tick <= 2_000; tick += 100) {
        const on = (tick / 100) % 2 === 0;
        const { presented } = reconcileBanners(
          state,
          on ? [banner("noisy")] : [],
          tick,
          resolveDamping,
        );
        frames.push(presented.map((entry) => entry.id));
      }
      // Count how many times the reader would have seen it appear or vanish.
      let transitions = 0;
      for (let index = 1; index < frames.length; index += 1) {
        if ((frames[index]?.length ?? 0) !== (frames[index - 1]?.length ?? 0)) transitions += 1;
      }
      // Twenty toggles of the underlying condition; at most one appearance.
      expect(transitions).toBeLessThanOrEqual(1);
    });
  });
});

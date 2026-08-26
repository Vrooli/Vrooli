import type { BannerDescriptor, BannerTone } from "./types";

/**
 * Temporal damping for a declarative banner region.
 *
 * `arbitrateBanners` answers "which conditions hold, and in what rank" for one
 * render. It has no time dimension, so a source that toggles its condition five
 * times a second produces five mount/unmount cycles — the region strobes even
 * though every individual render is correct.
 *
 * A banner region cannot fix noisy upstream state, and it should not try to:
 * the source is where a flapping condition gets diagnosed. What the region owes
 * its reader is a *stable surface* regardless of how noisy its inputs are. That
 * is this module.
 *
 * Four levers, each answering a different failure:
 *
 *   enterAfterMs   A condition that resolves itself faster than this never
 *                  paints at all. This is the one that matters most: most
 *                  flicker is a condition that was never worth showing.
 *   minVisibleMs   Once painted, a banner is readable. Nothing that appears
 *                  may vanish before a human can finish the sentence.
 *   exitAfterMs    A condition that clears and immediately re-asserts is one
 *                  event, not two. Holding across the gap turns a remove/add
 *                  pair into no visual change at all.
 *   primaryDwellMs Two banners of similar rank trading the top slot is its own
 *                  flicker, distinct from either one appearing or leaving.
 *
 * Plus flap backoff: a source that keeps re-asserting inside `flapWindowMs`
 * gets its hold widened geometrically, so a pathological producer settles into
 * one steady banner instead of strobing. It is a damper, not a mute — the
 * banner stays visible and truthful; only its *removal* is deferred.
 *
 * The reconciler is pure over (state, active, now). No timers, no React, no
 * clock of its own — the caller supplies `now` and acts on the returned
 * `wakeAt`. That is what makes the policy testable at millisecond resolution
 * without a DOM, and what will make it liftable into RCL unchanged.
 */

export interface BannerDamping {
  /** A condition must hold continuously for this long before the banner paints. */
  enterAfterMs: number;
  /** Once painted, the banner stays at least this long even if the condition clears. */
  minVisibleMs: number;
  /** After the condition clears, hold the banner this long before removing it. */
  exitAfterMs: number;
  /** Re-entries inside this window count toward flapping. */
  flapWindowMs: number;
  /** Re-entries tolerated before the exit hold starts widening. */
  flapThreshold: number;
  /** Geometric widening applied per re-entry past the threshold. */
  flapBackoffFactor: number;
  /** Ceiling on the widened hold, so a wedged source cannot pin a banner forever. */
  maxExitAfterMs: number;
  /** Once a banner holds the top slot, keep it there this long unless outranked. */
  primaryDwellMs: number;
}

/**
 * Defaults by tone, because urgency and patience trade off against each other.
 *
 * A danger banner paints immediately — something is broken and the reader needs
 * to know now, so we accept the occasional flash of a fault that self-healed.
 * An info banner waits, because a progress notice that resolves in 300ms was
 * never worth a layout shift. Warning sits between the two.
 */
export const DEFAULT_DAMPING: Record<BannerTone, BannerDamping> = {
  danger: {
    enterAfterMs: 0,
    minVisibleMs: 2_000,
    exitAfterMs: 250,
	flapWindowMs: 10 * 1000,
    flapThreshold: 2,
    flapBackoffFactor: 2,
    maxExitAfterMs: 5_000,
    primaryDwellMs: 1_500,
  },
  warning: {
    enterAfterMs: 250,
    minVisibleMs: 1_500,
    exitAfterMs: 500,
	flapWindowMs: 10 * 1000,
    flapThreshold: 2,
    flapBackoffFactor: 2,
    maxExitAfterMs: 8_000,
    primaryDwellMs: 1_500,
  },
  info: {
    enterAfterMs: 450,
    minVisibleMs: 1_200,
    exitAfterMs: 700,
	flapWindowMs: 10 * 1000,
    flapThreshold: 2,
    flapBackoffFactor: 2,
    maxExitAfterMs: 8_000,
    primaryDwellMs: 1_500,
  },
};

/**
 * Build a policy resolver. Precedence, weakest first:
 *
 *   tone default  →  region override  →  per-banner override
 *
 * A region override is for hosts whose chrome wants a different temperament
 * overall (a kiosk that should never flash, a test that wants no waiting). A
 * per-banner override is for one condition that is known to be noisier or more
 * urgent than its tone implies.
 */
export function makeDampingResolver(
  regionOverride?: Partial<BannerDamping>,
): (banner: BannerDescriptor) => BannerDamping {
  return (banner) => ({
    ...DEFAULT_DAMPING[banner.tone],
    ...regionOverride,
    ...banner.damping,
  });
}

export const resolveDamping = makeDampingResolver();

/** No waiting and no holding — for tests asserting content rather than timing. */
export const INSTANT_DAMPING: Partial<BannerDamping> = {
  enterAfterMs: 0,
  minVisibleMs: 0,
  exitAfterMs: 0,
  primaryDwellMs: 0,
};

/**
 * `visible`  — painted, condition holds.
 * `pending`  — condition holds but has not yet outlasted `enterAfterMs`.
 * `settling` — condition has cleared; held on screen until `hideAt`.
 * `dismissed`— the reader dismissed it; stays hidden until the condition
 *              actually clears, so dismissal is a suppression latch rather than
 *              a queue pop that the next render immediately undoes.
 */
export type BannerPhase = "pending" | "visible" | "settling" | "dismissed";

export interface TrackedBanner {
  phase: BannerPhase;
  /** When the current phase began. */
  since: number;
  /** When this banner first painted; 0 while it never has. */
  enteredAt: number;
  /** Absolute time it may leave. Meaningful in `settling`. */
  hideAt: number;
  /** Last descriptor seen, so a settling banner still has content to render. */
  descriptor: BannerDescriptor;
  /** Re-entries inside the current flap window. */
  flaps: number;
  flapWindowStart: number;
}

export interface PresentationState {
  tracked: Map<string, TrackedBanner>;
  primaryId: string | null;
  /** Absolute time the current primary may be displaced by an equal-rank peer. */
  primaryUntil: number;
}

export function createPresentationState(): PresentationState {
  return { tracked: new Map(), primaryId: null, primaryUntil: 0 };
}

export type PresentedBanner = BannerDescriptor & {
  /**
   * The underlying condition has already cleared and this banner is only being
   * held for readability. Its actions are stale — acting on a condition that no
   * longer holds is how a "Retry" ends up retrying nothing — so the region
   * renders them inert.
   */
  settling: boolean;
};

export interface ReconcileResult {
  /** Ordered for display; the primary slot is first. */
  presented: PresentedBanner[];
  /**
   * The next instant at which presentation could change on its own. The caller
   * schedules exactly one timer for this — no polling, no per-banner intervals.
   * `null` means nothing is pending and the region is at rest.
   */
  wakeAt: number | null;
}

/** Widen the exit hold for a source that keeps re-asserting. */
function exitHoldFor(policy: BannerDamping, flaps: number): number {
  if (flaps <= policy.flapThreshold) return policy.exitAfterMs;
  const over = flaps - policy.flapThreshold;
  return Math.min(policy.maxExitAfterMs, policy.exitAfterMs * policy.flapBackoffFactor ** over);
}

function earlier(current: number | null, candidate: number): number | null {
  if (current === null) return candidate;
  return Math.min(current, candidate);
}

/**
 * Advance the presentation one step. Mutates `state` — it is the caller's
 * retained record — and returns what should be on screen right now.
 *
 * Pure with respect to (state, active, now): calling it twice with the same
 * arguments produces the same result and the same state, so React's
 * double-invocation in development cannot double-count a flap.
 */
export function reconcileBanners(
  state: PresentationState,
  active: readonly BannerDescriptor[],
  now: number,
  policyFor: (banner: BannerDescriptor) => BannerDamping = resolveDamping,
): ReconcileResult {
  const activeById = new Map(active.map((banner) => [banner.id, banner]));
  let wakeAt: number | null = null;

  // ── Conditions that hold ────────────────────────────────────────────────
  for (const banner of active) {
    const policy = policyFor(banner);
    const record = state.tracked.get(banner.id);

    if (!record) {
      const immediate = policy.enterAfterMs <= 0;
      state.tracked.set(banner.id, {
        phase: immediate ? "visible" : "pending",
        since: now,
        enteredAt: immediate ? now : 0,
        hideAt: 0,
        descriptor: banner,
        flaps: 0,
        flapWindowStart: now,
      });
      if (!immediate) wakeAt = earlier(wakeAt, now + policy.enterAfterMs);
      continue;
    }

    // Keep content live even while settling or pending.
    record.descriptor = banner;

    switch (record.phase) {
      case "pending":
        if (now - record.since >= policy.enterAfterMs) {
          record.phase = "visible";
          record.since = now;
          record.enteredAt = now;
        } else {
          wakeAt = earlier(wakeAt, record.since + policy.enterAfterMs);
        }
        break;

      case "settling": {
        // The condition came back before the hold expired. The banner never
        // left the screen, so this is not a new appearance — but it IS the
        // signature of a flapping source, and it widens the next hold.
        if (now - record.flapWindowStart > policy.flapWindowMs) {
          record.flaps = 0;
          record.flapWindowStart = now;
        }
        record.flaps += 1;
        record.phase = "visible";
        record.since = now;
        break;
      }

      case "visible":
      case "dismissed":
        break;
    }
  }

  // ── Conditions that have cleared ────────────────────────────────────────
  for (const [id, record] of [...state.tracked]) {
    if (activeById.has(id)) continue;
    const policy = policyFor(record.descriptor);

    switch (record.phase) {
      case "pending":
        // Never painted, and now it never will. This is the case that removes
        // most flicker outright: the reader is not shown a condition that did
        // not outlast their reaction time.
        state.tracked.delete(id);
        break;

      case "dismissed":
        // Dismissal held until the condition actually cleared. It has.
        state.tracked.delete(id);
        break;

      case "visible": {
        record.phase = "settling";
        record.since = now;
        record.hideAt = Math.max(
          now + exitHoldFor(policy, record.flaps),
          record.enteredAt + policy.minVisibleMs,
        );
        wakeAt = earlier(wakeAt, record.hideAt);
        break;
      }

      case "settling":
        if (now >= record.hideAt) state.tracked.delete(id);
        else wakeAt = earlier(wakeAt, record.hideAt);
        break;
    }
  }

  // ── Order ───────────────────────────────────────────────────────────────
  const visible: PresentedBanner[] = [];
  for (const [id, record] of state.tracked) {
    if (record.phase === "pending" || record.phase === "dismissed") continue;
    visible.push({ ...record.descriptor, id, settling: record.phase === "settling" });
  }
  visible.sort((a, b) => b.priority - a.priority || a.id.localeCompare(b.id));

  // Primary dwell: hold the top slot against equal-or-lower-rank churn, but
  // never against something genuinely more urgent.
  let primaryIndex = 0;
  if (state.primaryId !== null && now < state.primaryUntil) {
    const heldIndex = visible.findIndex((banner) => banner.id === state.primaryId);
    const contender = visible[0];
    const held = heldIndex > 0 ? visible[heldIndex] : undefined;
    if (held && contender && contender.priority <= held.priority) {
      primaryIndex = heldIndex;
    }
  }

  const primary = visible[primaryIndex] as PresentedBanner | undefined;
  if (!primary) {
    state.primaryId = null;
    state.primaryUntil = 0;
  } else {
    if (primary.id !== state.primaryId) {
      state.primaryId = primary.id;
      state.primaryUntil = now + policyFor(primary).primaryDwellMs;
    }
    if (visible.length > 1) wakeAt = earlier(wakeAt, state.primaryUntil);
    if (primaryIndex > 0) {
      visible.splice(primaryIndex, 1);
      visible.unshift(primary);
    }
  }

  return { presented: visible, wakeAt };
}

/**
 * Mark a banner dismissed by the reader. It hides immediately — no minimum
 * visible time applies to something the reader has explicitly closed — and
 * stays hidden until its condition clears, so a caller whose suppression latch
 * lags by a render does not see the banner flash back.
 */
export function dismissBanner(state: PresentationState, id: string): boolean {
  const record = state.tracked.get(id);
  if (!record || record.phase === "dismissed") return false;
  record.phase = "dismissed";
  if (state.primaryId === id) {
    state.primaryId = null;
    state.primaryUntil = 0;
  }
  return true;
}

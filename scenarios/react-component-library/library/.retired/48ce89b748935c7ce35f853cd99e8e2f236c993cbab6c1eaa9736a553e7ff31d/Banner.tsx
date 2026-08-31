/**
 * @libraryId react-component-library:Banner
 * @displayName Banner
 * @description Application-level notice chrome with priority arbitration, temporal damping, deterministic stacking, actions, and status-bar tinting.
 * @version 1.0.0
 * @tags ["feedback","status","chrome","arbitration","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource feedback.banner */
import {
  useCallback,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
  type ComponentType,
  type CSSProperties,
  type ReactNode,
} from "react";
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import {
  useChromeContribution,
  type ChromeContribution,
} from "@vrooli/react-component-library/ChromeTheme/1.0.0";
import { bannerStyles } from "./styles";

/* ─────────────────────────── Model ─────────────────────────── */

/**
 * A banner is DECLARED, not pushed.
 *
 * It exists exactly while its condition holds, and the host re-declares the
 * whole set every render. That is the difference between this and a toast, and
 * it is why the region owns no queue: there is nothing to pop, so a condition
 * cannot be silently lost, shown twice, or outlive the state that raised it.
 *
 * `onDismiss` is therefore the caller's own suppression latch — a notification
 * that the reader closed the notice — not the switch that removes it.
 */
export type BannerTone = "danger" | "warning" | "info" | "success";

export interface BannerAction {
  /** Stable key; also used to build the action's `data-testid`. */
  readonly id: string;
  readonly label: string;
  readonly onSelect: () => void;
  /** Shows a spinner and blocks re-entry. Implies `disabled`. */
  readonly busy?: boolean;
  readonly disabled?: boolean;
  readonly title?: string;
  /** Emphasised styling. At most one per banner. */
  readonly primary?: boolean;
  readonly icon?: ComponentType<{ className?: string }>;
  /** Override the derived `${banner.testId}-${action.id}`. */
  readonly testId?: string;
}

export interface BannerDescriptor {
  /** Stable identity. Duplicate ids collapse to the highest-priority instance. */
  readonly id: string;
  readonly tone: BannerTone;
  readonly title: ReactNode;
  /** Secondary line under the title. */
  readonly description?: ReactNode;
  /** Third line — typically a failed retry's error text. */
  readonly detail?: ReactNode;
  readonly actions?: readonly BannerAction[];
  /** Caller's cleanup when the reader dismisses. Does not gate the close button. */
  readonly onDismiss?: () => void;
  readonly dismissLabel?: string;
  /** Arbitration weight. Higher wins the one full-size slot. */
  readonly priority: number;
  readonly icon?: ComponentType<{ className?: string }>;
  /** Spin the icon (work in flight). */
  readonly spin?: boolean;
  /** Extra `data-*` attributes, written verbatim. */
  readonly data?: Readonly<Record<string, string>>;
  readonly testId: string;
  /** Override this condition's timing when it is noisier than its tone implies. */
  readonly damping?: Partial<BannerDamping>;
}

/**
 * Anything falsy is "this condition does not hold", so callers can inline the
 * condition (`someError && errorBanner(...)`) without a ternary. The empty
 * string is in the union because `string | null` guards produce it.
 */
export type MaybeBanner = BannerDescriptor | null | false | undefined | "" | 0;

/* ──────────────────────── Arbitration ──────────────────────── */

export interface BannerArbitration {
  readonly primary: BannerDescriptor | null;
  readonly overflow: readonly BannerDescriptor[];
  readonly active: readonly BannerDescriptor[];
}

/**
 * Decide what the chrome shows. Pure, so "do banners stack?" is answered by a
 * test rather than by looking at the running app.
 *
 *   1. Falsy entries are inactive conditions and drop out.
 *   2. Duplicate ids collapse — highest priority wins, so a condition raised
 *      from two places cannot render twice.
 *   3. Sort by priority descending; ties break on id so the order is stable
 *      across renders and does not flicker when unrelated state changes.
 *   4. The head renders in full; the tail collapses behind one summary row.
 */
export function arbitrateBanners(
  banners: readonly MaybeBanner[],
): BannerArbitration {
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
  return { primary: active[0] ?? null, overflow: active.slice(1), active };
}

/* ───────────────────────── Damping ─────────────────────────── */

/**
 * `arbitrateBanners` has no time dimension, so a source toggling its condition
 * five times a second produces five mount/unmount cycles — the region strobes
 * even though every individual render is correct.
 *
 * A region cannot fix noisy upstream state and should not try; the source is
 * where a flapping condition gets diagnosed. What the region owes its reader is
 * a *stable surface* regardless of how noisy its inputs are. That is this.
 */
export interface BannerDamping {
  /** A condition must hold continuously this long before the banner paints. */
  enterAfterMs: number;
  /** Once painted, it stays at least this long even if the condition clears. */
  minVisibleMs: number;
  /** After the condition clears, hold this long before removing. */
  exitAfterMs: number;
  /** Re-entries inside this window count toward flapping. */
  flapWindowMs: number;
  /** Re-entries tolerated before the exit hold starts widening. */
  flapThreshold: number;
  /** Geometric widening applied per re-entry past the threshold. */
  flapBackoffFactor: number;
  /** Ceiling on the widened hold, so a wedged source cannot pin a banner. */
  maxExitAfterMs: number;
  /** Once a banner holds the top slot, keep it there this long unless outranked. */
  primaryDwellMs: number;
}

/**
 * Defaults by tone, because urgency and patience trade against each other.
 *
 * A danger banner paints immediately — something is broken and the reader needs
 * to know now, so we accept the occasional flash of a fault that self-healed.
 * An info banner waits, because a progress notice that resolves in 300ms was
 * never worth a layout shift.
 */
export const DEFAULT_DAMPING: Record<BannerTone, BannerDamping> = {
  danger: {
    enterAfterMs: 0,
    minVisibleMs: 2_000,
    exitAfterMs: 250,
    flapWindowMs: 10_000,
    flapThreshold: 2,
    flapBackoffFactor: 2,
    maxExitAfterMs: 5_000,
    primaryDwellMs: 1_500,
  },
  warning: {
    enterAfterMs: 250,
    minVisibleMs: 1_500,
    exitAfterMs: 500,
    flapWindowMs: 10_000,
    flapThreshold: 2,
    flapBackoffFactor: 2,
    maxExitAfterMs: 8_000,
    primaryDwellMs: 1_500,
  },
  info: {
    enterAfterMs: 450,
    minVisibleMs: 1_200,
    exitAfterMs: 700,
    flapWindowMs: 10_000,
    flapThreshold: 2,
    flapBackoffFactor: 2,
    maxExitAfterMs: 8_000,
    primaryDwellMs: 1_500,
  },
  success: {
    enterAfterMs: 250,
    minVisibleMs: 1_500,
    exitAfterMs: 500,
    flapWindowMs: 10_000,
    flapThreshold: 2,
    flapBackoffFactor: 2,
    maxExitAfterMs: 8_000,
    primaryDwellMs: 1_500,
  },
};

/** Precedence, weakest first: tone default → region override → per-banner override. */
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
 * `pending`  — condition holds but has not outlasted `enterAfterMs`.
 * `settling` — condition cleared; held on screen until `hideAt`.
 * `dismissed`— the reader closed it; stays hidden until the condition actually
 *              clears, so dismissal is a suppression latch rather than a queue
 *              pop that the next render immediately undoes.
 */
export type BannerPhase = "pending" | "visible" | "settling" | "dismissed";

export interface TrackedBanner {
  phase: BannerPhase;
  since: number;
  enteredAt: number;
  hideAt: number;
  descriptor: BannerDescriptor;
  flaps: number;
  flapWindowStart: number;
}

export interface PresentationState {
  tracked: Map<string, TrackedBanner>;
  primaryId: string | null;
  primaryUntil: number;
}

export function createPresentationState(): PresentationState {
  return { tracked: new Map(), primaryId: null, primaryUntil: 0 };
}

export type PresentedBanner = BannerDescriptor & {
  /**
   * The condition has cleared and this banner is held only for readability.
   * Its actions are stale — acting on a condition that no longer holds is how
   * a "Retry" ends up retrying nothing — so the region renders them inert.
   */
  settling: boolean;
};

export interface ReconcileResult {
  presented: PresentedBanner[];
  /**
   * The next instant presentation could change on its own. The caller schedules
   * exactly one timer for it — no polling, no per-banner intervals. `null`
   * means the region is at rest and schedules nothing.
   */
  wakeAt: number | null;
}

function exitHoldFor(policy: BannerDamping, flaps: number): number {
  if (flaps <= policy.flapThreshold) return policy.exitAfterMs;
  const over = flaps - policy.flapThreshold;
  return Math.min(
    policy.maxExitAfterMs,
    policy.exitAfterMs * policy.flapBackoffFactor ** over,
  );
}

function earlier(current: number | null, candidate: number): number | null {
  return current === null ? candidate : Math.min(current, candidate);
}

/**
 * Advance presentation one step. Mutates `state` — the caller's retained record
 * — and returns what should be on screen right now.
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

    record.descriptor = banner; // keep content live while pending or settling

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

  for (const [id, record] of [...state.tracked]) {
    if (activeById.has(id)) continue;
    const policy = policyFor(record.descriptor);
    switch (record.phase) {
      case "pending":
        // Never painted, and now it never will. This removes most flicker
        // outright: the reader is not shown a condition that did not outlast
        // their reaction time.
        state.tracked.delete(id);
        break;
      case "dismissed":
        state.tracked.delete(id); // the latch held until the condition cleared
        break;
      case "visible":
        record.phase = "settling";
        record.since = now;
        record.hideAt = Math.max(
          now + exitHoldFor(policy, record.flaps),
          record.enteredAt + policy.minVisibleMs,
        );
        wakeAt = earlier(wakeAt, record.hideAt);
        break;
      case "settling":
        if (now >= record.hideAt) state.tracked.delete(id);
        else wakeAt = earlier(wakeAt, record.hideAt);
        break;
    }
  }

  const visible: PresentedBanner[] = [];
  for (const [id, record] of state.tracked) {
    if (record.phase === "pending" || record.phase === "dismissed") continue;
    visible.push({
      ...record.descriptor,
      id,
      settling: record.phase === "settling",
    });
  }
  visible.sort((a, b) => b.priority - a.priority || a.id.localeCompare(b.id));

  // Primary dwell: hold the top slot against equal-or-lower-rank churn, but
  // never against something genuinely more urgent.
  let primaryIndex = 0;
  if (state.primaryId !== null && now < state.primaryUntil) {
    const heldIndex = visible.findIndex(
      (banner) => banner.id === state.primaryId,
    );
    const contender = visible[0];
    const held = heldIndex > 0 ? visible[heldIndex] : undefined;
    if (held && contender && contender.priority <= held.priority)
      primaryIndex = heldIndex;
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
 * visible time applies to something the reader explicitly closed — and stays
 * hidden until its condition clears, so a caller whose suppression latch lags
 * by a render does not see the banner flash back.
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

/* ──────────────────────── Presentation ─────────────────────── */

export interface BannerPresentation {
  readonly presented: readonly PresentedBanner[];
  readonly dismiss: (id: string) => void;
}

/**
 * Turn "which conditions hold this render" into "what the reader sees", with
 * the flicker damped out.
 *
 *   • Presence lives in a ref and advances only inside an effect, so render
 *     stays pure and React's development double-render cannot double-count.
 *   • The reconciler reports the single next instant presentation could change,
 *     and this hook holds exactly ONE timer for it. A region at rest schedules
 *     nothing.
 *   • Re-rendering is driven by a change in the presented *set*, not by the
 *     descriptors, which are rebuilt every render by design. Content updates on
 *     an already-visible banner flow through without scheduling anything.
 */
export function useBannerPresentation(
  banners: readonly MaybeBanner[],
  damping?: Partial<BannerDamping>,
): BannerPresentation {
  const stateRef = useRef(createPresentationState());
  const policyFor = useMemo(() => makeDampingResolver(damping), [damping]);
  const [, bump] = useReducer((count: number) => count + 1, 0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const scheduledForRef = useRef<number | null>(null);
  const orderRef = useRef<string[]>([]);
  const settlingRef = useRef<string[]>([]);

  const { active } = arbitrateBanners(banners);
  const activeById = useMemo(
    () => new Map(active.map((banner) => [banner.id, banner])),
    [active],
  );

  useEffect(() => {
    const now = Date.now();
    const { presented, wakeAt } = reconcileBanners(
      stateRef.current,
      active,
      now,
      policyFor,
    );

    const order = presented.map((banner) => banner.id);
    const settling = presented
      .filter((banner) => banner.settling)
      .map((banner) => banner.id);
    const changed =
      order.join(" ") !== orderRef.current.join(" ") ||
      settling.join(" ") !== settlingRef.current.join(" ");
    orderRef.current = order;
    settlingRef.current = settling;

    if (wakeAt === null) {
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = null;
      scheduledForRef.current = null;
    } else if (scheduledForRef.current !== wakeAt) {
      if (timerRef.current) clearTimeout(timerRef.current);
      scheduledForRef.current = wakeAt;
      timerRef.current = setTimeout(
        () => {
          timerRef.current = null;
          scheduledForRef.current = null;
          bump();
        },
        Math.max(0, wakeAt - now),
      );
    }

    if (changed) bump();
  });

  useEffect(
    () => () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = null;
    },
    [],
  );

  const dismiss = useCallback((id: string) => {
    if (dismissBanner(stateRef.current, id)) bump();
  }, []);

  const presented = orderRef.current.flatMap<PresentedBanner>((id) => {
    const descriptor =
      activeById.get(id) ?? stateRef.current.tracked.get(id)?.descriptor;
    if (!descriptor) return [];
    return [{ ...descriptor, id, settling: settlingRef.current.includes(id) }];
  });

  return { presented, dismiss };
}

/* ───────────────────────── Chrome ──────────────────────────── */

/**
 * The status-bar appearance matching each tone.
 *
 * Two values per tone because the status bar is two platform mechanisms — see
 * `ChromeTheme`. `statusColor` is opaque because the OS composites it and
 * renders alpha as black; `fillColor` keeps the banner's translucency so the
 * safe-area strip and the banner beneath it resolve to the same colour.
 *
 * These are concrete values rather than token references because
 * `<meta name="theme-color">` is read by the OS, which cannot resolve a CSS
 * custom property. Hosts with their own palette override via `chromeForTone`.
 */
export const BANNER_CHROME: Record<BannerTone, ChromeContribution> = {
  danger: { statusColor: "#3f0d0d", fillColor: "rgb(69 10 10 / 0.55)" },
  warning: { statusColor: "#3d2a06", fillColor: "rgb(69 39 3 / 0.5)" },
  info: { statusColor: "#0c2a3d", fillColor: "rgb(8 47 73 / 0.5)" },
  success: { statusColor: "#062e16", fillColor: "rgb(5 46 22 / 0.5)" },
};

/* ────────────────────────── Glyphs ─────────────────────────── */

/**
 * Inline rather than drawn from `IconRegistry`, which carries no status
 * glyphs. Forcing the ones it has produces the mapping `Alert` currently ships
 * — a plus sign for "warning" — which is worse than an inline path. Tone is
 * also carried by shape here, not colour alone, which is what keeps the
 * distinction alive under forced-colors.
 */
function ToneGlyph({ tone }: { tone: BannerTone }) {
  const common = {
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 2,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
  };
  if (tone === "danger" || tone === "warning") {
    return (
      <svg {...common}>
        <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
        <path d="M12 9v4" />
        <path d="M12 17h.01" />
      </svg>
    );
  }
  if (tone === "success") {
    return (
      <svg {...common}>
        <circle cx="12" cy="12" r="10" />
        <path d="m9 12 2 2 4-4" />
      </svg>
    );
  }
  return (
    <svg {...common}>
      <circle cx="12" cy="12" r="10" />
      <path d="M12 16v-4" />
      <path d="M12 8h.01" />
    </svg>
  );
}

function CloseGlyph() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      aria-hidden="true"
    >
      <path d="M18 6 6 18" />
      <path d="m6 6 12 12" />
    </svg>
  );
}

function ChevronGlyph({ up }: { up: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={up ? "m18 15-6-6-6 6" : "m6 9 6 6 6-6"} />
    </svg>
  );
}

function SpinnerGlyph() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      aria-hidden="true"
    >
      <path d="M21 12a9 9 0 1 1-6.219-8.56" />
    </svg>
  );
}

/* ────────────────────────── Surfaces ───────────────────────── */

const TONE_ROLE: Record<BannerTone, "alert" | "status"> = {
  danger: "alert",
  warning: "status",
  info: "status",
  success: "status",
};

const TONE_LIVE: Record<BannerTone, "assertive" | "polite"> = {
  danger: "assertive",
  warning: "polite",
  info: "polite",
  success: "polite",
};

export interface BannerProps {
  readonly banner: PresentedBanner;
  /** Rendered inside the region's collapsed list — slightly tighter. */
  readonly compact?: boolean;
  /** Region-level dismissal: hides now, stays hidden until the condition clears. */
  readonly onDismiss?: (id: string) => void;
  readonly className?: string;
  readonly style?: CSSProperties;
}

/**
 * The one visual base every notice renders through.
 *
 * `role` and `aria-live` are derived from tone rather than chosen per banner,
 * which is what stops an urgent notice shipping as a silent `<div>`.
 */
export const Banner = withClassName(function Banner({
  banner,
  compact = false,
  onDismiss,
  className,
  style,
}: BannerProps) {
  const strings = useStrings();
  const Icon = banner.icon;
  const dismissLabel =
    banner.dismissLabel ?? strings("banner.dismiss", "Dismiss");
  // A settling banner's condition has already cleared; it is on screen only so
  // the reader can finish reading it. Its actions would operate on state that
  // no longer exists, so they are inert for the hold.
  const inert = banner.settling;
  // Closing a banner out from under work it started would strand that work, so
  // the close button waits for the action rather than disappearing. Withdrawing
  // it would change the control's footprint mid-action.
  const busy = banner.actions?.some((action) => action.busy) ?? false;

  return (
    <div
      data-rcl-banner
      data-tone={banner.tone}
      data-compact={compact ? "true" : undefined}
      data-settling={banner.settling ? "true" : undefined}
      data-testid={banner.testId}
      role={TONE_ROLE[banner.tone]}
      aria-live={TONE_LIVE[banner.tone]}
      className={className}
      style={style}
      {...banner.data}
    >
      <span data-rcl-banner-icon data-spin={banner.spin ? "true" : undefined}>
        {Icon ? <Icon /> : <ToneGlyph tone={banner.tone} />}
      </span>

      <div data-rcl-banner-content>
        <span data-rcl-banner-title>{banner.title}</span>
        {banner.description ? (
          <span data-rcl-banner-description>{banner.description}</span>
        ) : null}
        {banner.detail ? (
          <span data-rcl-banner-detail data-testid={`${banner.testId}-detail`}>
            {banner.detail}
          </span>
        ) : null}
      </div>

      {banner.actions?.length ? (
        <div data-rcl-banner-actions>
          {banner.actions.map((action) => {
            const ActionIcon = action.icon;
            return (
              <button
                key={action.id}
                type="button"
                data-rcl-banner-action
                data-primary={action.primary ? "true" : undefined}
                data-testid={action.testId ?? `${banner.testId}-${action.id}`}
                onClick={action.onSelect}
                disabled={inert || action.disabled || action.busy}
                title={action.title}
              >
                {action.busy ? (
                  <SpinnerGlyph />
                ) : ActionIcon ? (
                  <ActionIcon />
                ) : null}
                <span>{action.label}</span>
              </button>
            );
          })}
        </div>
      ) : null}

      {/*
        Every banner closes. A banner is by definition a non-blocking notice,
        and one the reader cannot remove is just a broken banner — a condition
        that genuinely must be acknowledged before work continues wants a
        dialog, not a strip of chrome.

        Unconditional is safe because the region owns dismissal: it hides the
        banner and keeps it hidden until the condition actually clears, so
        nothing is permanently silenced and a recurrence is shown again.
      */}
      <button
        type="button"
        data-rcl-banner-dismiss
        data-testid={`${banner.testId}-dismiss`}
        disabled={inert || busy}
        onClick={() => {
          // Hide here first, so the reader gets an immediate response even if
          // the caller's own suppression latch needs a render to catch up.
          onDismiss?.(banner.id);
          banner.onDismiss?.();
        }}
        title={dismissLabel}
        aria-label={dismissLabel}
      >
        <CloseGlyph />
      </button>
    </div>
  );
});

export interface BannerRegionProps {
  /** Every possible notice; falsy entries are inactive conditions. */
  readonly banners: readonly MaybeBanner[];
  /** Region-wide timing, merged over the per-tone defaults. */
  readonly damping?: Partial<BannerDamping>;
  /** Accessible name for the region landmark. */
  readonly ariaLabel?: string;
  /**
   * Tint the OS status bar to match the banner on top. Pass `false` for a
   * region that is not pinned to the top of the viewport — tinting the notch
   * from a notice halfway down the page would be a lie.
   */
  readonly tintStatusBar?: boolean;
  /** Override the per-tone status-bar colours with the host's own palette. */
  readonly chromeForTone?: Partial<Record<BannerTone, ChromeContribution>>;
  /** Where this region sits among other status-bar contributors. */
  readonly chromePriority?: number;
  readonly testId?: string;
  readonly className?: string;
  readonly style?: CSSProperties;
}

/**
 * The single arbitrated home for application-level notices.
 *
 * Renders the highest-priority banner in full and collapses the rest behind one
 * summary row, so N simultaneous conditions cost one banner's height plus a
 * line instead of N banners shoving the content down. The region's own height
 * is capped and scrolls internally, so an unexpected pile-up can never push the
 * application off-screen.
 *
 * It is also the ONE owner of the banner's status-bar appearance, and it
 * publishes the *presented* tone — the one the reader can actually see, after
 * damping and after dismissal. Nothing else may derive that from the raw
 * descriptor list: a condition outlives the reader dismissing its notice, so a
 * second reader leaves the status bar tinted for a banner that is gone.
 */
export const BannerRegion = withClassName(function BannerRegion({
  banners,
  damping,
  ariaLabel,
  tintStatusBar = true,
  chromeForTone,
  chromePriority = 0,
  testId,
  className,
  style,
}: BannerRegionProps) {
  const strings = useStrings();
  const [expanded, setExpanded] = useState(false);
  const { presented, dismiss } = useBannerPresentation(banners, damping);

  const primary = presented[0];
  const overflow = presented.slice(1);
  const tone = primary?.tone ?? null;

  useChromeContribution(
    tintStatusBar && tone
      ? (chromeForTone?.[tone] ?? BANNER_CHROME[tone])
      : null,
    { priority: chromePriority },
  );

  // Collapse again once the pile-up clears, so the next one starts closed.
  useEffect(() => {
    if (overflow.length === 0 && expanded) setExpanded(false);
  }, [overflow.length, expanded]);

  if (!primary) return null;

  return (
    <div
      data-rcl-banner-region
      data-testid={testId}
      role="region"
      aria-label={ariaLabel ?? strings("banner.regionLabel", "Notices")}
      className={className}
      style={style}
    >
      <StyleSheet name="banner-1-0-0-1" css={bannerStyles} />
      <Banner banner={primary} onDismiss={dismiss} />

      {overflow.length > 0 ? (
        <>
          <button
            type="button"
            data-rcl-banner-overflow-toggle
            data-testid={testId ? `${testId}-overflow-toggle` : undefined}
            aria-expanded={expanded}
            onClick={() => {
              setExpanded((open) => !open);
            }}
          >
            <span>
              {expanded
                ? strings("banner.showLess", "Show less")
                : `${overflow.length} ${strings("banner.moreNotices", "more notices")}`}
            </span>
            <ChevronGlyph up={expanded} />
          </button>

          {expanded
            ? overflow.map((banner) => (
                <Banner
                  key={banner.id}
                  banner={banner}
                  onDismiss={dismiss}
                  compact
                />
              ))
            : null}
        </>
      ) : null}
    </div>
  );
});

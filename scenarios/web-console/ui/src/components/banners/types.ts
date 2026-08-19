import type { ComponentType, ReactNode } from "react";
import type { BannerDamping } from "./damping";

/**
 * Banner model — the single vocabulary every top-chrome notice speaks.
 *
 * Before this module web-console had eleven notice surfaces, each with its own
 * markup, its own colour vocabulary (`wc-*` semantic tokens, raw `amber-*` /
 * `blue-*` / `sky-*` palette, and `app-*` tokens that this app does not even
 * define), its own `role`/`aria-live` choice, and no arbitration — so several
 * could stack and push the workspace down mid-sentence.
 *
 * A banner is DECLARED, not pushed. It exists exactly while its condition
 * holds; `onDismiss` is the caller's own suppression latch, not a queue pop.
 * That distinction is why banners are not toasts and why the region owns no
 * lifecycle state of its own.
 *
 * This lives in web-console while the shape is proven. It is deliberately
 * written to lift into react-component-library unchanged — presentation is
 * token-driven, arbitration is a pure function, and nothing here imports
 * web-console state.
 */

export type BannerTone = "danger" | "warning" | "info";

export interface BannerAction {
  /** Stable key; also used to build the action's `data-testid`. */
  readonly id: string;
  readonly label: string;
  readonly onSelect: () => void;
  /** Shows a spinner and blocks re-entry. Implies `disabled`. */
  readonly busy?: boolean;
  readonly disabled?: boolean;
  /** `title` attribute — the explanatory hover text. */
  readonly title?: string;
  /** Emphasised styling. At most one per banner. */
  readonly primary?: boolean;
  readonly icon?: ComponentType<{ className?: string }>;
  /**
   * Override the derived `${banner.testId}-${action.id}`. Only for actions
   * with an external contract — `health-retry-button` is driven by the
   * `open-workspace` BAS action, which retries until the workspace loads.
   */
  readonly testId?: string;
}

export interface BannerDescriptor {
  /** Stable identity. Duplicate ids collapse to the highest-priority instance. */
  readonly id: string;
  readonly tone: BannerTone;
  readonly title: string;
  /** Secondary line under the title. */
  readonly description?: ReactNode;
  /** Third line, used for a failed retry's error text. */
  readonly detail?: ReactNode;
  readonly actions?: readonly BannerAction[];
  /** Present iff the notice can be dismissed. */
  readonly onDismiss?: () => void;
  readonly dismissLabel?: string;
  /** Arbitration weight — see BANNER_PRIORITY. */
  readonly priority: number;
  readonly icon?: ComponentType<{ className?: string }>;
  /** Spin the icon (in-flight work). */
  readonly spin?: boolean;
  /** Extra `data-*` attributes, preserved from the pre-refactor markup so
   *  existing selectors and BAS cases keep working. Keys are written verbatim. */
  readonly data?: Readonly<Record<string, string>>;
  readonly testId: string;
  /**
   * Override the tone's appearance/disappearance timing. Reach for this when a
   * specific condition is known to be noisier or more urgent than its tone
   * implies — not to work around a source that should be fixed. See
   * `damping.ts` for what each lever does.
   */
  readonly damping?: Partial<BannerDamping>;
}

/**
 * Priority ladder. Bands, highest first:
 *
 *   90+  blocking and actionable — the app cannot do its job
 *   50…70 recoverable with data at risk — something of yours is retained
 *   20…45 transient progress — work is in flight and can be abandoned
 *   0…19  informational — nothing is wrong, nothing is at risk
 *
 * Keep new entries inside a band rather than inventing values between them;
 * the band is the design decision, the number is just an ordering.
 */
export const BANNER_PRIORITY = {
  connectionLost: 90,
  audioUnavailable: 70,
  crashRecovery: 65,
  voiceRejection: 60,
  createError: 55,
  summarizeError: 50,
  voiceError: 45,
  voiceStaleMic: 42,
  sessionRecovery: 35,
  voiceFallback: 30,
  voiceTranscribing: 25,
  ttsSpeaking: 22,
  enableAudio: 20,
  trackingDegraded: 10,
} as const;

export type BannerPriority = (typeof BANNER_PRIORITY)[keyof typeof BANNER_PRIORITY];

/**
 * Anything falsy is "this condition does not hold", so callers can inline the
 * condition (`someError && errorBanner(...)`) without a ternary. The empty
 * string is in the union because `string | null` guards produce it.
 */
export type MaybeBanner = BannerDescriptor | null | false | undefined | "" | 0;

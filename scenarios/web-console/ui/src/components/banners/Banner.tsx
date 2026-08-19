import { AlertTriangle, Info, Loader2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import type { PresentedBanner } from "./damping";
import type { BannerTone } from "./types";

/**
 * The one visual base every top-chrome notice renders through.
 *
 * Colour comes entirely from `--wc-banner-*` custom properties resolved by
 * `[data-tone]` in styles.css. Nothing here composes a Tailwind opacity
 * modifier onto a token that already carries alpha — that pattern is what
 * silently produced `rgb(R G B / a / b)` elsewhere in this app and dropped the
 * declaration on the floor.
 *
 * `role` and `aria-live` are derived from tone rather than chosen per banner,
 * which is how `ErrorBanner` previously shipped with neither.
 */

const TONE_ROLE: Record<BannerTone, "alert" | "status"> = {
  danger: "alert",
  warning: "status",
  info: "status",
};

const TONE_LIVE: Record<BannerTone, "assertive" | "polite"> = {
  danger: "assertive",
  warning: "polite",
  info: "polite",
};

const TONE_ICON: Record<BannerTone, typeof AlertTriangle> = {
  danger: AlertTriangle,
  warning: AlertTriangle,
  info: Info,
};

export interface BannerProps {
  readonly banner: PresentedBanner;
  /** Rendered inside the region's collapsed list — slightly tighter. */
  readonly compact?: boolean;
  /** Region-level dismissal: hides now, stays hidden until the condition clears. */
  readonly onDismiss?: (id: string) => void;
}

export default function Banner({ banner, compact = false, onDismiss }: BannerProps) {
  const { t } = useTranslation();
  const Icon = banner.icon ?? TONE_ICON[banner.tone];
  const dismissLabel = banner.dismissLabel ?? t(strings.banners.dismiss);
  // A settling banner's condition has already cleared; it is on screen only so
  // the reader can finish reading it. Its actions would operate on state that
  // no longer exists, so they are inert for the hold.
  const inert = banner.settling;
  // Closing a banner out from under work it started would strand that work, so
  // the close button waits for the action rather than disappearing. Withdrawing
  // it would change the control's footprint mid-action — the same layout shift
  // that got recovery actions moved out of the microphone button.
  const busy = banner.actions?.some((action) => action.busy) ?? false;

  return (
    <div
      data-wc-banner=""
      data-tone={banner.tone}
      data-compact={compact ? "" : undefined}
      data-settling={banner.settling ? "" : undefined}
      data-testid={banner.testId}
      role={TONE_ROLE[banner.tone]}
      aria-live={TONE_LIVE[banner.tone]}
      {...banner.data}
    >
      <Icon
        className={`h-3.5 w-3.5 shrink-0 ${banner.spin ? "animate-spin" : ""}`}
        aria-hidden="true"
      />

      <div data-wc-banner-content="">
        <span data-wc-banner-title="">{banner.title}</span>
        {banner.description ? (
          <span data-wc-banner-description="">{banner.description}</span>
        ) : null}
        {banner.detail ? (
          <span data-wc-banner-detail="" data-testid={`${banner.testId}-detail`}>
            {banner.detail}
          </span>
        ) : null}
      </div>

      {banner.actions?.length ? (
        <div data-wc-banner-actions="">
          {banner.actions.map((action) => {
            const ActionIcon = action.busy ? Loader2 : action.icon;
            return (
              <button
                key={action.id}
                type="button"
                data-wc-banner-action=""
                data-primary={action.primary ? "" : undefined}
                data-testid={action.testId ?? `${banner.testId}-${action.id}`}
                onClick={action.onSelect}
                disabled={inert || action.disabled || action.busy}
                title={action.title}
              >
                {ActionIcon ? (
                  <ActionIcon
                    className={`h-3.5 w-3.5 ${action.busy ? "animate-spin" : ""}`}
                  />
                ) : null}
                <span>{action.label}</span>
              </button>
            );
          })}
        </div>
      ) : null}

      {/*
        Every banner closes. A banner is by definition a non-blocking notice,
        and one the reader cannot remove is just a broken banner — if a
        condition genuinely must be acknowledged before work continues, it
        wants a dialog, not a strip of chrome.

        This is safe to make unconditional because the region owns dismissal:
        it hides the banner and keeps it hidden until the condition actually
        clears, so nothing is permanently silenced and a recurrence is shown
        again. `banner.onDismiss` is now only the caller's notification for
        when it needs to run its own cleanup (releasing retained audio,
        latching a session-scoped suppression) — not the switch that decides
        whether a close button exists.
      */}
      <button
        type="button"
        data-wc-banner-dismiss=""
        data-testid={`${banner.testId}-dismiss`}
        disabled={inert || busy}
        onClick={() => {
          // Hide it here first, so the reader gets an immediate response even
          // if the caller's own suppression latch needs a render to catch up.
          onDismiss?.(banner.id);
          banner.onDismiss?.();
        }}
        title={dismissLabel}
        aria-label={dismissLabel}
      >
        <X className="h-3.5 w-3.5" aria-hidden="true" />
      </button>
    </div>
  );
}

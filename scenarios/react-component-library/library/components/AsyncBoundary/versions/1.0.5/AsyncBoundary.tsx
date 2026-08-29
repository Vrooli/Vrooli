/**
 * @libraryId react-component-library:AsyncBoundary
 * @displayName AsyncBoundary
 * @description A stable async state boundary that preserves structure while pending, failed, or retryable content changes.
 * @version 1.0.5
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

/** @vrooliComponentSource react-component-library:AsyncBoundary */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.1.0";
import { useEffect, useId, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { useAnnounce } from "@vrooli/react-component-library/useAnnounce/1.0.1";
import { useNetworkStatus } from "@vrooli/react-component-library/useNetworkStatus/1.0.0";

export type AsyncBoundaryStatus =
  | "idle"
  | "pending"
  | "success"
  | "refreshing"
  | "stale"
  | "partial-error"
  | "error"
  | "offline";

export interface AsyncBoundaryProps {
  status?: AsyncBoundaryStatus;
  children?: ReactNode;
  /** The first-load surface. It appears after loadingDelay to avoid flicker. */
  pending?: ReactNode;
  /** A product-specific failure explanation. */
  error?: ReactNode;
  /** A concise failure heading for composed state assets. */
  errorTitle?: ReactNode;
  /** Retry the failed or unavailable operation. */
  retry?: () => void | Promise<void>;
  /** Force offline mode, or leave unset to follow the browser connection. */
  offline?: boolean;
  /** Opt out of browser online/offline events when the data source owns that state. */
  detectOffline?: boolean;
  /** Preserve the last useful content while a refresh-like state is visible. */
  preserveContent?: boolean;
  /** Delay the replacement skeleton so fast requests do not flash a loader. */
  loadingDelay?: number;
  className?: string;
  style?: CSSProperties;
  id?: string;
  "aria-label"?: string;
  "aria-labelledby"?: string;
}

const styles = `
  [data-rcl-async-boundary] {
    --rcl-async-border: var(--color-border, #cbd5e1);
    --rcl-async-surface: var(--color-surface, #ffffff);
    --rcl-async-raised: var(--color-surface-raised, #ffffff);
    --rcl-async-foreground: var(--color-foreground, #0f172a);
    --rcl-async-muted: var(--color-muted-foreground, #64748b);
    --rcl-async-primary: var(--color-primary, #2563eb);
    --rcl-async-danger: var(--color-danger, #dc2626);
    --rcl-async-warning: var(--color-warning, #d97706);
    --rcl-async-success: var(--color-success, #16a34a);
    min-inline-size: 0;
    overflow: clip;
    border: 1px solid var(--rcl-async-border);
    border-radius: var(--radius-panel, 0.5rem);
    background: var(--rcl-async-surface);
    color: var(--rcl-async-foreground);
    box-shadow: var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10));
  }
  [data-rcl-async-content] {
    min-inline-size: 0;
    padding: var(--space-lg, 32px);
  }
  [data-rcl-async-state] {
    display: flex;
    align-items: flex-start;
    gap: var(--space-xs, 12px);
    padding: var(--space-sm, 16px) var(--space-md, 24px);
    border-block-end: 1px solid color-mix(in srgb, currentColor 16%, var(--rcl-async-border));
    background: color-mix(in srgb, currentColor 6%, var(--rcl-async-raised));
    color: var(--rcl-async-muted);
    font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans));
  }
  [data-rcl-async-state][data-tone="warning"] { color: var(--rcl-async-warning); }
  [data-rcl-async-state][data-tone="danger"] { color: var(--rcl-async-danger); }
  [data-rcl-async-state][data-tone="success"] { color: var(--rcl-async-success); }
  [data-rcl-async-state][data-tone="primary"] { color: var(--rcl-async-primary); }
  [data-rcl-async-state-mark] {
    display: grid;
    flex: 0 0 auto;
    place-items: center;
    inline-size: 1.25rem;
    block-size: 1.25rem;
    margin-block-start: -.125rem;
    border: 1px solid currentColor;
    border-radius: 50%;
    font: 700 .75rem/1 system-ui, sans-serif;
  }
  [data-rcl-async-state-copy] { display: grid; gap: var(--space-3xs, 4px); min-inline-size: 0; }
  [data-rcl-async-state-title] { color: var(--rcl-async-foreground); font-weight: 700; }
  [data-rcl-async-state-detail] { color: var(--rcl-async-muted); }
  [data-rcl-async-state-action] { flex: 0 0 auto; margin-inline-start: auto; min-block-size: var(--tap-target-min, 44px); border: 1px solid currentColor; border-radius: var(--radius-control, 0.375rem); background: transparent; color: inherit; padding-inline: var(--space-xs, 12px); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); cursor: pointer; }
  [data-rcl-async-state-action]:hover { background: color-mix(in srgb, currentColor 10%, transparent); }
  [data-rcl-async-loading], [data-rcl-async-failure] {
    display: grid;
    min-block-size: 10rem;
    place-items: center;
    gap: var(--space-sm, 16px);
    padding: var(--space-xl, 40px);
    text-align: center;
  }
  [data-rcl-async-loading-copy], [data-rcl-async-failure-copy] { display: grid; gap: var(--space-2xs, 8px); max-inline-size: 38rem; }
  [data-rcl-async-loading-title], [data-rcl-async-failure-title] { font: var(--text-subtitle, 600 var(--text-subheading-size) / var(--text-subheading-line) var(--font-sans)); }
  [data-rcl-async-muted] { color: var(--rcl-async-muted); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }

  [data-rcl-async-spinner] {
    inline-size: 1.5rem;
    block-size: 1.5rem;
    border: 2px solid color-mix(in srgb, var(--rcl-async-primary) 24%, transparent);
    border-block-start-color: var(--rcl-async-primary);
    border-radius: 50%;
    animation: rcl-async-spin var(--dur-moderate, 280ms) linear infinite;
  }
  [data-rcl-async-skeleton] { display: grid; align-content: center; gap: var(--space-sm, 16px); inline-size: min(100%, 34rem); }
  [data-rcl-async-skeleton-line] { block-size: .75rem; border-radius: var(--radius-pill, 9999px); background: linear-gradient(90deg, var(--color-surface-muted, #f1f5f9), color-mix(in srgb, var(--rcl-async-primary) 12%, var(--color-surface-muted, #f1f5f9)), var(--color-surface-muted, #f1f5f9)); background-size: 200% 100%; animation: rcl-async-shimmer 1.4s var(--ease-standard, cubic-bezier(.2, 0, 0, 1)) infinite; }
  [data-rcl-async-skeleton-line="wide"] { inline-size: 100%; }
  [data-rcl-async-skeleton-line="medium"] { inline-size: 72%; }
  [data-rcl-async-skeleton-line="short"] { inline-size: 44%; }
  [data-rcl-async-retry] {
    min-block-size: var(--tap-target-min, 44px);
    border: 1px solid var(--rcl-async-primary);
    border-radius: var(--radius-control, 0.375rem);
    background: var(--rcl-async-primary);
    color: var(--color-primary-foreground, #ffffff);
    padding-inline: var(--space-md, 24px);
    font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans));
    cursor: pointer;
    transition: transform var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), filter var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1));
  }
  [data-rcl-async-retry]:hover { filter: brightness(1.06); transform: translateY(-1px); }
  [data-rcl-async-retry]:active { transform: translateY(0) scale(.98); }
  [data-rcl-async-retry][aria-busy="true"] { cursor: wait; opacity: .78; }
  @keyframes rcl-async-spin { to { transform: rotate(360deg); } }
  @keyframes rcl-async-shimmer { to { background-position: -200% 0; } }

  @media (max-width: 30rem) {
    [data-rcl-async-state] { display: grid; grid-template-columns: auto minmax(0, 1fr); }
    [data-rcl-async-state-action] { grid-column: 2; justify-self: start; margin-inline-start: 0; }
    [data-rcl-async-content] { padding: var(--space-md, 24px); }
    [data-rcl-async-loading], [data-rcl-async-failure] { min-block-size: 9rem; padding: var(--space-lg, 32px) var(--space-md, 24px); }
  }
`;

type StateTone = "neutral" | "primary" | "warning" | "danger" | "success";

const stateCopy: Record<
  Exclude<AsyncBoundaryStatus, "idle" | "success" | "error">,
  { title: string; detail: string; mark: string; tone: StateTone }
> = {
  pending: {
    title: "Loading",
    detail: "Preparing the latest content…",
    mark: "•",
    tone: "primary",
  },
  refreshing: {
    title: "Refreshing",
    detail: "Checking for newer information…",
    mark: "↻",
    tone: "primary",
  },
  stale: {
    title: "Showing saved content",
    detail: "We’ll update this view when fresh information arrives.",
    mark: "•",
    tone: "warning",
  },
  "partial-error": {
    title: "Some information needs attention",
    detail: "The rest of this view is still available.",
    mark: "!",
    tone: "warning",
  },
  offline: {
    title: "You’re offline",
    detail: "Showing the latest content saved on this device.",
    mark: "⌁",
    tone: "neutral",
  },
};

function isPreservingStatus(status: AsyncBoundaryStatus) {
  return ["refreshing", "stale", "partial-error", "offline"].includes(status);
}

function LoadingSkeleton() {
  return (
    <div data-rcl-async-skeleton aria-hidden="true">
      <span data-rcl-async-skeleton-line="wide" />
      <span data-rcl-async-skeleton-line="medium" />
      <span data-rcl-async-skeleton-line="short" />
    </div>
  );
}

export const AsyncBoundary = withClassName(function AsyncBoundary({
  status = "idle",
  children,
  pending,
  error = "We couldn’t load this content. Try again when you’re ready.",
  errorTitle = "We hit a snag",
  retry,
  offline,
  detectOffline = true,
  preserveContent,
  loadingDelay = 160,
  className,
  style,
  id,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
}: AsyncBoundaryProps) {
  const strings = useStrings();
  const isOnline = useNetworkStatus();
  const announce = useAnnounce();
  const generatedId = useId().replace(/:/g, "");
  const descriptionId = `${id ?? `rcl-async-${generatedId}`}-description`;
  const [pendingReady, setPendingReady] = useState(status !== "pending");
  const [retrying, setRetrying] = useState(false);
  const lastAnnounced = useRef<AsyncBoundaryStatus>();
  const networkOffline = offline ?? (detectOffline && !isOnline);
  const effectiveStatus: AsyncBoundaryStatus =
    networkOffline && status !== "error" ? "offline" : status;
  const keepContent = preserveContent ?? isPreservingStatus(effectiveStatus);
  const hasChildren = children !== undefined && children !== null;
  const showChildren =
    hasChildren && (effectiveStatus === "idle" || effectiveStatus === "success" || keepContent);

  useEffect(() => {
    if (effectiveStatus !== "pending") {
      setPendingReady(true);
      return undefined;
    }
    setPendingReady(false);
    const delay = Math.max(0, Math.min(2000, loadingDelay));
    const timer = window.setTimeout(() => setPendingReady(true), delay);
    return () => window.clearTimeout(timer);
  }, [effectiveStatus, loadingDelay]);

  useEffect(() => {
    if (lastAnnounced.current === effectiveStatus) return;
    lastAnnounced.current = effectiveStatus;
    const announcement =
      effectiveStatus === "pending"
        ? "Loading content"
        : effectiveStatus === "refreshing"
          ? "Refreshing content"
          : effectiveStatus === "stale"
            ? "Showing saved content"
            : effectiveStatus === "partial-error"
              ? "Some content needs attention"
              : effectiveStatus === "offline"
                ? "You are offline. Showing saved content"
                : effectiveStatus === "error"
                  ? "Content failed to load"
                  : effectiveStatus === "success"
                    ? "Content loaded"
                    : undefined;
    if (announcement)
      announce(announcement, {
        priority: effectiveStatus === "error" ? "assertive" : "polite",
      });
  }, [announce, effectiveStatus]);

  const copy = Object.prototype.hasOwnProperty.call(stateCopy, effectiveStatus)
    ? stateCopy[effectiveStatus as keyof typeof stateCopy]
    : undefined;
  const role =
    effectiveStatus === "error"
      ? "alert"
      : effectiveStatus === "pending" ||
          effectiveStatus === "refreshing" ||
          effectiveStatus === "offline"
        ? "status"
        : undefined;
  const busy = effectiveStatus === "pending" || effectiveStatus === "refreshing" || retrying;
  const description =
    copy?.detail ?? (effectiveStatus === "error" ? "Content failed to load." : "Content is ready.");
  const retryAction = async () => {
    if (!retry || retrying) return;
    setRetrying(true);
    try {
      await retry();
    } finally {
      setRetrying(false);
    }
  };

  return (
    <>
      <StyleSheet name="asyncboundary-1-0-4-1" css={styles} />
      <section
        id={id}
        className={className}
        style={style}
        role={role}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        aria-describedby={descriptionId}
        aria-busy={busy || undefined}
        data-rcl-async-boundary="true"
        data-rcl-async-status={effectiveStatus}
      >
        <span id={descriptionId} data-rcl-async-description>
          {description}
        </span>
        {copy && (
          <div data-rcl-async-state data-tone={copy.tone}>
            <span data-rcl-async-state-mark aria-hidden="true">
              {copy.mark}
            </span>
            <span data-rcl-async-state-copy>
              <span data-rcl-async-state-title>{copy.title}</span>
              <span data-rcl-async-state-detail>{copy.detail}</span>
            </span>
            {retry && effectiveStatus !== "error" && (
              <button
                data-testid="feedback.async-boundary"
                type="button"
                data-rcl-async-state-action
                aria-busy={retrying || undefined}
                disabled={retrying}
                onClick={() => void retryAction()}
              >
                {retrying ? "Trying again…" : effectiveStatus === "offline" ? "Try again" : "Retry"}
              </button>
            )}
          </div>
        )}
        {showChildren ? (
          <div data-rcl-async-content>{children}</div>
        ) : effectiveStatus === "pending" ? (
          <div data-rcl-async-loading>
            {pending ? (
              pending
            ) : pendingReady ? (
              <LoadingSkeleton />
            ) : (
              <span data-rcl-async-muted>
                {strings("feedback.async-boundary.loading-content", "Loading content")}
              </span>
            )}
            {!pending && pendingReady && (
              <span data-rcl-async-muted>
                {strings(
                  "feedback.async-boundary.this-may-take-a-moment",
                  "This may take a moment.",
                )}
              </span>
            )}
          </div>
        ) : effectiveStatus === "error" ? (
          <div data-rcl-async-failure>
            <span data-rcl-async-state-mark data-tone="danger" aria-hidden="true">
              !
            </span>
            <span data-rcl-async-failure-copy>
              <strong data-rcl-async-failure-title>{errorTitle}</strong>
              <span data-rcl-async-muted>{error}</span>
              {retry && (
                <span>
                  <button
                    data-testid="feedback.async-boundary"
                    type="button"
                    data-rcl-async-retry
                    aria-busy={retrying || undefined}
                    disabled={retrying}
                    onClick={() => void retryAction()}
                  >
                    {retrying ? "Trying again…" : "Try again"}
                  </button>
                </span>
              )}
            </span>
          </div>
        ) : (
          <div data-rcl-async-content>
            <span data-rcl-async-muted>
              {effectiveStatus === "offline"
                ? "Reconnect to refresh this view."
                : "No content is available yet."}
            </span>
          </div>
        )}
      </section>
    </>
  );
});

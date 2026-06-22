import type { ReactNode } from "react";
import { AlertTriangle, Inbox, Loader2, type LucideIcon } from "lucide-react";

import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "./button";

/**
 * Shared loading / error / empty state primitives.
 *
 * Every data-fetching surface in this app reaches one of three non-success
 * states; rendering them as one-line text drifts visually and skips affordances
 * (no retry, no guidance). These composites are the single owner of that
 * vocabulary so the three states feel intentional and consistent across the
 * dashboard, fleet, validate, and advisor surfaces. They are presentation-only
 * — callers own the data and pass labels via the typed strings registry.
 */

/**
 * A shimmering placeholder block used to build loading skeletons. The pulse is
 * pure CSS (Tailwind `animate-pulse`) so it adds no JS and respects
 * `prefers-reduced-motion` via the design tokens.
 */
export function Skeleton({ className }: { className?: string }) {
  return (
    <div
      aria-hidden="true"
      className={cn("animate-pulse rounded-control bg-app-surface-muted", className)}
    />
  );
}

/**
 * Loading state. Defaults to a labelled spinner; pass `skeleton` to render a
 * bespoke skeleton tree instead (preferred when the eventual layout is known,
 * so the page doesn't jump). The status role + label keep it announced to
 * assistive tech.
 */
export function LoadingState({
  label,
  title,
  skeleton,
  className,
  testId = selectors.state.loading,
}: {
  /** Visible/announced label; defaults to the shared "Loading…" copy. */
  label?: string;
  /** Optional heading shown above the spinner. */
  title?: string;
  /** A bespoke skeleton tree to render in place of the spinner. */
  skeleton?: ReactNode;
  className?: string;
  testId?: string;
}) {
  const { t } = useTranslation();
  const resolvedLabel = label ?? t(strings.common.loading);

  if (skeleton) {
    return (
      <div
        data-testid={testId}
        role="status"
        aria-busy="true"
        aria-label={resolvedLabel}
        className={cn("flex flex-col gap-3", className)}
      >
        {skeleton}
      </div>
    );
  }

  return (
    <div
      data-testid={testId}
      role="status"
      aria-busy="true"
      className={cn(
        "flex flex-col items-center justify-center gap-2 rounded-panel border border-app-border bg-app-surface p-8 text-center",
        className,
      )}
    >
      <Loader2 aria-hidden="true" className="h-6 w-6 animate-spin text-app-primary" />
      {title && <p className="text-sm font-medium text-app-foreground">{title}</p>}
      <p className="text-sm text-app-muted-foreground">{resolvedLabel}</p>
    </div>
  );
}

/**
 * Error state. Actionable by default — when `onRetry` is supplied it renders a
 * retry button so the user can recover without reloading. Falls back to the
 * shared error heading when no `title` is given.
 */
export function ErrorState({
  message,
  title,
  onRetry,
  retryLabel,
  retrying,
  className,
  testId = selectors.state.error,
}: {
  message: string;
  title?: string;
  onRetry?: () => void;
  retryLabel?: string;
  retrying?: boolean;
  className?: string;
  testId?: string;
}) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={testId}
      role="alert"
      className={cn(
        "flex flex-col items-start gap-3 rounded-panel border border-app-danger/40 bg-app-danger/5 p-4",
        className,
      )}
    >
      <div className="flex items-start gap-2">
        <AlertTriangle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-app-danger" />
        <div className="flex flex-col gap-1">
          <p className="text-sm font-semibold text-app-danger">
            {title ?? t(strings.common.errorTitle)}
          </p>
          <p className="text-sm text-app-foreground">{message}</p>
        </div>
      </div>
      {onRetry && (
        <Button
          data-testid={selectors.state.errorRetry}
          variant="outline"
          size="sm"
          onClick={onRetry}
          disabled={retrying}
        >
          <Loader2
            aria-hidden="true"
            className={cn("me-1 h-4 w-4", retrying ? "animate-spin" : "hidden")}
          />
          {retryLabel ?? t(strings.common.retry)}
        </Button>
      )}
    </div>
  );
}

/**
 * Empty state. First-run / no-data guidance with an optional primary call to
 * action. `icon` defaults to an inbox; pass a domain-specific Lucide icon to
 * reinforce context. The action can be either a click handler (button) or
 * `actionSlot` for a router `<Link>`.
 */
export function EmptyState({
  title,
  message,
  icon: Icon = Inbox,
  actionLabel,
  onAction,
  actionSlot,
  className,
  testId = selectors.state.empty,
}: {
  title?: string;
  message: string;
  icon?: LucideIcon;
  actionLabel?: string;
  onAction?: () => void;
  actionSlot?: ReactNode;
  className?: string;
  testId?: string;
}) {
  return (
    <div
      data-testid={testId}
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-panel border border-dashed border-app-border bg-app-surface p-8 text-center",
        className,
      )}
    >
      <Icon aria-hidden="true" className="h-8 w-8 text-app-muted-foreground" />
      {title && <p className="text-base font-medium text-app-foreground">{title}</p>}
      <p className="max-w-prose text-sm text-app-muted-foreground">{message}</p>
      {actionSlot}
      {!actionSlot && actionLabel && onAction && (
        <Button data-testid={selectors.state.emptyAction} size="sm" onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </div>
  );
}

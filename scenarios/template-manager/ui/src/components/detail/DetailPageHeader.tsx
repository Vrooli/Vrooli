import { ArrowLeft } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import { StatusBadge } from "../ui/status-badge";
import type { Tone } from "../../lib/templateLabels";

export interface DetailPageHeaderProps {
  /** Route the back affordance navigates to (usually the parent list/overview). */
  backTo: string;
  /** Accessible label for the back link. */
  backLabel: string;
  /** Page title. Rendered as an `<h2>` carrying `titleId`. */
  title: string;
  /** Id wired to the page's `aria-labelledby`. */
  titleId: string;
  /** Optional status pill shown next to the title. */
  status?: { label: string; tone: Tone };
  /** Secondary line under the title (ids, counts, timestamps). */
  subtitle?: ReactNode;
  /**
   * At most one primary action, per the fleet detail-header contract. Read-only
   * views omit it. Menu/ellipsis actions are intentionally absent here.
   */
  primaryAction?: ReactNode;
  testId?: string;
}

/**
 * Detail-page header aligned with the fleet detail-header contract: a back
 * affordance, the title, an optional status pill, a subtitle line, and at most
 * one primary action. The entity icon deliberately lives in the body's first
 * "Overview" `DetailSection`, not here.
 */
export function DetailPageHeader({
  backTo,
  backLabel,
  title,
  titleId,
  status,
  subtitle,
  primaryAction,
  testId,
}: DetailPageHeaderProps) {
  return (
    <header data-testid={testId} className="flex flex-col gap-3">
      <Link
        to={backTo}
        aria-label={backLabel}
        className="inline-flex min-h-11 w-fit items-center gap-1.5 rounded-control px-2 text-sm font-medium text-app-muted-foreground transition-colors hover:text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
      >
        <ArrowLeft aria-hidden className="h-4 w-4" />
        {backLabel}
      </Link>
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-2">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h2 id={titleId} className="min-w-0 break-words text-2xl font-semibold">
              {title}
            </h2>
            {status && <StatusBadge tone={status.tone}>{status.label}</StatusBadge>}
          </div>
          {subtitle && (
            <p className="min-w-0 break-words text-sm text-app-muted-foreground">{subtitle}</p>
          )}
        </div>
        {primaryAction && <div className="shrink-0">{primaryAction}</div>}
      </div>
    </header>
  );
}

import { useState, type ReactNode } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import { EmptyState } from "@vrooli/react-component-library/EmptyState";
import { cn } from "../lib/classnames";

export interface PaneEmptyStateProps {
  /** The stable id used by tests and callers for this state. */
  testId: string;
  title: string;
  description?: string;
  icon?: ReactNode;
  /** A single note that is always visible, above the actions. */
  footnote?: string;
  /** Stable id for the footnote, when a caller or test addresses it. */
  footnoteTestId?: string;
  /** Operator detail kept behind a disclosure. */
  detail?: string;
  /** Labels for the disclosure control; hosts can localize them. */
  detailShowLabel?: string;
  detailHideLabel?: string;
  /** Buttons, typically a primary repair action and a secondary explanation. */
  actions?: ReactNode;
  tone?: "neutral" | "alert";
  /** Announce the state to assistive technology as an async status. */
  live?: boolean;
  className?: string;
}

/**
 * One empty/error moment for a pane.
 *
 * Browser, Messages and settings all use this so "there is nothing here"
 * reads the same everywhere: a soft card, a rounded icon badge, one title,
 * one plain sentence, and at most the actions that actually help. The layout
 * is the library's `EmptyState`; this wrapper only supplies the pane-level
 * surface and the optional operator detail.
 */
export function PaneEmptyState({
  testId,
  title,
  description,
  icon,
  footnote,
  footnoteTestId,
  detail,
  detailShowLabel = "Show details",
  detailHideLabel = "Hide details",
  actions,
  tone = "neutral",
  live = false,
  className,
}: PaneEmptyStateProps) {
  const [detailOpen, setDetailOpen] = useState(false);

  return (
    <div
      data-testid={testId}
      data-tone={tone}
      className={cn("flex h-full w-full items-center justify-center p-4 sm:p-6", className)}
    >
      <div
        className={cn(
          "w-full max-w-md overflow-hidden rounded-2xl border shadow-sm",
          tone === "alert"
            ? "border-red-500/25 bg-red-500/[0.04]"
            : "bg-wc-surface-raised/60",
        )}
        style={tone === "alert"
          ? undefined
          : { borderColor: "color-mix(in srgb, var(--color-border) 45%, transparent)" }}
      >
        {live && (
          <p role="status" className="sr-only">
            {title}{description ? `. ${description}` : ""}
          </p>
        )}
        <EmptyState
          density="compact"
          icon={icon}
          title={title}
          description={description}
          action={actions}
          className="px-2 py-2"
        />
        {footnote && (
          <p
            data-testid={footnoteTestId}
            className="mx-4 mb-4 rounded-lg border bg-wc-surface-input/30 px-3 py-2 text-center text-[11px] leading-relaxed text-wc-text-faint"
            style={{ borderColor: "color-mix(in srgb, var(--color-border) 35%, transparent)" }}
          >
            {footnote}
          </p>
        )}
        {detail && (
          <div className="mx-4 mb-4">
            <button
              type="button"
              onClick={() => { setDetailOpen((open) => !open); }}
              aria-expanded={detailOpen}
              data-testid={`${testId}-details-toggle`}
              className="inline-flex items-center gap-1 text-[11px] font-medium text-wc-text-muted underline-offset-2 transition-colors hover:text-wc-text-secondary hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
            >
              {detailOpen ? <ChevronUp className="h-3 w-3" aria-hidden /> : <ChevronDown className="h-3 w-3" aria-hidden />}
              {detailOpen ? detailHideLabel : detailShowLabel}
            </button>
            {detailOpen && (
              <pre
                data-testid={`${testId}-details`}
                className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md border border-wc-default bg-wc-surface-input p-2 text-start text-[11px] leading-relaxed text-wc-text-muted"
              >
                {detail}
              </pre>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default PaneEmptyState;

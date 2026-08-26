/**
 * @libraryId react-component-library:StatusIndicator
 * @displayName StatusIndicator
 * @description A compact live status signal with explicit semantic states and accessible status output.
 * @version 1.0.3
 * @tags ["primitive","feedback","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource feedback.status-indicator */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { CSSProperties } from "react";
export type StatusCertainty = "scheduled" | "estimated" | "predicted" | "observed" | "confirmed";
export type StatusUrgency = "ambient" | "informational" | "actionable" | "critical";
export const StatusIndicator = withClassName(function StatusIndicator({
  status = "idle",
  label,
  certainty = "observed",
  urgency = "ambient",
}: {
  status?: "idle" | "pending" | "success" | "error" | "offline";
  label?: string;
  certainty?: StatusCertainty;
  urgency?: StatusUrgency;
}) {
  const tone =
    status === "success"
      ? "var(--color-success)"
      : status === "error"
        ? "var(--color-danger)"
        : status === "pending"
          ? "var(--color-warning)"
          : "var(--color-muted-foreground)";
  return (
    <>
      <style
        data-rcl-status-indicator-styles
        dangerouslySetInnerHTML={{
          __html: `
        [data-rcl-status-indicator] { display: inline-flex; align-items: center; gap: var(--space-2xs); border: var(--border-hairline) solid var(--rcl-status-tone); border-radius: var(--radius-pill); color: var(--rcl-status-tone); padding: var(--space-3xs) var(--space-xs); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); font-weight: 650; }
        [data-rcl-status-dot] { width: var(--space-2xs); height: var(--space-2xs); flex: 0 0 auto; border-radius: 50%; background: var(--rcl-status-tone); }
        [data-rcl-status-indicator][data-status="pending"] [data-rcl-status-dot] { animation: rcl-status-pulse var(--dur-deliberate) var(--ease-standard) infinite; }
        @keyframes rcl-status-pulse { 0%, 100% { opacity: .55; transform: scale(.84); } 50% { opacity: 1; transform: scale(1); } }
        @media (prefers-reduced-motion: reduce) { [data-rcl-status-indicator] [data-rcl-status-dot] { animation: none; } }
      `,
        }}
      />
      <span
        data-testid="primitives.status-indicator"
        role="status"
        data-rcl-status-indicator="true"
        data-status={status}
        data-certainty={certainty}
        data-urgency={urgency}
        style={{ "--rcl-status-tone": tone } as CSSProperties}
      >
        <span aria-hidden data-rcl-status-dot="true" />
        {label ?? status}
      </span>
    </>
  );
});

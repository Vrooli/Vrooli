/**
 * @libraryId react-component-library:StatusBadge
 * @version 1.2.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { HTMLAttributes, ReactNode } from "react";
import { statusBadgeStyles } from "./styles";

export type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

export interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
  tone?: StatusTone;
}

export function StatusBadge({
  children,
  className,
  tone = "neutral",
  ...props
}: StatusBadgeProps) {
  return (
    <>
      <style
        data-rcl-status-badge-styles
        dangerouslySetInnerHTML={{ __html: statusBadgeStyles }}
      />
      <span
        {...props}
        className={className}
        data-rcl-status-badge
        data-tone={tone}
      >
        <span data-rcl-status-badge-indicator aria-hidden="true" />
        <span data-rcl-status-badge-label>{children}</span>
      </span>
    </>
  );
}

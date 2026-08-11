/**
 * @vrooliComponentSource react-component-library:StatusBadge
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption 4cd805e8-3c46-4750-b4eb-8df2b86e4028
 * @vrooliComponentAppliedAt 2026-08-11T00:48:00Z
 * @vrooliComponentSourceSha256 11733a6e285ed43571272759edc5cf45cdf020db12ee4358de5af5fb94fe90aa
 * @vrooliComponentDriftHash 1e4e7a57b0af5e79e927a27b2c9cf3f23f1be9ab007ca1b15ddc702717f1f3a7
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { HTMLAttributes, ReactNode } from "react";
import { statusBadgeStyles } from "./styles";

export type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

export interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
  tone?: StatusTone;
}

export function StatusBadge({ children, className, tone = "neutral", ...props }: StatusBadgeProps) {
  return (
    <>
      <style data-rcl-status-badge-styles dangerouslySetInnerHTML={{ __html: statusBadgeStyles }} />
      <span {...props} className={className} data-rcl-status-badge data-tone={tone}>
        <span data-rcl-status-badge-indicator aria-hidden="true" />
        <span data-rcl-status-badge-label>{children}</span>
      </span>
    </>
  );
}

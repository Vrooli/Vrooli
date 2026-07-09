/**
 * @vrooliComponentSource react-component-library:StatusBadge
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption template:react-vite:status-badge
 * @vrooliComponentAppliedAt 2026-07-07T00:00:00Z
 * @vrooliComponentSourceSha256 e05e4cdb04ee8b69ef249159a4d9dc8d0d7322190622f8af40ec22247bea5024
 * @vrooliComponentDriftHash e05e4cdb04ee8b69ef249159a4d9dc8d0d7322190622f8af40ec22247bea5024
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { HTMLAttributes, ReactNode } from "react";

type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

export interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
  tone?: StatusTone;
}

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

const toneClasses: Record<StatusTone, string> = {
  neutral: "border-app-border bg-app-surface-muted text-app-muted-foreground",
  success: "border-app-primary/30 bg-app-primary/10 text-app-primary",
  warning: "border-app-border bg-app-surface-muted text-app-foreground",
  danger: "border-app-danger/30 bg-app-danger/10 text-app-danger",
  info: "border-app-primary/30 bg-app-primary/10 text-app-primary",
};

export function StatusBadge({ children, className, tone = "neutral", ...props }: StatusBadgeProps) {
  return (
    <span
      className={joinClasses(
        "inline-flex min-h-7 max-w-full items-center rounded-pill border px-2.5 text-xs font-semibold leading-none",
        toneClasses[tone],
        className,
      )}
      {...props}
    >
      <span className="truncate whitespace-nowrap">{children}</span>
    </span>
  );
}

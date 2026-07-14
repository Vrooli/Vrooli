/**
 * @vrooliComponentSource react-component-library:StatusBadge
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption template:react-vite:status-badge
 * @vrooliComponentAppliedAt 2026-07-14T20:14:31Z
 * @vrooliComponentSourceSha256 5ca13bc8a8e6fe98519a4ee1fea0de5e622a47ae9cf664d4901177660edd22b8
 * @vrooliComponentDriftHash 5ca13bc8a8e6fe98519a4ee1fea0de5e622a47ae9cf664d4901177660edd22b8
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { HTMLAttributes, ReactNode } from "react";

type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

export interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
  tone?: StatusTone;
}

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const toneClasses: Record<StatusTone, string> = {
  neutral: "border-app-border bg-app-surface-muted text-app-muted-foreground",
  success: "border-app-success/30 bg-app-success/10 text-app-success",
  warning: "border-app-warning/30 bg-app-warning/10 text-app-warning",
  danger: "border-app-danger/30 bg-app-danger/10 text-app-danger",
  info: "border-app-info/30 bg-app-info/10 text-app-info",
};

export function StatusBadge({ children, className, tone = "neutral", ...props }: StatusBadgeProps) {
  return (
    <span
      className={cn(
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


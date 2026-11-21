import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export type NoticeTone = "success" | "warning" | "danger" | "info";

const BAR: Record<NoticeTone, string> = {
  success: "border-s-app-success",
  warning: "border-s-app-warning",
  danger: "border-s-app-danger",
  info: "border-s-app-primary",
};

/**
 * A quiet inline message beside the thing it is about. Problems that need
 * attention announce themselves; everything else is read when the reader gets
 * there.
 */
export function Notice({ tone = "info", title, children, className, testId }: {
  tone?: NoticeTone;
  title?: string;
  children?: ReactNode;
  className?: string;
  testId?: string;
}) {
  return (
    <div
      role={tone === "danger" ? "alert" : "status"}
      data-testid={testId}
      className={cn("flex flex-col gap-1 rounded-panel border border-app-border border-s-4 bg-app-surface px-4 py-3", BAR[tone], className)}
    >
      {title && <p className="text-sm font-semibold">{title}</p>}
      {children && <div className="text-sm text-app-muted-foreground">{children}</div>}
    </div>
  );
}

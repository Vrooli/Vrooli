import { forwardRef, type ComponentProps, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";

import { cn } from "../lib/utils";

/** A grouped list: rows share one rounded surface and are divided by hairlines. */
export function ListGroup({ className, ...props }: ComponentProps<"ul">) {
  return <ul className={cn("divide-y divide-app-border overflow-hidden rounded-panel border border-app-border bg-app-surface", className)} {...props} />;
}

/** The urgency of a row, drawn as a bar on its leading edge. Normal rows carry none. */
export type RowTone = "normal" | "attention" | "critical";

const TONE_BAR: Record<RowTone, string> = {
  normal: "",
  attention: "before:absolute before:inset-y-0 before:start-0 before:w-1 before:bg-app-warning",
  critical: "before:absolute before:inset-y-0 before:start-0 before:w-1 before:bg-app-danger",
};

export function ListItem({ tone = "normal", className, ...props }: ComponentProps<"li"> & { tone?: RowTone }) {
  return <li data-tone={tone} className={cn("relative", TONE_BAR[tone], className)} {...props} />;
}

export interface ListRowProps extends Omit<ComponentProps<typeof Link>, "children"> {
  children: ReactNode;
  /** Shown after the body and before the chevron, for a status or a count. */
  trailing?: ReactNode;
}

/** A whole-row link. The chevron turns around in right-to-left languages. */
export const ListRow = forwardRef<HTMLAnchorElement, ListRowProps>(function ListRow({ children, trailing, className, ...props }, ref) {
  return (
    <Link
      ref={ref}
      className={cn(
        "flex min-h-14 items-start gap-3 px-4 py-3 text-app-foreground transition-colors duration-quick hover:bg-app-surface-muted active:bg-app-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2",
        className,
      )}
      {...props}
    >
      <div className="min-w-0 flex-1">{children}</div>
      {trailing}
      <ChevronRight size={18} className="mt-0.5 shrink-0 text-app-muted-foreground rtl:rotate-180" aria-hidden="true" />
    </Link>
  );
});

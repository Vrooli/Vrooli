/**
 * DetailSection
 *
 * Lightweight section wrapper for entity detail pages. Replaces the
 * Card-inside-tab pattern with flat sections separated by thin dividers,
 * eliminating compounding padding on mobile.
 */

import { type ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import { cn } from "../../lib/utils";

export interface DetailSectionProps {
  /** Section heading text. */
  title: string;
  /** Optional icon rendered before the title. */
  icon?: LucideIcon;
  /** Optional action slot (e.g., edit button) rendered at the right of the heading. */
  action?: ReactNode;
  /** Section content. */
  children: ReactNode;
  /** Hide the top divider (for the first section in a group). */
  hideDivider?: boolean;
  className?: string;
  "data-testid"?: string;
}

export function DetailSection({
  title,
  icon: Icon,
  action,
  children,
  hideDivider,
  className,
  "data-testid": testId,
}: DetailSectionProps) {
  return (
    <section className={cn(!hideDivider && "mt-4 border-t border-slate-800 pt-4", hideDivider && "pt-1", className)} data-testid={testId}>
      <div className="flex items-center gap-2 pb-3">
        {Icon && <Icon className="h-4 w-4 text-slate-400" />}
        <h2 className="text-base font-semibold text-slate-100">{title}</h2>
        {action && <div className="ml-auto">{action}</div>}
      </div>
      <div className="pb-2">{children}</div>
    </section>
  );
}

/**
 * StatusChip
 *
 * Small labeled pill used to convey a single piece of status-like information:
 * a lifecycle status ("Completed"), an agent activity ("Workshopping"), or a
 * pending-input reason ("3 decisions"). Shared between the Initiative Details
 * row chip and the dependency chip list on the Backlog Details page.
 *
 * Visual vocabulary:
 *   border + tinted background + colored label text, optionally with a
 *   leading colored dot and a "busy" pulse animation.
 *
 * Keep this primitive stateless. Higher-level components compose it with
 * popovers / click-throughs as needed.
 */

import { cn } from "../../lib/utils";

export interface StatusChipColors {
  /** Tailwind background class (e.g. "bg-blue-500/20"). */
  background: string;
  /** Tailwind border class (e.g. "border-blue-400/80"). Optional — when absent no border is drawn. */
  border?: string;
  /** Tailwind text color class (e.g. "text-blue-300"). */
  text: string;
  /** Tailwind background class for the leading dot, if rendered (e.g. "bg-blue-500"). */
  dot?: string;
}

export interface StatusChipProps {
  label: string;
  colors: StatusChipColors;
  /** When true, renders an animated "busy" ping over the leading dot. */
  pulse?: boolean;
  /** When true, renders a small solid dot before the label (uses `colors.dot` if set, else falls back to `colors.text`). */
  leadingDot?: boolean;
  /** Optional native title attribute (tooltip). */
  title?: string;
  /** Optional click handler — if provided, renders as a <button>. */
  onClick?: (event: React.MouseEvent<HTMLElement>) => void;
  className?: string;
  "data-testid"?: string;
}

export function StatusChip({
  label,
  colors,
  pulse,
  leadingDot,
  title,
  onClick,
  className,
  "data-testid": testId,
}: StatusChipProps) {
  const chipClass = cn(
    "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[10px] font-medium leading-tight",
    colors.border ? "border" : "",
    colors.background,
    colors.border ?? "",
    colors.text,
    onClick ? "cursor-pointer transition-colors hover:brightness-125" : "",
    className,
  );

  const dotColor = colors.dot ?? colors.text;

  const content = (
    <>
      {leadingDot && (
        pulse ? (
          <span className="relative flex h-1.5 w-1.5 shrink-0" aria-hidden>
            <span className={cn("absolute inline-flex h-full w-full animate-ping rounded-full opacity-75", dotColor)} />
            <span className={cn("relative inline-flex h-1.5 w-1.5 rounded-full", dotColor)} />
          </span>
        ) : (
          <span className={cn("inline-block h-1.5 w-1.5 shrink-0 rounded-full", dotColor)} aria-hidden />
        )
      )}
      <span>{label}</span>
    </>
  );

  if (onClick) {
    return (
      <button type="button" onClick={onClick} title={title} className={chipClass} data-testid={testId}>
        {content}
      </button>
    );
  }

  return (
    <span title={title} className={chipClass} data-testid={testId}>
      {content}
    </span>
  );
}

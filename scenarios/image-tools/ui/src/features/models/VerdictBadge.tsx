import { useTranslation } from "../../i18n";
import type { Verdict } from "../../api/models";
import { cn } from "../../lib/utils";
import { gradeView } from "./verdictPresentation";

export interface VerdictBadgeProps {
  verdict: Verdict | undefined;
  testId?: string;
  className?: string;
}

/**
 * VerdictBadge is the grade of a model (for one operation, or rolled up) as a
 * chip. The one-line reason is the tooltip and the accessible description, so a
 * phone user can long-press and a screen reader hears why.
 */
export function VerdictBadge({ verdict, testId, className }: VerdictBadgeProps) {
  const { t } = useTranslation();
  const view = gradeView(verdict?.grade);
  return (
    <span
      data-testid={testId}
      title={verdict?.reason || undefined}
      className={cn("inline-flex shrink-0 items-center rounded px-1.5 py-0.5 text-[11px] font-medium", view.className, className)}
    >
      {t(view.label)}
    </span>
  );
}

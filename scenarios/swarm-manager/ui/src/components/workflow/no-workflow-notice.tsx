/**
 * NoWorkflowNotice
 *
 * Subtle affordance shown when the canonical workflow projection reports no
 * workflow document for the target (found=false). Post-migration this simply
 * means the item has never run an operation — a workflow document is created
 * on the first operation start. Action availability then comes from the
 * client-side default funnel (lib/backlog-queue-utils.ts).
 */

import { History } from "lucide-react";
import { cn } from "../../lib/utils";
import type { WorkflowProjection } from "../../types/agent-operations";

export function NoWorkflowNotice({
  projection,
  className,
}: {
  /** Resolved projection; render nothing while loading or when a workflow exists. */
  projection: WorkflowProjection | undefined | null;
  className?: string;
}) {
  if (!projection || projection.found) return null;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full border border-slate-700/70 bg-slate-800/50 px-2 py-0.5 text-[10px] font-medium text-slate-500",
        className,
      )}
      title="No workflow yet: this item hasn't run an operation. A workflow document is created on the first operation start; until then actions use the default client funnel."
      data-testid="no-workflow-notice"
    >
      <History className="h-3 w-3" aria-hidden />
      No workflow yet
    </span>
  );
}

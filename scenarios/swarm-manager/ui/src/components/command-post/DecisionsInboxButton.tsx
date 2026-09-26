/**
 * DecisionsInboxButton — the one affordance that opens the decision drawer.
 *
 * The sidebar header and the Plan board's Next column both offer this, and
 * they used to disagree on icon, colour, and count. One component keeps them
 * identical, and the count comes from usePendingDecisionCount so it always
 * matches what the drawer actually contains.
 */

import { Inbox } from "lucide-react";
import { cn } from "../../lib/utils";
import { usePendingDecisionCount } from "../../hooks/usePendingDecisionCount";

export interface DecisionsInboxButtonProps {
  onOpen: () => void;
  /** Hide entirely when nothing is pending (the Plan board's column header). */
  hideWhenEmpty?: boolean;
  className?: string;
  testId?: string;
}

export function DecisionsInboxButton({
  onOpen,
  hideWhenEmpty = false,
  className,
  testId = "decisions-inbox-button",
}: DecisionsInboxButtonProps) {
  const count = usePendingDecisionCount();

  if (hideWhenEmpty && count === 0) return null;

  const label = count === 1
    ? "Open decisions — 1 pending"
    : `Open decisions — ${count} pending`;

  return (
    <button
      type="button"
      onClick={onOpen}
      className={cn(
        "flex items-center gap-1 rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200",
        className,
      )}
      aria-label={label}
      title={label}
      data-testid={testId}
    >
      <Inbox className="h-4 w-4" aria-hidden />
      {count > 0 && (
        <span
          className="rounded-full bg-cyan-500/20 px-1.5 py-0.5 text-xs tabular-nums text-cyan-200"
          data-testid="decisions-inbox-count"
        >
          {count}
        </span>
      )}
    </button>
  );
}

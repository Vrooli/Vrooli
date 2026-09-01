import type { ReactNode } from "react";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";

/**
 * One card on a fleet shelf.
 *
 * The anatomy is fixed, and that is the point. Cards sit side by side in a
 * horizontal rail, which is scanned across one line at a time — silhouettes,
 * then names, then states — so a card that is 180px taller than its neighbour
 * breaks the scan and drags the whole shelf's height with it.
 *
 * Before this, the card grew with whatever the machine had wrong: one line per
 * drift item, one full-width button per missing tool. A healthy machine got a
 * short card and a broken one got a wall, which is exactly backwards from what
 * a shelf can absorb.
 *
 * So every slot below is always rendered, in this order and no other:
 *
 *   silhouette   fixed height
 *   title row    name + one status badge
 *   meta         exactly one line, truncated
 *   body         exactly one line, truncated — the permission sentence
 *   state        pinned to the bottom; "Nothing to fix" is a line too
 *   actions      one primary, one secondary, one overflow
 *
 * The state slot is why `state` takes a node rather than a string: the count
 * of everything wrong belongs on the card, and the detail belongs in the
 * machine's own Configuration tab, one tap away.
 */

export interface FleetCardProps {
  testId?: string;
  title: string;
  meta?: string;
  /** Reachability, as a badge. One fact, one place. */
  status?: string;
  statusTone?: StatusTone;
  silhouette: ReactNode;
  /** The single-line sentence under the meta — what this thing may do. */
  children?: ReactNode;
  /** The bottom-pinned state row. Always occupied, so cards stay level. */
  state?: ReactNode;
  actions?: ReactNode;
}

export function FleetCard({
  testId,
  title,
  meta,
  status,
  statusTone = "neutral",
  silhouette,
  children,
  state,
  actions,
}: FleetCardProps) {
  return (
    <article
      data-testid={testId}
      className="flex w-[268px] shrink-0 flex-col overflow-hidden rounded-xl border border-wc-default bg-wc-surface-input p-4 shadow-sm transition hover:border-wc-accent/50"
    >
      <div className="relative h-24 w-full shrink-0 overflow-hidden rounded-lg bg-wc-surface-base">
        {silhouette}
      </div>

      <div className="mt-3 flex items-start justify-between gap-2">
        <h3 className="min-w-0 flex-1 truncate text-sm font-medium text-wc-text-primary">{title}</h3>
        {status && (
          <span className="shrink-0">
            <StatusBadge tone={statusTone}>{status}</StatusBadge>
          </span>
        )}
      </div>

      {/* Both single-line slots reserve their height whether or not they have
          content, which is what keeps a card with no permission sentence level
          with one that has it. */}
      <p className="mt-1 h-4 truncate text-xs text-wc-text-faint">{meta}</p>
      <div className="mt-2 h-8 overflow-hidden text-xs leading-4 text-wc-text-secondary">{children}</div>

      <div className="mt-auto pt-3">{state}</div>
      {actions && <div className="mt-3 flex items-center gap-2">{actions}</div>}
    </article>
  );
}

export default FleetCard;

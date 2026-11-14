/**
 * WaveGroup — one collapsible card group inside a plan column (a Later
 * nearest-blocker group, the Next gates band, etc.).
 */

import { useEffect, useState } from "react";
import { ChevronRight } from "lucide-react";
import { cn } from "../../../lib/utils";
import type { PlanCardGroupData } from "../types";
import { PlanCardView } from "./PlanCardView";

const BLOCKER_ICON: Record<string, string> = {
  gate: "⚡",
  items: "⛓",
  cycle: "⟳",
};

export interface WaveGroupProps {
  group: PlanCardGroupData;
  showWaves?: boolean;
  defaultCollapsed?: boolean;
  /** Card ids to render dimmed (snoozed cards with show-snoozed on). */
  dimmedIds?: Set<string>;
  /** Card id to render with the deep-link emphasis ring. */
  highlightedId?: string | null;
}

export function WaveGroup({
  group,
  showWaves = false,
  defaultCollapsed = false,
  dimmedIds,
  highlightedId,
}: WaveGroupProps) {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);
  const icon = BLOCKER_ICON[group.blockerKind];

  // A deep-linked card must be visible to receive its emphasis ring.
  useEffect(() => {
    if (highlightedId && group.cards.some((card) => card.id === highlightedId)) {
      setCollapsed(false);
    }
  }, [group.cards, highlightedId]);

  return (
    <div data-testid={`plan-group-${group.id}`}>
      <button
        type="button"
        onClick={() => setCollapsed((prev) => !prev)}
        className="flex w-full items-center gap-1.5 rounded px-1 py-1 text-left text-xs font-medium text-slate-400 transition-colors hover:text-slate-200"
        data-testid="plan-group-toggle"
        aria-expanded={!collapsed}
      >
        <ChevronRight
          className={cn("h-3.5 w-3.5 shrink-0 transition-transform", !collapsed && "rotate-90")}
        />
        {icon ? (
          <span aria-hidden className={cn(group.blockerKind === "cycle" && "text-rose-400")}>
            {icon}
          </span>
        ) : null}
        <span className="min-w-0 flex-1 truncate">{group.label}</span>
        <span className="shrink-0 text-slate-600">{group.cards.length}</span>
      </button>
      {!collapsed && (
        <div className="mt-1 space-y-1.5">
          {group.cards.map((card) => (
            <PlanCardView
              key={card.id}
              card={card}
              showWave={showWaves}
              dimmed={dimmedIds?.has(card.id) ?? false}
              highlighted={card.id === highlightedId}
            />
          ))}
        </div>
      )}
    </div>
  );
}

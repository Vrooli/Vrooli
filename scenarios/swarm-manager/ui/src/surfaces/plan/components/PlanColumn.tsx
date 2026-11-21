/**
 * PlanColumn — one scrollable board column of card groups. Column semantics
 * (which groups arrive, wave badges, horizon rollups) are decided by
 * PlanBoard; this component owns only the frame.
 */

import type { ReactNode } from "react";
import type { PlanCardGroupData } from "../types";
import { ColumnHeader } from "./ColumnHeader";
import { WaveGroup } from "./WaveGroup";

export interface PlanColumnProps {
  title: string;
  count: number;
  subtitle?: ReactNode;
  headerAction?: ReactNode;
  groups: PlanCardGroupData[];
  showWaves?: boolean;
  /** Card ids to render dimmed (snoozed cards with show-snoozed on). */
  dimmedIds?: Set<string>;
  /** Card id to render with the deep-link emphasis ring. */
  highlightedId?: string | null;
  /** Rendered after the groups (e.g. the beyond-horizon rollup). */
  footer?: ReactNode;
  emptyState: ReactNode;
  testId: string;
}

export function PlanColumn({
  title,
  count,
  subtitle,
  headerAction,
  groups,
  showWaves = false,
  dimmedIds,
  highlightedId,
  footer,
  emptyState,
  testId,
}: PlanColumnProps) {
  const isEmpty = groups.length === 0 && !footer;

  return (
    <section
      className="flex h-full w-72 shrink-0 flex-col bg-slate-950/60 md:w-80"
      data-testid={testId}
    >
      <ColumnHeader
        title={title}
        count={count}
        subtitle={subtitle}
        action={headerAction}
        testId={`${testId}-header`}
      />
      <div className="flex-1 space-y-3 overflow-y-auto p-2">
        {isEmpty
          ? emptyState
          : (
            <>
              {groups.map((group) => (
                <WaveGroup key={group.id} group={group} showWaves={showWaves} dimmedIds={dimmedIds} highlightedId={highlightedId} />
              ))}
              {footer}
            </>
          )}
      </div>
    </section>
  );
}

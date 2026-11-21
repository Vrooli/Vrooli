import type { ReactNode } from "react";
import { EmptyState } from "./EmptyState";
import { VerdictCell, type VerdictKind } from "./VerdictCell";
import { cn } from "../../lib/utils";

export interface VerdictGridRow {
  /** Stable row id. */
  id: string;
  /** Row label (left column). */
  label: ReactNode;
  /** Sub-label (e.g. tool name, version). */
  subLabel?: ReactNode;
  /** Verdict kind. */
  kind: VerdictKind;
  /** Optional metric (e.g. duration). */
  metric?: ReactNode;
}

export interface VerdictGridProps {
  /** Optional table heading. */
  caption?: ReactNode;
  rows: readonly VerdictGridRow[];
  /** Called with the row id when a row is clicked. */
  onRowClick?: (rowId: string) => void;
  /** Rendered when `rows` is empty. */
  emptyState?: ReactNode;
  className?: string;
  testId?: string;
}

/**
 * Tabular verdict grid. Desktop: row label / verdict cell / metric in a
 * 3-column grid. Mobile (`<sm`): stacks each row into a card.
 *
 * Surfaces compose this with whatever data they have; the grid does not
 * own data fetching.
 */
export function VerdictGrid({
  caption,
  rows,
  onRowClick,
  emptyState,
  className,
  testId,
}: VerdictGridProps) {
  if (rows.length === 0) {
    if (emptyState) return <>{emptyState}</>;
    return (
      <EmptyState
        testId={testId ? `${testId}-empty` : undefined}
        title={caption}
      />
    );
  }

  return (
    <div data-testid={testId} className={cn("flex flex-col gap-1", className)}>
      {caption ? (
        <p className="px-2 text-[10px] uppercase tracking-wide text-app-muted-foreground">
          {caption}
        </p>
      ) : null}
      <ul className="flex flex-col gap-1">
        {rows.map((row) => (
          <li
            key={row.id}
            className="grid grid-cols-1 items-center gap-1 rounded-control border border-app-border bg-app-surface p-2 sm:grid-cols-[1fr_auto] sm:gap-3"
          >
            <div className="min-w-0">
              <p className="truncate text-sm text-app-foreground">{row.label}</p>
              {row.subLabel ? (
                <p className="truncate text-xs text-app-muted-foreground">{row.subLabel}</p>
              ) : null}
            </div>
            <VerdictCell
              kind={row.kind}
              metric={row.metric}
              testId={testId ? `${testId}-row-${row.id}` : undefined}
              onClick={onRowClick ? () => onRowClick(row.id) : undefined}
            />
          </li>
        ))}
      </ul>
    </div>
  );
}

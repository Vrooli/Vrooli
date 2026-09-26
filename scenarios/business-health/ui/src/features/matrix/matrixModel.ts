import type { MatrixRow } from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

/**
 * One operational-target group: the OT header plus the requirement rows that
 * trace to it. Pure view model derived from the flat matrix so the page can
 * render grouped sections and so grouping is unit-testable in isolation.
 */
export interface MatrixGroup {
  readonly otId: string;
  readonly otTitle: string;
  readonly otChecked: boolean;
  readonly otPriority: string;
  /** True when the OT has no requirement tracing to it (orphaned target). */
  readonly isOrphanTarget: boolean;
  /** Requirement rows tracing to this OT (may be empty for an orphan target). */
  readonly rows: MatrixRow[];
}

/**
 * The unlinked group carries requirements whose PRD reference did not resolve
 * (empty ot_id). Surfaced separately from real OT groups.
 */
export const UNLINKED_GROUP_ID = "__unlinked__";

const isRequirementRow = (row: MatrixRow): boolean => row.requirementId.trim() !== "";

/**
 * Group matrix rows by operational target, preserving first-seen order. Rows
 * with an empty ot_id collect under a single `UNLINKED_GROUP_ID` group; a group
 * that only ever sees an empty-requirement placeholder is flagged as an
 * orphaned target.
 */
export function groupMatrixRows(rows: readonly MatrixRow[]): MatrixGroup[] {
  const order: string[] = [];
  // Mutable during assembly; the return type re-narrows to the readonly view.
  const byKey = new Map<string, {
    otId: string;
    otTitle: string;
    otChecked: boolean;
    otPriority: string;
    isOrphanTarget: boolean;
    rows: MatrixRow[];
  }>();

  for (const row of rows) {
    const linked = row.otId.trim() !== "";
    const key = linked ? row.otId : UNLINKED_GROUP_ID;

    let group = byKey.get(key);
    if (!group) {
      group = {
        otId: linked ? row.otId : UNLINKED_GROUP_ID,
        otTitle: linked ? row.otTitle : "",
        otChecked: linked ? row.otChecked : false,
        otPriority: linked ? row.otPriority : "",
        isOrphanTarget: false,
        rows: [],
      };
      byKey.set(key, group);
      order.push(key);
    }

    if (isRequirementRow(row)) {
      group.rows.push(row);
    } else if (linked) {
      // OT placeholder row with no requirement — orphaned target.
      group.isOrphanTarget = true;
    }
  }

  return order.flatMap((key) => {
    const group = byKey.get(key);
    return group ? [group] : [];
  });
}

/** Count rows flagged as unproven claims across the whole matrix. */
export function countUnproven(rows: readonly MatrixRow[]): number {
  return rows.reduce((total, row) => (row.unproven ? total + 1 : total), 0);
}

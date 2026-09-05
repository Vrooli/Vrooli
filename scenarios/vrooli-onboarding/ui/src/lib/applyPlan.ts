/**
 * Apply-plan disclosure logic, shared by every surface that asks an operator to
 * consent to host changes.
 *
 * The plan is a desired-state list, not a change list: an item already present
 * on the host still appears. Rendering it flat therefore reads as if every
 * entry were a pending change, which is what an operator reported after being
 * shown 53 items on a host where none were pending. The split below is the
 * whole point of this module, and it mirrors the wizard CLI renderer so the two
 * surfaces disclose the same facts.
 */

export type ApplyItemState = "satisfied" | "pending" | "unknown";

export interface ApplyPlanItem {
  id: string;
  kind: string;
  name: string;
  required: boolean;
  privileged?: boolean;
  /** Absent when the API predates state reporting; never assume presence. */
  state?: string;
}

/**
 * An item with no reported state is unknown, never satisfied. A CLI or UI newer
 * than its API must not claim an item is already in place on the strength of a
 * missing field.
 */
export function itemState(item: ApplyPlanItem): ApplyItemState {
  return item.state === "satisfied" || item.state === "pending" ? item.state : "unknown";
}

export interface ApplyPlanSummary {
  total: number;
  pending: ApplyPlanItem[];
  satisfied: ApplyPlanItem[];
  unknown: ApplyPlanItem[];
  elevatedPending: number;
  elevatedTotal: number;
}

export function summarizeApplyPlan(items: ApplyPlanItem[]): ApplyPlanSummary {
  const pending = items.filter((item) => itemState(item) === "pending");
  const satisfied = items.filter((item) => itemState(item) === "satisfied");
  const unknown = items.filter((item) => itemState(item) === "unknown");
  return {
    total: items.length,
    pending,
    satisfied,
    unknown,
    elevatedPending: pending.filter((item) => item.privileged).length,
    elevatedTotal: items.filter((item) => item.privileged).length,
  };
}

const KIND_ORDER = ["tool", "safeguard", "resource", "scenario"];

const KIND_LABELS: Record<string, string> = {
  tool: "Host tool",
  safeguard: "Host safeguard",
  resource: "Resource",
  scenario: "Scenario",
};

export function kindLabel(kind: string, count: number): string {
  const label = KIND_LABELS[kind] ?? kind.charAt(0).toUpperCase() + kind.slice(1);
  if (count === 1 || label.endsWith("s")) return label;
  return `${label}s`;
}

export interface KindGroup {
  kind: string;
  items: ApplyPlanItem[];
}

/** Groups items by kind in a stable declared order, then by first appearance. */
export function groupByKind(items: ApplyPlanItem[]): KindGroup[] {
  const groups = new Map<string, ApplyPlanItem[]>();
  for (const item of items) {
    const existing = groups.get(item.kind);
    if (existing) existing.push(item);
    else groups.set(item.kind, [item]);
  }
  return [...groups.entries()]
    .map(([kind, grouped]) => ({ kind, items: grouped }))
    .sort((left, right) => {
      const leftRank = KIND_ORDER.indexOf(left.kind);
      const rightRank = KIND_ORDER.indexOf(right.kind);
      return (leftRank === -1 ? KIND_ORDER.length : leftRank) - (rightRank === -1 ? KIND_ORDER.length : rightRank);
    });
}

/**
 * What apply actually runs, per kind. These are the real control-plane commands
 * the executor invokes, so the operator can see that apply has host side
 * effects rather than being a configuration save.
 */
export const APPLY_KIND_ACTIONS: Array<{ kind: string; command: string; effect: string }> = [
  { kind: "tool", command: "vrooli host install <name>", effect: "installs a program on this host" },
  { kind: "safeguard", command: "vrooli host safeguard <name>", effect: "changes host configuration (sysctl, systemd, sudoers and similar)" },
  { kind: "resource", command: "vrooli resource enable <name>", effect: "starts a local service" },
  { kind: "scenario", command: "vrooli scenario start <name>", effect: "starts an app's processes" },
];

/**
 * Two facts an operator asked for that are not visible from the item list.
 * Both are properties of the executor, not wording choices: every item runs
 * even when already present because the run-level already_satisfied check is
 * keyed on the whole selection digest, and nothing is ever removed because the
 * planner skips deselected items rather than emitting a removal.
 */
export const APPLY_CONVERGENCE_NOTE =
  "Every item runs even when already in place; those runs converge rather than reinstall.";
export const APPLY_NO_REMOVAL_NOTE =
  "Nothing is removed, disabled, or uninstalled by apply. Deselected items are skipped, not reverted.";

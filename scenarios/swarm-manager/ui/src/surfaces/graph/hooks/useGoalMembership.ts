/**
 * useGoalMembership — a client-side overlay mapping graph nodes to the goals
 * they belong to, so nodes can render goal badges/tint without the graph proto
 * carrying goal data. Membership = a node whose ref is in a goal's closure or is
 * a goal target. The nodeId→ref index is built once per goals snapshot (the
 * goals query returns a stable array reference until it refetches).
 */

import { useGoals } from "../../plan/hooks/useGoals";
import type { GoalWithScope } from "../../../types/goal";

export interface GoalBadge {
  name: string;
  title: string;
  priority: number;
}

/**
 * nodeIdToGoalRef converts a graph node id to the ref used in goal targets and
 * closures. Backlog nodes ("backlog-item/<kind>/<name>") map to "<kind>/<name>";
 * initiative nodes ("initiative/<name>") pass through. Other nodes cannot be
 * goal members.
 */
function nodeIdToGoalRef(nodeId: string): string | null {
  if (nodeId.startsWith("backlog-item/")) return nodeId.slice("backlog-item/".length);
  if (nodeId.startsWith("initiative/")) return nodeId;
  return null;
}

// Module-level memo keyed by the goals array identity: react-query hands every
// node the same reference, so the index is built once per snapshot rather than
// once per node.
let cachedGoals: GoalWithScope[] | null = null;
let cachedIndex = new Map<string, GoalBadge[]>();

function buildIndex(goals: GoalWithScope[]): Map<string, GoalBadge[]> {
  const index = new Map<string, GoalBadge[]>();
  const add = (ref: string, badge: GoalBadge) => {
    const list = index.get(ref);
    if (list) {
      if (!list.some((b) => b.name === badge.name)) list.push(badge);
    } else {
      index.set(ref, [badge]);
    }
  };
  for (const g of goals) {
    if (g.goal.status !== "active") continue;
    const badge: GoalBadge = { name: g.goal.name, title: g.goal.title, priority: g.goal.priority };
    for (const ref of g.scope.closure) add(ref, badge);
    for (const target of g.goal.targets) add(target, badge);
  }
  return index;
}

function membershipIndex(goals: GoalWithScope[]): Map<string, GoalBadge[]> {
  if (goals !== cachedGoals) {
    cachedGoals = goals;
    cachedIndex = buildIndex(goals);
  }
  return cachedIndex;
}

/**
 * useNodeGoalBadges returns the active goals a node belongs to, highest priority
 * first. Empty when the node is not part of any goal or goals have not loaded.
 */
export function useNodeGoalBadges(nodeId: string): GoalBadge[] {
  const { data: goals } = useGoals();
  if (!goals || goals.length === 0) return [];
  const ref = nodeIdToGoalRef(nodeId);
  if (!ref) return [];
  const badges = membershipIndex(goals).get(ref);
  if (!badges) return [];
  return badges.slice().sort((a, b) => b.priority - a.priority || a.title.localeCompare(b.title));
}

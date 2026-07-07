/**
 * useGoalMembership — a client-side overlay mapping graph nodes to the goals
 * they belong to, so nodes can render goal badges/tint without the graph proto
 * carrying goal data. Membership = a node whose ref is in a goal's closure or is
 * a goal target. The nodeId→ref index is built once per goals snapshot (the
 * goals query returns a stable array reference until it refetches).
 */

import { useMemo } from "react";
import { useGoals } from "../../plan/hooks/useGoals";
import type { GoalWithScope } from "../../../types/goal";
import type { GoalBadge } from "../types";

/**
 * nodeIdToGoalRef converts a graph node id to the ref used in goal targets and
 * closures. Backlog nodes ("backlog-item/<kind>/<name>") map to "<kind>/<name>";
 * initiative nodes ("initiative/<name>") pass through. Other nodes cannot be
 * goal members.
 */
export function nodeIdToGoalRef(nodeId: string): string | null {
  if (nodeId.startsWith("backlog-item/")) return nodeId.slice("backlog-item/".length);
  if (nodeId.startsWith("initiative/")) return nodeId;
  return null;
}

export function buildGoalMembershipIndex(goals: GoalWithScope[]): Map<string, GoalBadge[]> {
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
  for (const badges of index.values()) {
    badges.sort((a, b) => b.priority - a.priority || a.title.localeCompare(b.title));
  }
  return index;
}

const EMPTY_GOAL_MEMBERSHIP_INDEX = new Map<string, GoalBadge[]>();

export function useGoalMembershipIndex(): Map<string, GoalBadge[]> {
  const { data: goals } = useGoals();
  return useMemo(() => {
    if (!goals || goals.length === 0) return EMPTY_GOAL_MEMBERSHIP_INDEX;
    return buildGoalMembershipIndex(goals);
  }, [goals]);
}

/**
 * useNodeGoalBadges returns the active goals a node belongs to, highest priority
 * first. Empty when the node is not part of any goal or goals have not loaded.
 */
export function useNodeGoalBadges(nodeId: string): GoalBadge[] {
  const index = useGoalMembershipIndex();
  const ref = nodeIdToGoalRef(nodeId);
  if (!ref) return [];
  return index.get(ref) ?? [];
}

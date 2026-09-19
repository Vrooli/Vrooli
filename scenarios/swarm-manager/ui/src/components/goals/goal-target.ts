/**
 * goal-target — canonical mapping from a backlog item to the
 * "<kind>/<name>" ref + human title that
 * SetAsGoalDialog consumes. Shared by the graph node inspector, the plan card
 * menu, and the sidebar item context menus so every entry point promotes an
 * entity to a goal with identical semantics.
 */

export interface GoalTarget {
  /** Target ref: "<kind>/<name>" for a backlog item. */
  ref: string;
  /** Human title used to seed a new goal and label the dialog. */
  title: string;
}

/** Goal target for a backlog item ("<kind>/<name>"). */
export function backlogGoalTarget(item: { kind: string; name: string; title?: string }): GoalTarget {
  return { ref: `${item.kind}/${item.name}`, title: item.title || item.name };
}

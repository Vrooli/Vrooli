/**
 * goal-target — canonical mapping from a backlog item or initiative to the
 * "<kind>/<name>" (or "initiative/<name>") ref + human title that
 * SetAsGoalDialog consumes. Shared by the graph node inspector, the plan card
 * menu, and the sidebar item context menus so every entry point promotes an
 * entity to a goal with identical semantics.
 */

export interface GoalTarget {
  /** Target ref: "<kind>/<name>" for an item or "initiative/<name>". */
  ref: string;
  /** Human title used to seed a new goal and label the dialog. */
  title: string;
}

/** Goal target for a backlog item ("<kind>/<name>"). */
export function backlogGoalTarget(item: { kind: string; name: string; title?: string }): GoalTarget {
  return { ref: `${item.kind}/${item.name}`, title: item.title || item.name };
}

/** Goal target for an initiative ("initiative/<name>"). */
export function initiativeGoalTarget(initiative: { name: string; title?: string }): GoalTarget {
  return { ref: `initiative/${initiative.name}`, title: initiative.title || initiative.name };
}

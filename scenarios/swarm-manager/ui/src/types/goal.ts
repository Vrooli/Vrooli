/**
 * Goal types — a goal is an explicit set of end-state targets (backlog items
 * and/or initiatives) whose transitive prerequisite closure defines the work
 * tracked toward it. The API is mux+JSON (snake_case); the goals service
 * normalizes to the camelCase shapes below.
 */

import type { PlanEtaBandData } from "../surfaces/plan/types";

export const GOAL_STATUSES = ["active", "archived"] as const;
export type GoalStatus = (typeof GOAL_STATUSES)[number];

/** A point-in-time record of a goal's closure size (scope-creep tracking). */
export interface GoalScopeSnapshot {
  at: string;
  targetCount: number;
  closureSize: number;
  completed: number;
}

/** The persisted goal entity. */
export interface Goal {
  name: string;
  title: string;
  description?: string;
  status: GoalStatus;
  priority: number;
  /** Target refs: "<kind>/<name>" for items, "initiative/<name>" for initiatives. */
  targets: string[];
  seeded: boolean;
  scopeHistory: GoalScopeSnapshot[];
  created: string;
  updated: string;
  archivedAt?: string;
}

/** The computed scope for a goal (transitive closure + progress). */
export interface GoalScope {
  targets: string[];
  closure: string[];
  completed: string[];
  ready: string[];
  blocked: string[];
  total: number;
  completedCount: number;
  blockedCount: number;
  progressPct: number;
}

/** A goal paired with its computed scope and (optional) ETA band. */
export interface GoalWithScope {
  goal: Goal;
  scope: GoalScope;
  eta: PlanEtaBandData | null;
}

/** Fields for creating a goal. */
export interface CreateGoalInput {
  name?: string;
  title: string;
  description?: string;
  priority?: number;
  targets?: string[];
}

/** Optional fields for updating a goal. */
export interface UpdateGoalInput {
  title?: string;
  description?: string;
  priority?: number;
  targets?: string[];
}

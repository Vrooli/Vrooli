/**
 * Goal types — a goal is an explicit set of end-state backlog item targets
 * whose transitive prerequisite closure defines the work
 * tracked toward it. The API is mux+JSON (snake_case); the goals service
 * normalizes to the camelCase shapes below.
 */

import type { PlanEtaBandData } from "../surfaces/plan/types";
import type { BacklogItem } from "./backlog";

export const GOAL_STATUSES = ["active", "archived"] as const;
export type GoalStatus = (typeof GOAL_STATUSES)[number];

/** A point-in-time record of a goal's closure size (scope-creep tracking). */
export interface GoalScopeSnapshot {
  at: string;
  targetCount: number;
  closureSize: number;
  completed: number;
}

/** An optional, goal-owned subdivision of the goal's derived scope. */
export interface GoalMilestone {
  name: string;
  title: string;
  description?: string;
  items: string[];
  acceptanceCriteria: string[];
  dependsOn: string[];
  archivedAt?: string;
}

/** A read-only file held in a goal's on-disk folder. */
export interface GoalFile {
  path: string;
  size: number;
}

/** The persisted goal entity. */
export interface Goal {
  name: string;
  title: string;
  description?: string;
  status: GoalStatus;
  priority: number;
  /** Target refs: "<kind>/<name>" for backlog items. */
  targets: string[];
  milestones: GoalMilestone[];
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

/**
 * Read-time hydration of the refs the goal detail view renders (targets ∪
 * ready ∪ blocked), keyed by ref — full items so the detail page can render
 * the standard cards without joining list endpoints.
 */
export interface GoalScopeEntities {
  items: Record<string, BacklogItem>;
}

/** A goal paired with its computed scope and (optional) ETA band. */
export interface GoalWithScope {
  goal: Goal;
  scope: GoalScope;
  eta: PlanEtaBandData | null;
  /** Present on detail reads (GET one goal); omitted from list reads. */
  scopeEntities?: GoalScopeEntities;
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

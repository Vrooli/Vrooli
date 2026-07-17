/**
 * Initiative domain types.
 */

import type {
  Initiative as ProtoInitiative,
  InitiativeRollup as ProtoInitiativeRollup,
} from "@vrooli/proto-types/swarm-manager/v1/domain/initiative_pb";
import type { ProtoMessage } from "./shared";
import type { PlanRef } from "./shared";
import type { AgentSessionAttribution } from "./agent-session";

/**
 * Valid lifecycle states for an initiative.
 *
 * Lifecycle:
 *   active → in_review → review_pending → completed | failed | needs_followup
 *
 * - `in_review`: all member items reached terminal status; initiative review
 *   agent is running.
 * - `review_pending`: review complete; awaiting user decision via review-decide.
 * - Terminal transitions (`completed`, `failed`, `needs_followup`) are user-only.
 */
export type InitiativeStatus =
  | "active"
  | "in_review"
  | "review_pending"
  | "completed"
  | "failed"
  | "needs_followup";

/** Initiative operating mode: the execution and validation methodology. */
export type InitiativeOperatingMode = string;

/**
 * A named grouping of related backlog items.
 */
export type Initiative = Omit<ProtoMessage<ProtoInitiative>, "createdBy"> & {
  /** ISO timestamp when the initiative was archived, or undefined if not archived. */
  archivedAt?: string;
  /**
   * Operating mode; blank/legacy records default to the member-item workflow
   * strategy's legacy wire value ("item-level") — see lib/member-item-strategy.ts.
   */
  mode?: InitiativeOperatingMode;
  /** Initiative-level acceptance criteria used by genuine operating modes. */
  acceptanceCriteria?: string[];
  /** Verified provenance for the actor/session that created this initiative. */
  createdBy?: AgentSessionAttribution;
  /** Canonical plan-manager plan backing this initiative or operating mode. */
  planRef?: PlanRef;
};

/**
 * Aggregated status counts for an initiative's member items.
 */
export type InitiativeRollup = ProtoMessage<ProtoInitiativeRollup>;

/**
 * Initiative with computed rollup from member items and the deduped list of
 * scenarios the member items target (derived server-side from each item's
 * acceptance_allow globs via pathutil.ScenariosFromGlobs).
 */
export interface InitiativeWithRollup {
  initiative: Initiative;
  rollup: InitiativeRollup;
  targetScenarios?: string[];
}

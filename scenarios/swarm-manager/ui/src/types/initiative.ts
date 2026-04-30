/**
 * Initiative domain types.
 */

import type {
  Initiative as ProtoInitiative,
  InitiativeRollup as ProtoInitiativeRollup,
} from "@vrooli/proto-types/swarm-manager/v1/domain/initiative_pb";
import type { ProtoMessage } from "./shared";

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
export type InitiativeOperatingMode =
  | "item-level"
  | "holistic-loop"
  | "phased-plan-drain";

/**
 * A named grouping of related backlog items.
 */
export type Initiative = ProtoMessage<ProtoInitiative> & {
  /** ISO timestamp when the initiative was archived, or undefined if not archived. */
  archivedAt?: string;
  /** Operating mode defaults to item-level for historical records. */
  mode?: InitiativeOperatingMode;
  /** Initiative-level acceptance criteria used by non-item-level modes. */
  acceptanceCriteria?: string[];
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

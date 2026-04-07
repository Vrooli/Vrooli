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
 */
export type InitiativeStatus = "active" | "completed";

/**
 * A named grouping of related backlog items.
 */
export type Initiative = ProtoMessage<ProtoInitiative> & {
  /** ISO timestamp when the initiative was archived, or undefined if not archived. */
  archivedAt?: string;
};

/**
 * Aggregated status counts for an initiative's member items.
 */
export type InitiativeRollup = ProtoMessage<ProtoInitiativeRollup>;

/**
 * Initiative with computed rollup from member items.
 */
export interface InitiativeWithRollup {
  initiative: Initiative;
  rollup: InitiativeRollup;
}

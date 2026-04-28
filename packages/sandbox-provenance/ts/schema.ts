// Cross-scenario contract for sandbox provenance records. See
// packages/sandbox-provenance/go/schema.go for the source-of-truth
// documentation and packages/sandbox-provenance/COORDINATION.md for the
// version-bump protocol.

export const SCHEMA_VERSION = "1.0.0";

// Wire keys — both writer and readers reference these constants instead
// of literal strings so a rename catches at compile time.
export const FIELD_RUN_OUTCOME = "runOutcome";
export const FIELD_STATE = "state";
export const FIELD_CONVERSATION_ID = "conversationId";
export const FIELD_COST_USD = "costUsd";
export const FIELD_SCHEMA_VERSION = "schemaVersion";

export type RunOutcome = "" | "success" | "failure" | "cancelled" | "timeout";

export const RUN_OUTCOMES: ReadonlyArray<Exclude<RunOutcome, "">> = [
  "success",
  "failure",
  "cancelled",
  "timeout",
];

export function isRunOutcome(value: string): value is RunOutcome {
  return value === "" || (RUN_OUTCOMES as readonly string[]).includes(value);
}

export type FileState = "" | "applied" | "pending-review" | "denied";

export const FILE_STATES: ReadonlyArray<Exclude<FileState, "">> = [
  "applied",
  "pending-review",
  "denied",
];

export function isFileState(value: string): value is FileState {
  return value === "" || (FILE_STATES as readonly string[]).includes(value);
}

export interface ProvenanceRecord {
  schemaVersion: string;
  runId: string;
  runOutcome?: RunOutcome;
  state?: FileState;
  conversationId?: string;
  costUsd?: number;
}

export class UnknownSchemaVersionError extends Error {
  constructor(got: string) {
    super(`sandbox-provenance: unknown schema version ${JSON.stringify(got)}, want ${JSON.stringify(SCHEMA_VERSION)}`);
    this.name = "UnknownSchemaVersionError";
  }
}

export function validateRecord(r: ProvenanceRecord): void {
  if (r.schemaVersion !== "" && r.schemaVersion !== SCHEMA_VERSION) {
    throw new UnknownSchemaVersionError(r.schemaVersion);
  }
  if (!r.runId) {
    throw new Error("sandbox-provenance: runId is required");
  }
  if (r.runOutcome !== undefined && !isRunOutcome(r.runOutcome)) {
    throw new Error(`sandbox-provenance: invalid runOutcome ${JSON.stringify(r.runOutcome)}`);
  }
  if (r.state !== undefined && !isFileState(r.state)) {
    throw new Error(`sandbox-provenance: invalid state ${JSON.stringify(r.state)}`);
  }
  if (r.costUsd !== undefined && r.costUsd < 0) {
    throw new Error(`sandbox-provenance: costUsd must be ≥ 0, got ${r.costUsd}`);
  }
}

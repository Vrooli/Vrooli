/**
 * Record domain types.
 *
 * Records are immutable narrative artifacts of completed work. See
 * api/internal/records/types.go for the canonical schema.
 */

import type { BacklogKind } from "./backlog";

export type RecordKind = BacklogKind;

export type RecordOutcome = "shipped" | "partial" | "abandoned" | "duplicate";

export const ALL_RECORD_KINDS: RecordKind[] = [
  "idea",
  "research",
  "fix",
  "execute",
  "chore",
];

export const ALL_RECORD_OUTCOMES: RecordOutcome[] = [
  "shipped",
  "partial",
  "abandoned",
  "duplicate",
];

export interface RecordItem {
  id: string;
  kind: RecordKind;
  scenario: string;
  backlogRef?: string;
  milestoneId?: string;
  supersedes?: string;
  supersededBy?: string;
  trigger: string;
  approach: string;
  ruledOut: string[];
  evidence?: string;
  commit?: string;
  filesChanged: string[];
  outcome: RecordOutcome;
  stub: boolean;
  /** Private recovery state from progressive `records capture`. */
  draft?: boolean;
  capture?: RecordCaptureMetadata;
  createdAt: string;
  createdBy?: string;
  narrativeAt?: string;
}

export interface RecordCaptureInvalidField {
  field: string;
  value: string;
  message: string;
}

export interface RecordCaptureMetadata {
  raw?: Record<string, string>;
  accepted?: Record<string, string>;
  needs?: string[];
  invalid?: RecordCaptureInvalidField[];
  warnings?: string[];
}

/**
 * Permissive progressive-intake payload. The capture endpoint deliberately
 * accepts incomplete values and reports what needs repair instead of failing
 * the whole submission.
 */
export interface RecordCaptureInput {
  kind: string;
  scenario: string;
  trigger: string;
  approach: string;
  evidence?: string;
  ruledOut: string[];
  outcome: string;
  createdBy?: string;
  idempotencyKey?: string;
}

export interface RecordCaptureResult {
  disposition: "published" | "draft";
  record: RecordItem;
  accepted: Record<string, string>;
  needs: string[];
  invalid: RecordCaptureInvalidField[];
  warnings: string[];
  nextAction: string[];
}

export interface RecordSearchHit {
  record: RecordItem;
  score: number;
}

export interface RecordListFilter {
  scenario?: string;
  kind?: RecordKind;
  backlogRef?: string;
  includeStubs?: boolean;
  limit?: number;
  offset?: number;
}

export interface RecordNarrativeInput {
  trigger: string;
  approach: string;
  ruledOut: string[];
  commit?: string;
  filesChanged?: string[];
  outcome: RecordOutcome;
}

export interface RecordCreateInput extends RecordNarrativeInput {
  kind: RecordKind;
  scenario: string;
  backlogRef?: string;
  milestoneId?: string;
  supersedes?: string;
  createdBy?: string;
}

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
  supersedes?: string;
  supersededBy?: string;
  trigger: string;
  approach: string;
  ruledOut: string[];
  commit?: string;
  filesChanged: string[];
  outcome: RecordOutcome;
  stub: boolean;
  createdAt: string;
  createdBy?: string;
  narrativeAt?: string;
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
  supersedes?: string;
  createdBy?: string;
}

import { strings } from "../../consts/strings";
import type { Ask, AskOption, AskRevision } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

/** The parts of a decision a requester can change by revising it, and a reader can compare. */
export type RevisionField = "question" | "bigPicture" | "technicalDetails" | "recommendationReason" | "options" | "deadline";

export interface FieldChange {
  field: RevisionField;
  before: string;
  after: string;
}

interface Snapshot {
  question: string;
  bigPicture: string;
  technicalDetails: string;
  recommendationReason: string;
  options: readonly Pick<AskOption, "label">[];
  deadline: string;
}

const optionsText = (options: readonly Pick<AskOption, "label">[]) => options.map((option) => option.label).join(" · ");

const FIELDS: readonly [RevisionField, (snapshot: Snapshot) => string][] = [
  ["question", (snapshot) => snapshot.question],
  ["bigPicture", (snapshot) => snapshot.bigPicture],
  ["technicalDetails", (snapshot) => snapshot.technicalDetails],
  ["recommendationReason", (snapshot) => snapshot.recommendationReason],
  ["options", (snapshot) => optionsText(snapshot.options)],
  ["deadline", (snapshot) => snapshot.deadline],
];

/**
 * What an update changed: the earlier text beside the text that replaced it.
 * `AskRevision` holds what the decision said before update N, so the text
 * after update N is the next revision (N + 1), or the decision as it is now.
 */
export function revisionChanges(ask: Ask, revision: AskRevision): FieldChange[] {
  const after: Snapshot = ask.revisions.find((candidate) => candidate.revision === revision.revision + 1) ?? ask;
  return FIELDS.flatMap(([field, read]) => {
    const [before, now] = [read(revision), read(after)];
    return before !== now ? [{ field, before, after: now }] : [];
  });
}

/** Catalog keys of the field names, written out so each one has a visible call site. */
export const FIELD_LABEL = {
  question: strings.packet.revisions.fields.question,
  bigPicture: strings.packet.revisions.fields.bigPicture,
  technicalDetails: strings.packet.revisions.fields.technicalDetails,
  recommendationReason: strings.packet.revisions.fields.recommendationReason,
  options: strings.packet.revisions.fields.options,
  deadline: strings.packet.revisions.fields.deadline,
} as const satisfies Record<RevisionField, string>;

/**
 * The operator's unsent answer to one decision packet. It lives in the
 * browser (per signed-in subject and ask) so a reload, a lost connection or a
 * revised ask never costs them what they have already chosen. It is plain
 * data: `answer.ts` turns it into the immutable payload that is sent.
 */
export interface SectionVerdictDraft {
  verdict: string;
  note: string;
}

export interface ResponseDraft {
  selected: string[];
  text: string;
  /** Verdicts by section key; a section without an entry takes the default. */
  sections: Record<string, SectionVerdictDraft>;
}

export interface AttachmentNoteDraft {
  attachment: string;
  /** The file inside a folder attachment; "" for a file attachment. */
  path: string;
  note: string;
}

export interface PacketDraft {
  /** The main answer: an option key, or "other". */
  answer: string;
  note: string;
  responses: Record<string, ResponseDraft>;
  attachmentNotes: AttachmentNoteDraft[];
  answerChanges: Record<string, { label: string; value: unknown }>;
  answerImages: { name: string; sha256: string; mediaType: string; sizeBytes: number }[];
  answerAttachments: { name: string; sha256: string; mediaType: string; sizeBytes: number }[];
}

export const emptyResponse = (): ResponseDraft => ({ selected: [], text: "", sections: {} });

export const emptyDraft = (): PacketDraft => ({ answer: "", note: "", responses: {}, attachmentNotes: [], answerChanges: {}, answerImages: [], answerAttachments: [] });

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

function textOf(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function normalizeResponse(value: unknown): ResponseDraft {
  if (!isRecord(value)) return emptyResponse();
  const sections: Record<string, SectionVerdictDraft> = {};
  if (isRecord(value.sections)) {
    for (const [key, section] of Object.entries(value.sections)) {
      if (isRecord(section)) sections[key] = { verdict: textOf(section.verdict), note: textOf(section.note) };
    }
  }
  return {
    selected: Array.isArray(value.selected) ? value.selected.filter((item): item is string => typeof item === "string") : [],
    text: textOf(value.text),
    sections,
  };
}

/** Accept any stored shape, including drafts written before questions existed. */
export function normalizeDraft(value: unknown): PacketDraft | null {
  if (!isRecord(value) || typeof value.answer !== "string" || typeof value.note !== "string") return null;
  const responses: Record<string, ResponseDraft> = {};
  if (isRecord(value.responses)) {
    for (const [id, response] of Object.entries(value.responses)) responses[id] = normalizeResponse(response);
  }
  const attachmentNotes = Array.isArray(value.attachmentNotes)
    ? value.attachmentNotes.flatMap((item): AttachmentNoteDraft[] =>
        isRecord(item) && typeof item.attachment === "string"
          ? [{ attachment: item.attachment, path: textOf(item.path), note: textOf(item.note) }]
          : [])
    : [];
  const answerChanges: PacketDraft["answerChanges"] = {};
  if (isRecord(value.answerChanges)) {
    for (const [key, change] of Object.entries(value.answerChanges)) {
      if (isRecord(change) && typeof change.label === "string") answerChanges[key] = { label: change.label, value: change.value };
    }
  }
  const answerImages = Array.isArray(value.answerImages)
    ? value.answerImages.flatMap((item) => isRecord(item) && typeof item.name === "string" && typeof item.sha256 === "string" && typeof item.mediaType === "string" && typeof item.sizeBytes === "number"
      ? [{ name: item.name, sha256: item.sha256, mediaType: item.mediaType, sizeBytes: item.sizeBytes }]
      : [])
    : [];
  const answerAttachments = Array.isArray(value.answerAttachments)
    ? value.answerAttachments.flatMap((item) => isRecord(item) && typeof item.name === "string" && typeof item.sha256 === "string" && typeof item.mediaType === "string" && typeof item.sizeBytes === "number"
      ? [{ name: item.name, sha256: item.sha256, mediaType: item.mediaType, sizeBytes: item.sizeBytes }]
      : [])
    : [];
  return { answer: value.answer, note: value.note, responses, attachmentNotes, answerChanges, answerImages, answerAttachments };
}

/** The draft with one question's response replaced. */
export function withResponse(draft: PacketDraft, questionId: string, response: ResponseDraft): PacketDraft {
  return { ...draft, responses: { ...draft.responses, [questionId]: response } };
}

/** The draft with the note for one attachment (or one file in a folder) set; an empty note removes it. */
export function withAttachmentNote(draft: PacketDraft, attachment: string, path: string, note: string): PacketDraft {
  const others = draft.attachmentNotes.filter((item) => item.attachment !== attachment || item.path !== path);
  if (!note.trim()) return { ...draft, attachmentNotes: others };
  const index = draft.attachmentNotes.findIndex((item) => item.attachment === attachment && item.path === path);
  const next = { attachment, path, note };
  if (index < 0) return { ...draft, attachmentNotes: [...others, next] };
  const replaced = [...draft.attachmentNotes];
  replaced[index] = next;
  return { ...draft, attachmentNotes: replaced };
}

export function attachmentNoteFor(draft: PacketDraft, attachment: string, path = ""): string {
  return draft.attachmentNotes.find((item) => item.attachment === attachment && item.path === path)?.note ?? "";
}

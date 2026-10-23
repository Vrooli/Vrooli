export interface AskDraft {
  answer: string;
  note: string;
}

const drafts = new Map<string, AskDraft>();
const keyOf = (subject: string, askId: string) => `${subject}\u0000${askId}`;

export function readAskDraft(subject: string, askId: string): AskDraft {
  return drafts.get(keyOf(subject, askId)) ?? { answer: "", note: "" };
}

export function writeAskDraft(subject: string, askId: string, draft: AskDraft): void {
  drafts.set(keyOf(subject, askId), draft);
}

export function clearAskDraft(subject: string, askId: string): void {
  drafts.delete(keyOf(subject, askId));
}

export function clearAskDrafts(): void {
  drafts.clear();
}

/** One operator entry is at most this many characters (rulings F9). */
export const ENTRY_LIMIT = 4000;

/** Characters as the API counts them: code points of the text that is sent. */
export function entryLength(text: string): number {
  return Array.from(text.trim()).length;
}

export function entryTooLong(text: string): boolean {
  return entryLength(text) > ENTRY_LIMIT;
}

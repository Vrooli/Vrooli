/** Characters as the API counts them: code points of the text that is sent. */
export function entryLength(text: string): number {
  return Array.from(text.trim()).length;
}

/**
 * Past the entry limit the API sends with the page's data (`entry_chars_max`).
 * A limit of 0 is unknown: nothing is refused here, and the API refuses the write.
 */
export function entryTooLong(text: string, limit: number): boolean {
  return limit > 0 && entryLength(text) > limit;
}

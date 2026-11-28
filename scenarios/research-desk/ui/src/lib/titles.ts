/**
 * A commission's generated title is "Research commission <id> — <what it is>".
 * The id already sits in the eyebrow, so the display title is the part after
 * the dash. Any other title is shown as written.
 */
export function displayTitle(title: string | undefined, id: string): string {
  const value = title?.trim() || id;
  const dash = value.indexOf("—");
  return dash >= 0 ? value.slice(dash + 1).trim() || value : value;
}

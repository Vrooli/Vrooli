/**
 * Pick an inline `defaultValue` template for a count.
 *
 * The workbench passes English copy as `t(key, { defaultValue })` rather than
 * registering a catalog entry with `_one`/`_other` suffixes, so i18next cannot
 * pluralize it on its own. This keeps "1 revision" / "2 revisions" honest
 * without inventing a second copy of the string in JSON.
 */
export function pluralDefault(count: number, singular: string, plural: string): string {
  return count === 1 ? singular : plural;
}

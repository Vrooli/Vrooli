/**
 * Substitute `{{name}}` placeholders in a catalog string with concrete
 * values for assertion. Centralised so an eventual ICU-MessageFormat
 * migration is a one-file change instead of touching every plural test.
 *
 * Use only in tests that explicitly verify the i18n pipeline end-to-end
 * — cimode-default tests assert on the raw key path via `strings.x.y`
 * and never need this helper.
 */
export const interp = (template: string, vars: Record<string, string | number>): string =>
  template.replace(/\{\{(\w+)\}\}/g, (_match, name: string) => {
    const value = vars[name];
    if (value === undefined) {
      throw new Error(`interp(): template expects '{{${name}}}' but no value was provided`);
    }
    return String(value);
  });

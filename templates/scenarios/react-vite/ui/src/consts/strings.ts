/**
 * Vrooli Ascension string registry
 *
 * Single source of truth for every user-facing string in the UI. Mirrors the
 * shape of `selectors.ts`: a typed nested object, a single `strings` export,
 * and a small interpolation helper. Components and tests both read through
 * this file.
 *
 * ## Why a registry instead of inline JSX strings?
 *
 * 1. **Test stability.** Tests reference `strings.health.refresh` instead of
 *    the literal `"Refresh"`. Copy edits don't break assertions.
 * 2. **i18n upgrade path.** When a scenario actually needs a second language,
 *    swap this module's `strings` export for a `t()` accessor backed by
 *    react-i18next (or similar) without changing any callsite. The nested
 *    object structure already maps to i18n JSON namespaces.
 * 3. **Discoverability.** "Where does this string come from?" is one grep,
 *    not a haystack search across the codebase.
 *
 * ## How to add a string
 *
 * 1. Add the key under the appropriate feature in the `en` object below.
 * 2. Reference it in JSX as `{strings.feature.key}`.
 * 3. For dynamic values, use `${var}` placeholders and call
 *    `format(strings.feature.key, { var: value })`. Do NOT use template
 *    literals to splice values into a registry string — that hides the
 *    final shape from translators and from the type system.
 *
 * ## What NOT to do
 *
 * - Don't inline JSX literals (`<h1>Hello</h1>`). The `no-restricted-syntax`
 *   ESLint rule rejects them and points contributors back to this file.
 * - Don't put non-user-facing strings here (test IDs go in `selectors.ts`,
 *   API URLs go in config, log messages stay where they're emitted).
 * - Don't mutate `strings` at runtime. The `as const` assertion is
 *   load-bearing for the `Strings` type.
 */

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

/**
 * Substitute `${var}` placeholders in a registry string with values from
 * `params`. Throws on missing parameters so typos surface in tests instead
 * of leaking literal `${...}` text into the UI.
 */
export const format = (
  template: string,
  params: Record<string, string | number>,
): string =>
  template.replace(TEMPLATE_TOKEN, (_match: string, token: string) => {
    if (!(token in params)) {
      throw new Error(
        `Missing parameter '${token}' for string template '${template}'`,
      );
    }
    return String(params[token]);
  });

const en = {
  app: {
    title: "{{SCENARIO_DISPLAY_NAME}}",
    eyebrow: "Scenario Template",
    description:
      "This starter UI is intentionally minimal. Replace it with your scenario-specific experience while keeping the styling conventions (Tailwind + shadcn) and API wiring in place.",
  },
  health: {
    title: "API Health",
    loading: "Checking API status…",
    error:
      "Unable to reach the API. Make sure the scenario is running through `vrooli scenario start`.",
    refresh: "Refresh",
    statusLabel: "Status:",
    serviceLabel: "Service:",
    timestampLabel: "Timestamp:",
  },
} as const;

export const strings = en;
export type Strings = typeof strings;

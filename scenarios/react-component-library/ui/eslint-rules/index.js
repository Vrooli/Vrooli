/**
 * Vrooli string-registry ESLint plugin.
 *
 * Two custom rules enforce the i18n-registry contract that the rest of the
 * template scaffolds:
 *
 *   - `strings/codegen-fresh`   — fails when src/consts/strings.generated.ts
 *                                 diverges from src/i18n/locales/en.json.
 *   - `strings/no-unused-keys`  — fails when a catalog key in en.json has
 *                                 no callsite under src/.
 *
 * Both rules anchor on `src/consts/strings.generated.ts` so the gates run
 * exactly once per lint pass, regardless of how many files ESLint visits.
 *
 * Wired into eslint.config.js as `plugins: { strings: stringsPlugin }`,
 * with each rule turned on at error severity. test-genie's lint phase
 * runs `eslint . --format json` and the template's `.vrooli/testing.json`
 * sets `node_package: { strict: true }`, so any rule violation fails the
 * lint phase under `vrooli scenario test`.
 */
import codegenFresh from "./codegen-fresh.js";
import noUnusedKeys from "./no-unused-keys.js";
import noRawDimensions from "./no-raw-dimensions.js";

export default {
  rules: {
    "codegen-fresh": codegenFresh,
    "no-unused-keys": noUnusedKeys,
  },
};

/**
 * Vrooli design-system ESLint plugin.
 *
 * Separate from the string-registry plugin above because it applies to a
 * different surface: the string rules anchor on this scenario's generated
 * registry and run once per pass, while the design-system rules are per-file
 * and are shared by BOTH eslint configs — the app config over `src/**` and
 * the catalog config over `../library/**`. Keeping them in one plugin object
 * that both configs register is what stops the app and the library from
 * drifting into different rulesets again.
 *
 *   - `design-system/no-raw-dimensions` — rejects raw Tailwind spacing and
 *     sizing utilities in class strings, with a different remediation per
 *     family (spacing has a ramp step to move to; sizing does not).
 */
export const designSystem = {
  rules: {
    "no-raw-dimensions": noRawDimensions,
  },
};

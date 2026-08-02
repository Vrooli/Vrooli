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

export default {
  rules: {
    "codegen-fresh": codegenFresh,
    "no-unused-keys": noUnusedKeys,
  },
};

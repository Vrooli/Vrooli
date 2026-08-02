/**
 * ESLint rule: `strings/codegen-fresh`
 *
 * Fails when `src/consts/strings.generated.ts` diverges from what the
 * codegen would produce given the current `src/i18n/locales/en.json`.
 *
 * Why this rule exists: the Vite plugin auto-regenerates `strings.generated.ts`
 * during dev/test/build, so drift typically auto-heals locally — which means
 * a developer can hand-edit the generated file (they shouldn't, it's a
 * `// AUTO-GENERATED` file) and not notice. Without this rule, that drift
 * wouldn't surface until someone bumped a key in `en.json` and the codegen
 * overwrote their hand-edit. This rule catches it at lint time.
 *
 * Anchoring strategy: the rule emits exactly when the file being linted is
 * `src/consts/strings.generated.ts`. ESLint already lints that file via the
 * template's `files: ["**\/*.{ts,tsx}"]` glob, the file exists exactly once
 * per scenario, and tying the diagnostic to it makes the report location
 * navigable in the editor.
 *
 * Source-of-truth reuse: the rule imports `generateContents()` from
 * `scripts/gen-strings.mjs` rather than reimplementing the codegen. If both
 * sides walked the catalog independently, drift between *their* logic would
 * defeat the entire freshness check. Inheriting the codegen also inherits
 * the underscore-prefix sentinel skip from gen-strings.mjs's `isSentinelKey`.
 */
import { readFileSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, relative } from "node:path";
import { generateContents } from "../scripts/gen-strings.mjs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
// eslint-rules/ sits one level under <ui-root>; back up to find the target.
const UI_ROOT = join(__dirname, "..");
const TARGET_PATH = join(UI_ROOT, "src/consts/strings.generated.ts");

const ANCHOR_BASENAME = "strings.generated.ts";

const firstDivergentLine = (a, b) => {
  const aLines = a.split("\n");
  const bLines = b.split("\n");
  const len = Math.max(aLines.length, bLines.length);
  for (let i = 0; i < len; i++) {
    if (aLines[i] !== bLines[i]) {
      return {
        lineNumber: i + 1,
        actual: aLines[i] ?? "(missing line)",
        expected: bLines[i] ?? "(missing line)",
      };
    }
  }
  return null;
};

export default {
  meta: {
    type: "problem",
    docs: {
      description:
        "Ensure src/consts/strings.generated.ts matches what scripts/gen-strings.mjs would produce from src/i18n/locales/en.json.",
    },
    schema: [],
    messages: {
      stale:
        "{{file}} is out of sync with en.json (first divergence at line {{line}}). Run `pnpm strings:gen` and commit the result.\n  expected: {{expected}}\n  actual:   {{actual}}",
    },
  },
  create(context) {
    // Anchor: only emit when linting the generated file itself. Every other
    // file pays a cheap basename check and bails.
    const filename = context.filename;
    if (!filename.endsWith(ANCHOR_BASENAME)) {
      return {};
    }

    return {
      Program(node) {
        if (!existsSync(TARGET_PATH)) {
          // The generated file is missing entirely — codegen has never run
          // or the file was deleted. Surface as stale; the message points
          // at the fix command.
          context.report({
            node,
            messageId: "stale",
            data: {
              file: relative(UI_ROOT, TARGET_PATH),
              line: "1",
              expected: "<file should exist>",
              actual: "<missing>",
            },
          });
          return;
        }
        const expected = generateContents();
        const actual = readFileSync(TARGET_PATH, "utf-8");
        if (expected === actual) return;

        const diff = firstDivergentLine(actual, expected) ?? {
          lineNumber: 1,
          actual: "(no line-level divergence found)",
          expected: "(no line-level divergence found)",
        };

        context.report({
          node,
          messageId: "stale",
          data: {
            file: relative(UI_ROOT, TARGET_PATH),
            line: String(diff.lineNumber),
            expected: diff.expected,
            actual: diff.actual,
          },
        });
      },
    };
  },
};

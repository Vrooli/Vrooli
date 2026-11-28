/**
 * Calibration for `design-system/no-raw-dimensions`.
 *
 * Every case here is a deliberately-broken input the rule must catch, or a
 * legitimate input it must leave alone. The repo convention is that no check
 * is trusted until it has been shown to fail on a known break — a rule that
 * has only ever been observed passing is indistinguishable from a rule that
 * inspects nothing (which is exactly how the Go gates grew their
 * `_zero_inspected` contract).
 *
 * The `valid` half matters as much as the `invalid` half: false positives are
 * what get a rule disabled, and a disabled rule protects nothing.
 */
import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import rule from "./no-raw-dimensions.js";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "module",
    parserOptions: { ecmaFeatures: { jsx: true } },
  },
});

describe("design-system/no-raw-dimensions", () => {
  it("catches raw dimensions and leaves ramp usage alone", () => {
    ruleTester.run("no-raw-dimensions", rule, {
      valid: [
        // Ramp utilities are the point of the rule; they must never report.
        `const a = <div className="p-space-sm gap-space-2xs" />;`,
        `const b = <div className="mx-space-md" />;`,
        // Non-dimension utilities that merely contain digits or hyphens.
        `const c = <div className="grid-cols-2 z-10 opacity-50" />;`,
        // Structural utilities with no ramp equivalent are out of scope.
        `const d = <div className="min-h-0 flex-1 w-full h-auto" />;`,
        // Zero is the absence of spacing, not an untokenized amount of it.
        // The ramp has no zero rung and should not grow one.
        `const d0 = <div className="p-0 gap-0 -mt-0 w-0" />;`,
        // Prose and non-class strings must not be swept.
        `const e = "increase padding to p-4 for the next release";`,
        `const f = someOtherHelper("p-4");`,
        // Ramp usage inside a class helper.
        `const g = clsx("p-space-sm", isActive && "gap-space-xs");`,
      ],
      invalid: [
        // Spacing that maps cleanly onto a ramp step — autofixable.
        {
          code: `const a = <div className="p-4" />;`,
          output: `const a = <div className="p-space-sm" />;`,
          errors: [{ messageId: "spacingFixable" }],
        },
        {
          code: `const b = <div className="gap-3 mt-6" />;`,
          output: `const b = <div className="gap-space-xs mt-space-md" />;`,
          errors: [{ messageId: "spacingFixable" }, { messageId: "spacingFixable" }],
        },
        // Responsive and state variants must be preserved through the fix.
        {
          code: `const c = <div className="md:p-8 hover:gap-2" />;`,
          output: `const c = <div className="md:p-space-lg hover:gap-space-2xs" />;`,
          errors: [{ messageId: "spacingFixable" }, { messageId: "spacingFixable" }],
        },
        // Negative margins keep their sign.
        {
          code: `const d = <div className="-mt-4" />;`,
          output: `const d = <div className="-mt-space-sm" />;`,
          errors: [{ messageId: "spacingFixable" }],
        },
        // Spacing that lands between ramp steps has no mechanical fix.
        {
          code: `const e = <div className="p-5" />;`,
          errors: [{ messageId: "spacingUnmapped" }],
        },
        // Sizing must NOT be offered a token — the ramp publishes none. This
        // is the case that made the old Go gate message actively misleading.
        {
          code: `const f = <Search className="h-4 w-4" />;`,
          errors: [{ messageId: "sizing" }, { messageId: "sizing" }],
        },
        {
          code: `const g = <div className="h-3.5" />;`,
          errors: [{ messageId: "sizing" }],
        },
        // Arbitrary values: fixable only when they land on a ramp step.
        {
          code: `const h = <div className="p-[16px]" />;`,
          output: `const h = <div className="p-space-sm" />;`,
          errors: [{ messageId: "arbitraryFixable" }],
        },
        {
          code: `const i = <div className="p-[13px]" />;`,
          errors: [{ messageId: "arbitrary" }],
        },
        // Class helpers are in scope.
        {
          code: `const j = clsx("p-4", active && "gap-2");`,
          output: `const j = clsx("p-space-sm", active && "gap-space-2xs");`,
          errors: [{ messageId: "spacingFixable" }, { messageId: "spacingFixable" }],
        },
        // Template literals report against the correct quasi.
        {
          code: "const k = <div className={`p-4 ${extra}`} />;",
          output: "const k = <div className={`p-space-sm ${extra}`} />;",
          errors: [{ messageId: "spacingFixable" }],
        },
      ],
    });
  });
});

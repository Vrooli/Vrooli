/**
 * Self-tests for `interp`.
 *
 * `interp` is the only piece of test-utils logic that touches a regex
 * and throws on bad input — silent regressions here would corrupt every
 * real-locale plural test that relies on it. The contract pinned below:
 *
 *   - `{{name}}` placeholders are substituted by string or number values
 *   - templates without placeholders pass through unchanged
 *   - missing values throw with the placeholder name in the message
 *     (so the failure is self-diagnosing in CI logs)
 *   - braces with non-word content (`{{1+1}}`, `{{ }}`) are left alone —
 *     the regex is `\w+`, not `.*`, on purpose
 */
import { describe, expect, it } from "vitest";

import { interp } from "./interp";

describe("interp", () => {
  it("substitutes a single placeholder with a string value", () => {
    expect(interp("Hello, {{name}}!", { name: "world" })).toBe("Hello, world!");
  });

  it("substitutes multiple placeholders in a single template", () => {
    expect(interp("{{count}} of {{total}}", { count: 3, total: 10 })).toBe("3 of 10");
  });

  it("coerces numeric values via String()", () => {
    expect(interp("{{n}}", { n: 0 })).toBe("0");
    expect(interp("{{n}}", { n: -1.5 })).toBe("-1.5");
  });

  it("returns templates without placeholders unchanged", () => {
    expect(interp("plain text", {})).toBe("plain text");
  });

  it("throws when a referenced placeholder has no value", () => {
    expect(() => interp("Hello, {{name}}!", {})).toThrowError(
      /interp\(\): template expects '\{\{name\}\}' but no value was provided/,
    );
  });

  it("ignores braces whose contents don't match \\w+", () => {
    // CLDR plural keys never contain non-word chars, so the strict
    // regex is by design — substitution must not corrupt template
    // fragments like `{{1 + 1}}` or `{{ a }}`.
    expect(interp("{{ a }}", { a: "x" })).toBe("{{ a }}");
    expect(interp("{{1+1}}", {})).toBe("{{1+1}}");
  });

  it("substitutes the same placeholder name in multiple positions", () => {
    expect(interp("{{n}} + {{n}} = {{n}}{{n}}", { n: 1 })).toBe("1 + 1 = 11");
  });
});

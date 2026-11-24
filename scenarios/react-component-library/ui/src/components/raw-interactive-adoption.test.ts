import { readFileSync, readdirSync } from "node:fs";
import { dirname, join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

/**
 * Adoption guard.
 *
 * The rule is deliberately not "never render a DOM element": a primitive has to
 * bottom out in a real element somewhere, or the library would be turtles all the
 * way down. What the guard enforces is that a *composition* — a component in this
 * folder that is not itself the versioned primitive layer — may not hand-roll an
 * interactive element when `Button` / `IconButton` / `Pressable` / `ControlBase` /
 * `Input` / `Select` / `Textarea` already own that control's tap target, hover,
 * :focus-visible, disabled treatment and token-backed motion.
 *
 * A new component that renders a raw interactive element fails this test until
 * someone either composes the primitive or writes down, here, why it cannot.
 * Prefer the former.
 */
const RAW_INTERACTIVE_ELEMENTS = ["button", "select", "textarea", "input"] as const;

/**
 * Files under `versions/` directories ARE the primitive layer: extracted, version-pinned
 * assets whose whole job is to turn a token contract into a DOM element. They are
 * exempt by category, not by oversight.
 */
const PRIMITIVE_LAYER = /[\\/]versions[\\/]/;

/**
 * Compositions that legitimately render a raw interactive element. Every entry
 * needs a reason a reviewer can disagree with. Adding an entry is the fallback;
 * composing the primitive is the fix.
 */
const JUSTIFIED_RAW_ELEMENTS: Record<string, string> = {};

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) return sourceFiles(full);
    if (!entry.name.endsWith(".tsx")) return [];
    if (entry.name.endsWith(".test.tsx")) return [];
    return [full];
  });
}

/** Comments describe the rule; they must not trip it. */
function stripComments(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, "");
}

function rawElementsIn(source: string): string[] {
  const body = stripComments(source);
  return RAW_INTERACTIVE_ELEMENTS.filter((tag) => new RegExp(`<${tag}(\\s|>|$)`, "m").test(body));
}

describe("raw interactive element adoption", () => {
  const root = dirname(fileURLToPath(import.meta.url));
  const compositions = sourceFiles(root)
    .filter((file) => !PRIMITIVE_LAYER.test(file))
    .map((file) => ({ key: relative(root, file).split(sep).join("/"), file }));

  it("finds the composition layer it is supposed to police", () => {
    expect(compositions.length).toBeGreaterThan(15);
  });

  it("lets no composition hand-roll an interactive element without a written reason", () => {
    const unexplained = compositions
      .map(({ key, file }) => ({ key, tags: rawElementsIn(readFileSync(file, "utf8")) }))
      .filter(({ key, tags }) => tags.length > 0 && !JUSTIFIED_RAW_ELEMENTS[key])
      .map(({ key, tags }) => `${key} renders <${tags.join(">, <")}>`);

    expect(
      unexplained,
      "Compose Button/IconButton/Pressable/ControlBase/Input/Select/Textarea instead, " +
        "or add a justification to JUSTIFIED_RAW_ELEMENTS in this file.",
    ).toEqual([]);
  });

  it("keeps every justification live, so the exemption list cannot rot", () => {
    const stale = Object.keys(JUSTIFIED_RAW_ELEMENTS).filter((key) => {
      const match = compositions.find((entry) => entry.key === key);
      return !match || rawElementsIn(readFileSync(match.file, "utf8")).length === 0;
    });

    expect(
      stale,
      "These files no longer render a raw interactive element — drop the entry.",
    ).toEqual([]);
  });

  it("holds every justification to a real explanation", () => {
    for (const [key, reason] of Object.entries(JUSTIFIED_RAW_ELEMENTS)) {
      expect(reason.length, `${key} needs a reason a reviewer can disagree with`).toBeGreaterThan(
        60,
      );
    }
  });
});

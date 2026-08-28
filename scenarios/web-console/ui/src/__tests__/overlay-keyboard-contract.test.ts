import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

/**
 * Every overlay states whether it moves for the virtual keyboard.
 *
 * The library implements keyboard avoidance once — BaseStyles publishes the
 * host viewport contract (`--rcl-keyboard-inset`, `--rcl-viewport-height`),
 * `useAppViewport` fills those in from `visualViewport`, and each overlay
 * primitive's CSS consumes them. But the primitive's `avoidKeyboard` prop
 * defaults to FALSE, so an overlay that simply never mentions it silently
 * gets the wrong behaviour: on a phone the keyboard slides up over the very
 * field the operator is typing into.
 *
 * That is exactly how twelve of this app's fifteen overlays ended up broken.
 * The bug is invisible in code review because the correct code and the broken
 * code differ by an absent line, so the guard is that the line can never be
 * absent: an overlay either opts in, or says in one annotated `false` why it
 * has nothing to avoid.
 */

const OVERLAY_PRIMITIVES = ["ResponsiveDialog", "FullPageDrawer", "BottomSheet", "DrawerShell"];
const COMPONENT_ROOT = join(__dirname, "..", "components");

/** Every .tsx under src/components, at any depth. */
function componentFiles(dir: string): string[] {
  const found: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === "__tests__") continue;
      found.push(...componentFiles(path));
    } else if (entry.name.endsWith(".tsx") && !entry.name.includes(".test.")) {
      found.push(path);
    }
  }
  return found;
}

/** Files that actually render one of the overlay primitives. */
const overlayFiles = componentFiles(COMPONENT_ROOT).filter((path) => {
  const source = readFileSync(path, "utf8");
  return OVERLAY_PRIMITIVES.some((prim) => new RegExp(`<${prim}[\\s>]`).test(source));
});

describe("overlay keyboard contract", () => {
  // A guard whose subject set can silently empty out is not a guard. If a
  // refactor moves every overlay elsewhere, this fails rather than passing
  // vacuously.
  it("finds the overlays it is meant to police", () => {
    expect(overlayFiles.length).toBeGreaterThanOrEqual(10);
  });

  it.each(overlayFiles.map((path) => [path.slice(path.indexOf("src/")), path] as const))(
    "%s states whether it avoids the virtual keyboard",
    (_label, path) => {
      expect(readFileSync(path, "utf8")).toContain("avoidKeyboard");
    },
  );

  it("fails an overlay that says nothing", () => {
    // The mutation this guard exists to catch: the same file with the line
    // removed must not pass. Asserting only the current tree would leave a
    // check that cannot distinguish a fixed codebase from a broken one.
    const stripped = readFileSync(overlayFiles[0] as string, "utf8").replace(/avoidKeyboard/g, "");
    expect(stripped).not.toContain("avoidKeyboard");
  });

  // The opt-out is the dangerous half: it is the shape the default already
  // had, so it must be a decision someone wrote down rather than one that
  // drifted back in.
  it.each(overlayFiles.map((path) => [path.slice(path.indexOf("src/")), path] as const))(
    "%s explains any opt-out",
    (_label, path) => {
      const source = readFileSync(path, "utf8");
      if (!source.includes("avoidKeyboard={false}")) return;
      const before = source.slice(0, source.indexOf("avoidKeyboard={false}"));
      const precedingLines = before.trimEnd().split("\n").slice(-3).join("\n");
      expect(precedingLines, `${path} opts out with no stated reason`).toMatch(/\/\/|\*/);
    },
  );
});

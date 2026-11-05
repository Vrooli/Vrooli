import { execFileSync } from "node:child_process";
import path from "node:path";
import { describe, expect, it } from "vitest";

/**
 * The unit phase owns this gate on purpose.
 *
 * A tsconfig that does not allow what the test files use makes the whole
 * scenario unbuildable, and an unbuildable scenario produces no phase evidence
 * at all — the suite reports a run with no result rather than a named failure,
 * which is the least actionable outcome there is. Running the project's own
 * type-check here turns that into one named unit failure.
 */
describe("TypeScript project contract", () => {
  it("type-checks the whole project with the committed tsconfig", () => {
    const uiRoot = path.resolve(__dirname, "..");
    expect(() =>
      execFileSync("./node_modules/.bin/tsc", ["--noEmit"], {
        cwd: uiRoot,
        encoding: "utf8",
        stdio: "pipe",
      }),
    ).not.toThrow();
  }, 180_000);
});

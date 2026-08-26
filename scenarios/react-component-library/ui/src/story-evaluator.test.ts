import * as testingLibraryDOM from "@testing-library/dom";
import { describe, expect, it } from "vitest";

import { jsdomEnv, runStory } from "../../api/handlers/preview/assets/story-evaluator.js";

describe("shared preview story evaluator", () => {
  it("uses Testing Library role/name queries and skips layout in jsdom", async () => {
    document.body.innerHTML = `<main><button aria-label="Save changes">Save</button></main>`;
    const reports: unknown[] = [];

    const result = await runStory(
      {
        kind: "component",
        interactions: [],
        expect: [
          { kind: "role", role: "button", name: "Save changes" },
          { kind: "text", value: "Save" },
          { kind: "layout", selector: "main", minWidth: 1 },
        ],
      },
      { document, window },
      { ...jsdomEnv, queries: testingLibraryDOM, report: (...args: unknown[]) => reports.push(args) },
    );

    expect(result.passed).toBe(true);
    expect(result.failures).toEqual([]);
    expect(result.skipped).toHaveLength(1);
    expect(reports).toHaveLength(1);
  });
});

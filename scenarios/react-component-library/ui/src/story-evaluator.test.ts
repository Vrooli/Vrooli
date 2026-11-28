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
      {
        ...jsdomEnv,
        queries: testingLibraryDOM,
        report: (...args: unknown[]) => reports.push(args),
      },
    );

    expect(result.passed).toBe(true);
    expect(result.failures).toEqual([]);
    expect(result.skipped).toHaveLength(1);
    expect(reports).toHaveLength(1);
  });

  it("fails a same-row expectation when visible children wrap", async () => {
    document.body.innerHTML = `<main data-actions><button>Cancel</button><button>Reset</button><button>Save</button></main>`;
    const root = document.querySelector("main")!;
    const buttons = Array.from(root.children);
    root.getBoundingClientRect = () => ({ x: 0, y: 0, top: 0, right: 320, bottom: 64, left: 0, width: 320, height: 64, toJSON: () => ({}) });
    buttons.forEach((button, index) => {
      button.getBoundingClientRect = () => ({ x: 0, y: index === 1 ? 60 : 0, top: index === 1 ? 60 : 0, right: 100, bottom: 44, left: 0, width: 100, height: 44, toJSON: () => ({}) });
    });

    const result = await runStory(
      {
        kind: "component",
        interactions: [],
        expect: [{ kind: "layout", selector: "[data-actions]", sameRow: true }],
      },
      { document, window },
      { ...jsdomEnv, skipKinds: new Set(), queries: testingLibraryDOM },
    );

    expect(result.passed).toBe(false);
    expect(result.failures[0]?.message).toBe("expected visible children to remain on one row");
  });

  it("passes a same-row expectation when visible children share one row", async () => {
    document.body.innerHTML = `<main data-actions><button>Cancel</button><button>Reset</button><button>Save</button></main>`;
    const root = document.querySelector("main")!;
    const buttons = Array.from(root.children);
    root.getBoundingClientRect = () => ({ x: 0, y: 0, top: 0, right: 320, bottom: 44, left: 0, width: 320, height: 44, toJSON: () => ({}) });
    buttons.forEach((button) => {
      button.getBoundingClientRect = () => ({ x: 0, y: 0, top: 0, right: 100, bottom: 44, left: 0, width: 100, height: 44, toJSON: () => ({}) });
    });

    const result = await runStory(
      {
        kind: "component",
        interactions: [],
        expect: [{ kind: "layout", selector: "[data-actions]", sameRow: true }],
      },
      { document, window },
      { ...jsdomEnv, skipKinds: new Set(), queries: testingLibraryDOM },
    );

    expect(result.passed).toBe(true);
    expect(result.failures).toEqual([]);
  });
});

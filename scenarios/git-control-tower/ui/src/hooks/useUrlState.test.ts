import { parseUrlState, buildUrlSearch } from "./useUrlState";
import type { UrlState } from "./useUrlState";

// --- parseUrlState ---

describe("parseUrlState", () => {
  test("returns empty state for empty search string", () => {
    expect(parseUrlState("")).toEqual({});
  });

  test("parses file param with URL decoding", () => {
    const state = parseUrlState("?file=src%2FApp.tsx");
    expect(state.file).toBe("src/App.tsx");
  });

  test("parses file with special characters", () => {
    const state = parseUrlState("?file=path%20with%20spaces%2Ffile%26name.ts");
    expect(state.file).toBe("path with spaces/file&name.ts");
  });

  test("parses valid mode values", () => {
    for (const mode of ["diff", "full_diff", "source", "preview"]) {
      expect(parseUrlState(`?mode=${mode}`).mode).toBe(mode);
    }
  });

  test("ignores invalid mode values", () => {
    expect(parseUrlState("?mode=invalid").mode).toBeUndefined();
    expect(parseUrlState("?mode=").mode).toBeUndefined();
  });

  test("parses staged=true", () => {
    expect(parseUrlState("?staged=true").staged).toBe(true);
  });

  test("parses staged=false", () => {
    expect(parseUrlState("?staged=false").staged).toBe(false);
  });

  test("ignores invalid staged values", () => {
    expect(parseUrlState("?staged=yes").staged).toBeUndefined();
  });

  test("parses panel values", () => {
    expect(parseUrlState("?panel=changes").panel).toBe("changes");
    expect(parseUrlState("?panel=related").panel).toBe("related");
    expect(parseUrlState("?panel=invalid").panel).toBeUndefined();
  });

  test("parses commit hash", () => {
    expect(parseUrlState("?commit=abc123").commit).toBe("abc123");
  });

  test("parses primary panel", () => {
    expect(parseUrlState("?primary=review").primary).toBe("review");
    expect(parseUrlState("?primary=history").primary).toBe("history");
  });

  test("parses reviewScenario with URL decoding", () => {
    expect(parseUrlState("?reviewScenario=my-scenario").reviewScenario).toBe("my-scenario");
    expect(parseUrlState("?reviewScenario=name%20with%20spaces").reviewScenario).toBe("name with spaces");
  });

  test("parses valid reviewTab values", () => {
    for (const tab of ["overview", "metrics", "screenshots", "workflows", "tests", "code-quality", "agent"]) {
      expect(parseUrlState(`?reviewTab=${tab}`).reviewTab).toBe(tab);
    }
  });

  test("ignores invalid reviewTab values", () => {
    expect(parseUrlState("?reviewTab=invalid").reviewTab).toBeUndefined();
    expect(parseUrlState("?reviewTab=").reviewTab).toBeUndefined();
  });

  test("parses anyFile=true", () => {
    expect(parseUrlState("?anyFile=true").anyFile).toBe(true);
  });

  test("ignores anyFile with non-true values", () => {
    expect(parseUrlState("?anyFile=false").anyFile).toBeUndefined();
    expect(parseUrlState("?anyFile=yes").anyFile).toBeUndefined();
  });

  test("parses agentRunId", () => {
    expect(parseUrlState("?agentRunId=run-abc-123").agentRunId).toBe("run-abc-123");
  });

  test("parses multiple params together", () => {
    const state = parseUrlState("?file=src%2Findex.ts&mode=source&staged=true&primary=review&reviewScenario=my-app&reviewTab=tests&anyFile=true&agentRunId=run-1");
    expect(state).toEqual({
      file: "src/index.ts",
      mode: "source",
      staged: true,
      primary: "review",
      reviewScenario: "my-app",
      reviewTab: "tests",
      anyFile: true,
      agentRunId: "run-1",
    });
  });
});

// --- buildUrlSearch ---

describe("buildUrlSearch", () => {
  test("returns empty string for empty state", () => {
    expect(buildUrlSearch({})).toBe("");
  });

  test("omits default values", () => {
    // mode=diff is the default
    expect(buildUrlSearch({ mode: "diff" })).toBe("");
    // panel=changes is the default (only "related" is written)
    expect(buildUrlSearch({ panel: "changes" })).toBe("");
    // primary=diff is the default
    expect(buildUrlSearch({ primary: "diff" })).toBe("");
    // reviewTab=overview is the default
    expect(buildUrlSearch({ reviewTab: "overview" })).toBe("");
    // staged=false is the default
    expect(buildUrlSearch({ staged: false })).toBe("");
    // anyFile=false is the default
    expect(buildUrlSearch({ anyFile: false })).toBe("");
  });

  test("includes non-default values", () => {
    expect(buildUrlSearch({ mode: "source" })).toContain("mode=source");
    expect(buildUrlSearch({ panel: "related" })).toContain("panel=related");
    expect(buildUrlSearch({ primary: "review" })).toContain("primary=review");
    expect(buildUrlSearch({ reviewTab: "tests" })).toContain("reviewTab=tests");
    expect(buildUrlSearch({ staged: true })).toContain("staged=true");
    expect(buildUrlSearch({ anyFile: true })).toContain("anyFile=true");
    expect(buildUrlSearch({ agentRunId: "run-1" })).toContain("agentRunId=run-1");
  });

  test("encodes file path (double-encoded since URLSearchParams also encodes)", () => {
    const result = buildUrlSearch({ file: "src/App.tsx" });
    // encodeURIComponent("src/App.tsx") = "src%2FApp.tsx", then URLSearchParams encodes the % → %25
    expect(result).toContain("file=src%252FApp.tsx");
    // Verify round-trip decode works
    expect(parseUrlState(result).file).toBe("src/App.tsx");
  });

  test("encodes reviewScenario (double-encoded)", () => {
    const result = buildUrlSearch({ reviewScenario: "my scenario" });
    expect(parseUrlState(result).reviewScenario).toBe("my scenario");
  });

  test("builds full state", () => {
    const state: UrlState = {
      file: "src/index.ts",
      mode: "source",
      staged: true,
      panel: "related",
      commit: "abc123",
      primary: "review",
      reviewScenario: "my-app",
      reviewTab: "tests",
      anyFile: true,
      agentRunId: "run-42",
    };
    const result = buildUrlSearch(state);
    expect(result).toContain("file=src%252Findex.ts");
    expect(result).toContain("mode=source");
    expect(result).toContain("staged=true");
    expect(result).toContain("panel=related");
    expect(result).toContain("commit=abc123");
    expect(result).toContain("primary=review");
    expect(result).toContain("reviewScenario=my-app");
    expect(result).toContain("reviewTab=tests");
    expect(result).toContain("anyFile=true");
    expect(result).toContain("agentRunId=run-42");
    expect(result.startsWith("?")).toBe(true);
  });
});

// --- Round-trip ---

describe("round-trip: parseUrlState → buildUrlSearch", () => {
  test("simple file URL round-trips", () => {
    const search = "?file=src%2FApp.tsx&mode=source";
    const state = parseUrlState(search);
    const rebuilt = buildUrlSearch(state);
    expect(parseUrlState(rebuilt)).toEqual(state);
  });

  test("changed file URL round-trips (no anyFile flag)", () => {
    const search = "?file=src%2FApp.tsx&staged=true";
    const state = parseUrlState(search);
    // No anyFile in state — this is a changed file
    expect(state.anyFile).toBeUndefined();
    const rebuilt = buildUrlSearch(state);
    expect(parseUrlState(rebuilt)).toEqual(state);
  });

  test("any-file URL round-trips", () => {
    const search = "?file=src%2FApp.tsx&mode=source&anyFile=true";
    const state = parseUrlState(search);
    expect(state.anyFile).toBe(true);
    const rebuilt = buildUrlSearch(state);
    expect(parseUrlState(rebuilt)).toEqual(state);
  });

  test("full state round-trips", () => {
    const original: UrlState = {
      file: "path/to/file.ts",
      mode: "full_diff",
      staged: true,
      panel: "related",
      commit: "deadbeef",
      primary: "review",
      reviewScenario: "scenario-name",
      reviewTab: "agent",
      anyFile: true,
      agentRunId: "run-abc",
    };
    const rebuilt = buildUrlSearch(original);
    const parsed = parseUrlState(rebuilt);
    expect(parsed).toEqual(original);
  });

  test("default-only state round-trips to empty", () => {
    const state: UrlState = { mode: "diff", panel: "changes", primary: "diff", reviewTab: "overview", anyFile: false };
    expect(buildUrlSearch(state)).toBe("");
  });

  test("agent run ID round-trips", () => {
    const state: UrlState = { primary: "review", reviewScenario: "my-app", reviewTab: "agent", agentRunId: "run-123" };
    const rebuilt = buildUrlSearch(state);
    expect(parseUrlState(rebuilt)).toEqual(state);
  });
});

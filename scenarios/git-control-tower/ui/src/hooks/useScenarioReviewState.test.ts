import { deepMerge, loadState, saveState, pruneOldEntries, validateTab, DEFAULT_STATE } from "./useScenarioReviewState";
import type { ScenarioReviewState, DeepPartial } from "./useScenarioReviewState";
import type { ReviewTab } from "./useUrlState";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/** Helper to create a stored entry JSON string. */
function storedEntry(state: Partial<ScenarioReviewState>, lastAccessed = Date.now()): string {
  return JSON.stringify({ version: 1, lastAccessed, state });
}

beforeEach(() => {
  localStorage.clear();
});

// ---------------------------------------------------------------------------
// deepMerge
// ---------------------------------------------------------------------------

describe("deepMerge", () => {
  test("shallow patch replaces top-level primitive", () => {
    const result = deepMerge(DEFAULT_STATE, { activeTab: "agent" });
    expect(result.activeTab).toBe("agent");
    // Other fields unchanged
    expect(result.agentRunId).toBeNull();
    expect(result.screenshots).toEqual(DEFAULT_STATE.screenshots);
  });

  test("nested patch merges without clobbering sibling keys", () => {
    const result = deepMerge(DEFAULT_STATE, { screenshots: { selectedPage: 3 } });
    expect(result.screenshots.selectedPage).toBe(3);
    // Sibling key preserved
    expect(result.screenshots.activePresetIndex).toBe(0);
  });

  test("null values are preserved", () => {
    const base = { ...DEFAULT_STATE, agentRunId: "run-123" };
    const result = deepMerge(base, { agentRunId: null } as DeepPartial<ScenarioReviewState>);
    expect(result.agentRunId).toBeNull();
  });

  test("array values replace rather than merge", () => {
    const result = deepMerge(DEFAULT_STATE, {
      workflows: { selectedModes: ["mutating", "destructive"] },
    });
    expect(result.workflows.selectedModes).toEqual(["mutating", "destructive"]);
    // viewRole sibling preserved
    expect(result.workflows.viewRole).toBe("capture");
  });

  test("multiple nested patches compose correctly", () => {
    let state = deepMerge(DEFAULT_STATE, { activeTab: "tests" as ReviewTab });
    state = deepMerge(state, { screenshots: { activePresetIndex: 2 } });
    state = deepMerge(state, { workflows: { viewRole: "baseline" } });

    expect(state.activeTab).toBe("tests");
    expect(state.screenshots.activePresetIndex).toBe(2);
    expect(state.screenshots.selectedPage).toBe(0);
    expect(state.workflows.viewRole).toBe("baseline");
    expect(state.workflows.selectedModes).toEqual(["observer"]);
  });

  test("empty patch returns a copy of base", () => {
    const result = deepMerge(DEFAULT_STATE, {} as DeepPartial<ScenarioReviewState>);
    expect(result).toEqual(DEFAULT_STATE);
    expect(result).not.toBe(DEFAULT_STATE); // different reference
  });

  test("does not mutate the base object", () => {
    const base = { ...DEFAULT_STATE, screenshots: { ...DEFAULT_STATE.screenshots } };
    deepMerge(base, { screenshots: { selectedPage: 5 } });
    expect(base.screenshots.selectedPage).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// loadState
// ---------------------------------------------------------------------------

describe("loadState", () => {
  test("returns DEFAULT_STATE when no entry exists", () => {
    expect(loadState("nonexistent")).toEqual(DEFAULT_STATE);
  });

  test("returns DEFAULT_STATE for empty slug", () => {
    expect(loadState("")).toEqual(DEFAULT_STATE);
  });

  test("deep-merges stored partial state with defaults (schema evolution)", () => {
    // Simulate an older schema that only has activeTab
    localStorage.setItem(
      "gct.reviewState.my-scenario",
      storedEntry({ activeTab: "agent" }),
    );
    const state = loadState("my-scenario");
    expect(state.activeTab).toBe("agent");
    // Missing fields filled from defaults
    expect(state.agentRunId).toBeNull();
    expect(state.screenshots).toEqual(DEFAULT_STATE.screenshots);
    expect(state.workflows).toEqual(DEFAULT_STATE.workflows);
    expect(state.codeQuality).toEqual(DEFAULT_STATE.codeQuality);
    expect(state.rules).toEqual(DEFAULT_STATE.rules);
  });

  test("returns defaults on corrupted JSON", () => {
    localStorage.setItem("gct.reviewState.bad", "not-json{{{");
    expect(loadState("bad")).toEqual(DEFAULT_STATE);
  });

  test("returns defaults when version is wrong", () => {
    localStorage.setItem(
      "gct.reviewState.v2",
      JSON.stringify({ version: 2, lastAccessed: Date.now(), state: { activeTab: "agent" } }),
    );
    expect(loadState("v2")).toEqual(DEFAULT_STATE);
  });

  test("returns defaults when state field is missing", () => {
    localStorage.setItem(
      "gct.reviewState.no-state",
      JSON.stringify({ version: 1, lastAccessed: Date.now() }),
    );
    expect(loadState("no-state")).toEqual(DEFAULT_STATE);
  });

  test("fully populated state loads correctly", () => {
    const full: ScenarioReviewState = {
      activeTab: "workflows",
      agentRunId: "run-abc",
      screenshots: { activePresetIndex: 2, selectedPage: 5 },
      workflows: { selectedModes: ["mutating"], viewRole: "baseline" },
      codeQuality: { view: "scenario" },
      rules: { jobId: "job-xyz" },
    };
    localStorage.setItem("gct.reviewState.full", storedEntry(full));
    expect(loadState("full")).toEqual(full);
  });
});

// ---------------------------------------------------------------------------
// saveState
// ---------------------------------------------------------------------------

describe("saveState", () => {
  test("round-trips correctly with loadState", () => {
    const state: ScenarioReviewState = {
      ...DEFAULT_STATE,
      activeTab: "agent",
      agentRunId: "run-456",
      workflows: { selectedModes: ["destructive"], viewRole: "baseline" },
    };
    saveState("roundtrip", state);
    expect(loadState("roundtrip")).toEqual(state);
  });

  test("stores version: 1 and lastAccessed timestamp", () => {
    const before = Date.now();
    saveState("ts-check", DEFAULT_STATE);
    const after = Date.now();

    const raw = localStorage.getItem("gct.reviewState.ts-check");
    expect(raw).not.toBeNull();
    const entry = JSON.parse(raw ?? "") as { version: number; lastAccessed: number; state: ScenarioReviewState };
    expect(entry.version).toBe(1);
    expect(entry.lastAccessed).toBeGreaterThanOrEqual(before);
    expect(entry.lastAccessed).toBeLessThanOrEqual(after);
  });

  test("does nothing for empty slug", () => {
    saveState("", DEFAULT_STATE);
    // No keys should have been written
    expect(localStorage.length).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// pruneOldEntries
// ---------------------------------------------------------------------------

describe("pruneOldEntries", () => {
  test("removes entries older than 30 days", () => {
    const oldTimestamp = Date.now() - 31 * 24 * 60 * 60 * 1000;
    localStorage.setItem(
      "gct.reviewState.old-scenario",
      JSON.stringify({ version: 1, lastAccessed: oldTimestamp, state: DEFAULT_STATE }),
    );
    localStorage.setItem(
      "gct.reviewState.recent-scenario",
      JSON.stringify({ version: 1, lastAccessed: Date.now(), state: DEFAULT_STATE }),
    );

    pruneOldEntries("current");

    expect(localStorage.getItem("gct.reviewState.old-scenario")).toBeNull();
    expect(localStorage.getItem("gct.reviewState.recent-scenario")).not.toBeNull();
  });

  test("never prunes the current scenario", () => {
    const oldTimestamp = Date.now() - 31 * 24 * 60 * 60 * 1000;
    localStorage.setItem(
      "gct.reviewState.current",
      JSON.stringify({ version: 1, lastAccessed: oldTimestamp, state: DEFAULT_STATE }),
    );

    pruneOldEntries("current");

    expect(localStorage.getItem("gct.reviewState.current")).not.toBeNull();
  });

  test("evicts oldest entries when over 50 count", () => {
    // Create 52 entries (+ current = 53 total)
    for (let i = 0; i < 52; i++) {
      localStorage.setItem(
        `gct.reviewState.scenario-${String(i).padStart(3, "0")}`,
        JSON.stringify({
          version: 1,
          lastAccessed: Date.now() - (52 - i) * 1000, // older entries first
          state: DEFAULT_STATE,
        }),
      );
    }

    pruneOldEntries("current");

    // Count remaining gct.reviewState.* entries (excluding current which doesn't exist)
    let count = 0;
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith("gct.reviewState.")) count++;
    }
    expect(count).toBeLessThanOrEqual(50);

    // The oldest entries should have been removed
    expect(localStorage.getItem("gct.reviewState.scenario-000")).toBeNull();
    expect(localStorage.getItem("gct.reviewState.scenario-001")).toBeNull();
    // The newest should remain
    expect(localStorage.getItem("gct.reviewState.scenario-051")).not.toBeNull();
  });

  test("removes corrupt entries", () => {
    localStorage.setItem("gct.reviewState.corrupt", "not-valid-json");
    localStorage.setItem(
      "gct.reviewState.valid",
      JSON.stringify({ version: 1, lastAccessed: Date.now(), state: DEFAULT_STATE }),
    );

    pruneOldEntries("current");

    expect(localStorage.getItem("gct.reviewState.corrupt")).toBeNull();
    expect(localStorage.getItem("gct.reviewState.valid")).not.toBeNull();
  });

  test("does not touch non-review-state keys", () => {
    localStorage.setItem("gct.sidebarWidth", "300");
    localStorage.setItem("gct.agent.defaultProfileId", "profile-1");

    pruneOldEntries("current");

    expect(localStorage.getItem("gct.sidebarWidth")).toBe("300");
    expect(localStorage.getItem("gct.agent.defaultProfileId")).toBe("profile-1");
  });
});

// ---------------------------------------------------------------------------
// validateTab
// ---------------------------------------------------------------------------

describe("validateTab", () => {
  const allTabs: ReviewTab[] = ["overview", "metrics", "screenshots", "workflows", "tests", "code-quality", "rules", "ai-provenance", "agent"];

  test("valid tab passes through unchanged", () => {
    expect(validateTab("agent", allTabs)).toBe("agent");
    expect(validateTab("tests", allTabs)).toBe("tests");
    expect(validateTab("overview", allTabs)).toBe("overview");
  });

  test("invalid tab falls back to overview", () => {
    const limited: ReviewTab[] = ["overview", "screenshots", "tests"];
    expect(validateTab("agent", limited)).toBe("overview");
    expect(validateTab("rules", limited)).toBe("overview");
  });

  test("empty visibleTabs does not trigger fallback (capabilities still loading)", () => {
    expect(validateTab("agent", [])).toBe("agent");
    expect(validateTab("code-quality", [])).toBe("code-quality");
  });

  test("overview is always valid if in visibleTabs", () => {
    expect(validateTab("overview", ["overview"])).toBe("overview");
  });
});

// ---------------------------------------------------------------------------
// switchScenario integration (using saveState + loadState)
// ---------------------------------------------------------------------------

describe("switchScenario flow (save + load)", () => {
  test("saves current state then loads next", () => {
    const scenarioAState: ScenarioReviewState = {
      ...DEFAULT_STATE,
      activeTab: "agent",
      agentRunId: "run-A",
    };
    const scenarioBState: ScenarioReviewState = {
      ...DEFAULT_STATE,
      activeTab: "tests",
      rules: { jobId: "job-B" },
    };

    // Pre-populate B
    saveState("scenario-b", scenarioBState);

    // Simulate switching from A to B
    saveState("scenario-a", scenarioAState);
    const loaded = loadState("scenario-b");

    expect(loaded.activeTab).toBe("tests");
    expect(loaded.rules.jobId).toBe("job-B");

    // Verify A was saved
    const savedA = loadState("scenario-a");
    expect(savedA.activeTab).toBe("agent");
    expect(savedA.agentRunId).toBe("run-A");
  });

  test("returns DEFAULT_STATE for a never-visited scenario", () => {
    saveState("old-scenario", { ...DEFAULT_STATE, activeTab: "rules" });
    const loaded = loadState("new-scenario");
    expect(loaded).toEqual(DEFAULT_STATE);
  });
});

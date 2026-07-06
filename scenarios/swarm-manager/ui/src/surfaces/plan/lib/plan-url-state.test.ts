import { describe, expect, it } from "vitest";
import { DEFAULT_PLAN_WINDOW_SECONDS } from "../stores/plan-data-store";
import {
  hasActiveFilters,
  readPlanStateFromUrl,
  writePlanStateToParams,
} from "./plan-url-state";

describe("readPlanStateFromUrl", () => {
  it("returns defaults for an empty query", () => {
    const state = readPlanStateFromUrl(new URLSearchParams());
    expect(state.filters.windowSeconds).toBe(DEFAULT_PLAN_WINDOW_SECONDS);
    expect(state.filters.statuses).toEqual([]);
    expect(state.viewMode).toBe("by-initiative");
    expect(state.showSnoozed).toBe(false);
    expect(state.goal).toBe("");
  });

  it("reads and round-trips the goal scope param", () => {
    const state = readPlanStateFromUrl(new URLSearchParams("goal=monetization-v1"));
    expect(state.goal).toBe("monetization-v1");
    const params = writePlanStateToParams(new URLSearchParams(), state);
    expect(params.get("goal")).toBe("monetization-v1");
    // Clearing the goal drops the param.
    const cleared = writePlanStateToParams(params, { ...state, goal: "" });
    expect(cleared.has("goal")).toBe(false);
  });

  it("reads valid values", () => {
    const params = new URLSearchParams(
      "status=running&lane=execute&owner_type=backlog&q=whisper&window_seconds=3600&view=by-phase&show_snoozed=1",
    );
    const state = readPlanStateFromUrl(params);
    expect(state.filters.statuses).toEqual(["running"]);
    expect(state.filters.lanes).toEqual(["execute"]);
    expect(state.filters.ownerTypes).toEqual(["backlog"]);
    expect(state.filters.q).toBe("whisper");
    expect(state.filters.windowSeconds).toBe(3600);
    expect(state.viewMode).toBe("by-phase");
    expect(state.showSnoozed).toBe(true);
  });

  it("drops invalid values back to defaults", () => {
    const params = new URLSearchParams(
      "status=bogus&lane=nope&owner_type=alien&window_seconds=42&view=sideways",
    );
    const state = readPlanStateFromUrl(params);
    expect(state.filters.statuses).toEqual([]);
    expect(state.filters.lanes).toEqual([]);
    expect(state.filters.ownerTypes).toEqual([]);
    expect(state.filters.windowSeconds).toBe(DEFAULT_PLAN_WINDOW_SECONDS);
    expect(state.viewMode).toBe("by-initiative");
  });
});

describe("writePlanStateToParams", () => {
  it("round-trips with read", () => {
    const original = readPlanStateFromUrl(new URLSearchParams(
      "status=failed&lane=review&q=x&window_seconds=3600&view=by-phase&show_snoozed=1",
    ));
    const written = writePlanStateToParams(new URLSearchParams(), original);
    expect(readPlanStateFromUrl(written)).toEqual(original);
  });

  it("omits defaults from the URL", () => {
    const state = readPlanStateFromUrl(new URLSearchParams());
    const written = writePlanStateToParams(new URLSearchParams(), state);
    expect(written.toString()).toBe("");
  });

  it("preserves unrelated params", () => {
    const state = readPlanStateFromUrl(new URLSearchParams("status=running"));
    const written = writePlanStateToParams(new URLSearchParams("focus=abc"), state);
    expect(written.get("focus")).toBe("abc");
    expect(written.get("status")).toBe("running");
  });
});

describe("hasActiveFilters", () => {
  it("is false at defaults and true with any filter", () => {
    const defaults = readPlanStateFromUrl(new URLSearchParams());
    expect(hasActiveFilters(defaults)).toBe(false);
    expect(hasActiveFilters(readPlanStateFromUrl(new URLSearchParams("q=x")))).toBe(true);
    expect(hasActiveFilters(readPlanStateFromUrl(new URLSearchParams("show_snoozed=1")))).toBe(true);
  });
});

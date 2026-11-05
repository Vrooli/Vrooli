/**
 * Reducer tests — pure logic, no DOM.
 *
 * The state shape for this hook is *the* contract for tab independence
 * (per-tab search, per-tab filters, per-tab sort), so these tests pin
 * the cross-tab isolation invariant we rely on in the rendered Sidebar.
 */

import { describe, it, expect } from "vitest";
import { sidebarReducer, createInitialState, type SidebarState, type SidebarAction } from "./useSidebarState";
import { DEFAULT_FILTERS } from "./types";

function reduce(actions: SidebarAction[], initial?: SidebarState): SidebarState {
  return actions.reduce((s, a) => sidebarReducer(s, a), initial ?? createInitialState());
}

describe("sidebarReducer", () => {
  it("starts on the Active tab with empty filters", () => {
    const s = createInitialState();
    expect(s.activeTab).toBe("active");
    expect(s.filters.active).toEqual(DEFAULT_FILTERS.active);
    expect(s.filters.history).toEqual(DEFAULT_FILTERS.history);
  });

  it("SET_TAB switches tabs without touching filters or search", () => {
    const s = reduce([
      { type: "SET_SEARCH", tab: "active", query: "hello" },
      { type: "SET_HISTORY_FILTERS", filters: { owner: "alice" } },
      { type: "SET_TAB", tab: "history" },
    ]);
    expect(s.activeTab).toBe("history");
    expect(s.searchQuery.active).toBe("hello");
    expect(s.filters.history.owner).toBe("alice");
  });

  it("per-tab search is isolated", () => {
    const s = reduce([
      { type: "SET_SEARCH", tab: "active", query: "active query" },
      { type: "SET_SEARCH", tab: "history", query: "history query" },
    ]);
    expect(s.searchQuery.active).toBe("active query");
    expect(s.searchQuery.history).toBe("history query");
  });

  it("per-tab filters are isolated — setting active doesn't touch history", () => {
    const s = reduce([
      { type: "SET_HISTORY_FILTERS", filters: { owner: "bob" } },
      { type: "SET_ACTIVE_FILTERS", filters: { owner: "alice" } },
    ]);
    expect(s.filters.active.owner).toBe("alice");
    expect(s.filters.history.owner).toBe("bob");
  });

  it("CLEAR_FILTERS only clears the targeted tab", () => {
    const s = reduce([
      { type: "SET_ACTIVE_FILTERS", filters: { owner: "alice", projectRoot: "/x" } },
      { type: "SET_HISTORY_FILTERS", filters: { owner: "bob" } },
      { type: "SET_SEARCH", tab: "active", query: "foo" },
      { type: "CLEAR_FILTERS", tab: "active" },
    ]);
    expect(s.filters.active).toEqual(DEFAULT_FILTERS.active);
    expect(s.filters.history.owner).toBe("bob");
    expect(s.searchQuery.active).toBe("");
  });

  it("toggling status filter accumulates pills", () => {
    const s = reduce([
      { type: "SET_ACTIVE_FILTERS", filters: { statuses: ["active"] } },
      { type: "SET_ACTIVE_FILTERS", filters: { statuses: ["active", "stopped"] } },
    ]);
    expect(s.filters.active.statuses).toEqual(["active", "stopped"]);
  });

  it("history sort field defaults to snapshotAt desc", () => {
    const s = createInitialState();
    expect(s.sorts.history.field).toBe("snapshotAt");
    expect(s.sorts.history.direction).toBe("desc");
  });

  it("active sort can flip direction independently of history", () => {
    const s = reduce([
      { type: "SET_ACTIVE_SORT", sort: { direction: "asc" } },
    ]);
    expect(s.sorts.active.direction).toBe("asc");
    expect(s.sorts.history.direction).toBe("desc");
  });

  it("SET_TAB on the same tab returns the same state reference", () => {
    const before = createInitialState();
    const after = sidebarReducer(before, { type: "SET_TAB", tab: "active" });
    expect(after).toBe(before);
  });
});

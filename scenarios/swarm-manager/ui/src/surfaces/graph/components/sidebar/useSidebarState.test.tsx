import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useSidebarState } from "./useSidebarState";

const STORAGE_KEY = "swarm-manager.sidebar.state.v1";

describe("useSidebarState", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("persists active tab, search, filters, and sort", () => {
    const { result } = renderHook(() => useSidebarState());

    act(() => {
      result.current[1]({ type: "SET_TAB", tab: "backlog" });
      result.current[1]({ type: "SET_SEARCH", query: "routing" });
      result.current[1]({ type: "SET_SEARCH_MODE", mode: "ai" });
      result.current[1]({ type: "SET_BACKLOG_FILTERS", filters: { kinds: ["fix"], showArchived: true } });
      result.current[1]({ type: "SET_SORT", tab: "backlog", sort: { field: "alphabetical", direction: "desc" } });
    });

    const stored = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "{}") as Record<string, unknown>;
    expect(stored).toMatchObject({
      activeTab: "backlog",
      searchQuery: "routing",
      searchMode: "ai",
      filters: {
        backlog: {
          kinds: ["fix"],
          showArchived: true,
        },
      },
      sorts: {
        backlog: {
          field: "alphabetical",
          direction: "desc",
        },
      },
    });
  });

  it("restores persisted state on mount", () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({
      activeTab: "executions",
      searchQuery: "review",
      searchMode: "plain",
      filters: {
        executions: {
          statuses: ["running"],
          modes: ["review"],
        },
      },
      sorts: {
        executions: {
          field: "recency",
          direction: "asc",
        },
      },
    }));

    const { result } = renderHook(() => useSidebarState());

    expect(result.current[0]).toMatchObject({
      activeTab: "executions",
      searchQuery: "review",
      searchMode: "plain",
      filters: {
        executions: {
          statuses: ["running"],
          modes: ["review"],
        },
      },
      sorts: {
        executions: {
          field: "recency",
          direction: "asc",
        },
      },
    });
  });
});

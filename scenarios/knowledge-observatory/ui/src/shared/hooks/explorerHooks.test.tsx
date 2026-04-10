import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useScenarioExplorer } from "./explorerHooks";
import * as api from "../services/documentationApi";

vi.mock("../services/documentationApi");

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe("useScenarioExplorer", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it("loads scenarios and selects the first one", async () => {
    vi.mocked(api.fetchScenarioSummaries).mockResolvedValue([
      {
        name: "alpha",
        path: "scenarios/alpha",
        doc_count: 3,
        health_score: 0.9,
        has_manifest: true,
        has_readme: true,
        last_modified: "2026-01-26T10:00:00Z",
      },
    ]);
    vi.mocked(api.fetchScenarioDocTree).mockResolvedValue({
      name: "alpha",
      path: "scenarios/alpha",
      type: "directory",
    });
    vi.mocked(api.fetchScenarioDocHealth).mockResolvedValue({
      scenario_name: "alpha",
      health_score: 0.9,
      total_docs: 3,
      misplaced_docs: [],
      missing_docs: [],
      extra_docs: [],
      warnings: [],
      can_auto_fix: false,
      fix_category: "none",
    });

    const { result } = renderHook(() => useScenarioExplorer(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.scenarios.length).toBe(1);
    });

    expect(result.current.selectedScenario).toBe("alpha");
    expect(result.current.docTree?.name).toBe("alpha");
  });
});

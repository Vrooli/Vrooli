import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { makeApiMocks, makeHealthResponse, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

import { fetchHealth } from "../../api/health";
import { recordLiveSearchHealth } from "../../lib/liveSearchHealth";
import { OpsPanel } from "./OpsPanel";

describe("OpsPanel", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders dependency rows from /health", async () => {
    vi.mocked(fetchHealth).mockResolvedValue(
      makeHealthResponse({
        dependencies: {
          searxng: { connected: true, latencyMs: 12, error: "", database: "" },
          database: { connected: false, latencyMs: 0, error: "down", database: "sqlite" },
        },
      }),
    );

    renderWithProviders(<OpsPanel />);

    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.ops.dependency)).toHaveLength(2);
    });
    expect(screen.getByText(strings.ops.dependencyConnected)).toBeInTheDocument();
    expect(screen.getByText(strings.ops.dependencyDisconnected)).toBeInTheDocument();
  });

  it("shows the empty last-query state before any live search", () => {
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
    renderWithProviders(<OpsPanel />);
    expect(screen.getByTestId(selectors.ops.lastQueryEmpty)).toBeInTheDocument();
  });

  it("surfaces the last live-search health signals", async () => {
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
    renderWithProviders(<OpsPanel />);

    recordLiveSearchHealth({
      query: "vrooli",
      cached: true,
      degraded: true,
      degradedReason: "budget exhausted",
      resultCount: 3,
      at: Date.now(),
    });

    await waitFor(() => {
      expect(screen.getByText(strings.ops.cachedYes)).toBeInTheDocument();
    });
    expect(screen.getByText(strings.ops.degradedYes)).toBeInTheDocument();
    expect(screen.getByText(/budget exhausted/)).toBeInTheDocument();
  });
});

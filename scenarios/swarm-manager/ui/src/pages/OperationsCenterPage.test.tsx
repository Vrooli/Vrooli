import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { describe, expect, it, beforeEach, beforeAll } from "vitest";
import { OperationsCenterPage } from "./OperationsCenterPage";
import { selectors } from "../consts/selectors";
import {
  installMatchMediaMock,
  installResizeObserverMock,
} from "../test-utils/browser";
import {
  setOperationsStoreService,
  resetOperationsStoreService,
  useOperationsStore,
} from "../stores/operations-store";
import type { OperationsView } from "../types/operations";
import type { IOperationsService } from "../services/operations-service";

beforeAll(() => {
  installResizeObserverMock();
  installMatchMediaMock();
});

function makeView(overrides: Partial<OperationsView> = {}): OperationsView {
  return {
    lanes: [
      { lane: "investigate", active: 0, capacity: 6, queue: 0 },
      { lane: "execute", active: 1, capacity: 3, queue: 0 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 0, maxDepth: 50 },
    activities: [
      {
        activityId: "a-1",
        runId: "run-1",
        ownerType: "initiative",
        ownerName: "auth-rewrite",
        ownerTitle: "Auth Rewrite",
        initiativeName: "auth-rewrite",
        purpose: "process",
        phaseKind: "execute",
        lane: "execute",
        status: "running",
        mode: "holistic-loop",
        phase: "execute",
        round: 4,
        requestedAt: "2026-05-02T01:00:00Z",
        runtimeSeconds: 180,
      },
    ],
    recentlyFinished: [],
    generatedAt: "2026-05-02T01:05:00Z",
    windowSeconds: 10800,
    ...overrides,
  };
}

let lastFilters: unknown = null;
let mockResponse: OperationsView = makeView();
let mockError: Error | null = null;

const fakeService: IOperationsService = {
  async fetchOperations(filters) {
    lastFilters = filters;
    if (mockError) throw mockError;
    return mockResponse;
  },
};

let locationCapture: ReturnType<typeof useLocation> | null = null;
function CaptureLocation() {
  locationCapture = useLocation();
  return null;
}

function renderPage(initial = "/operations") {
  return render(
    <MemoryRouter initialEntries={[initial]}>
      <Routes>
        <Route
          path="/operations"
          element={
            <>
              <CaptureLocation />
              <OperationsCenterPage />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  lastFilters = null;
  mockError = null;
  mockResponse = makeView();
  locationCapture = null;
  setOperationsStoreService(fakeService);
  useOperationsStore.getState().reset();
  return () => {
    resetOperationsStoreService();
  };
});

describe("OperationsCenterPage", () => {
  it("renders the page shell with header, filter bar, and body", async () => {
    renderPage();
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.page),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByTestId(selectors.operationsCenter.header),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.operationsCenter.filterBar),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.body),
      ).toBeInTheDocument();
    });
  });

  it("renders the by-initiative card after the first refresh", async () => {
    renderPage();
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.initiativeCard),
      ).toBeInTheDocument();
    });
    expect(screen.getByText("auth-rewrite")).toBeInTheDocument();
  });

  it("shows the empty state when activities, queue and recently-finished are all empty", async () => {
    mockResponse = makeView({
      activities: [],
      recentlyFinished: [],
      queue: { depth: 0, maxDepth: 50 },
    });
    renderPage();
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.emptyState),
      ).toBeInTheDocument();
    });
  });

  it("shows an error state when the first load fails", async () => {
    mockError = new Error("upstream offline");
    renderPage();
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.errorState),
      ).toBeInTheDocument();
    });
  });

  it("syncs filters from the URL on mount", async () => {
    renderPage("/operations?status=running&lane=execute&q=auth&window_seconds=3600");
    await waitFor(() => {
      expect(lastFilters).toMatchObject({
        statuses: ["running"],
        lanes: ["execute"],
        q: "auth",
        windowSeconds: 3600,
      });
    });
  });

  it("ignores invalid query values", async () => {
    renderPage("/operations?status=bogus&lane=cosmic&window_seconds=99999");
    await waitFor(() => {
      expect(lastFilters).toMatchObject({
        statuses: [],
        lanes: [],
        windowSeconds: 3 * 60 * 60,
      });
    });
  });

  it("propagates a status select change into a fetch call and the URL", async () => {
    renderPage();
    await waitFor(() => {
      expect(lastFilters).not.toBeNull();
    });

    await act(async () => {
      await userEvent.selectOptions(
        screen.getByLabelText(/status filter/i),
        "running",
      );
    });

    await waitFor(() => {
      expect((lastFilters as { statuses?: string[] }).statuses).toEqual(["running"]);
    });
    await waitFor(() => {
      expect(locationCapture?.search).toContain("status=running");
    });
  });

  it("clears filters on reset", async () => {
    renderPage("/operations?status=running");
    await waitFor(() => {
      expect((lastFilters as { statuses?: string[] }).statuses).toEqual(["running"]);
    });
    await userEvent.click(screen.getByRole("button", { name: /reset/i }));
    await waitFor(() => {
      expect((lastFilters as { statuses?: string[] }).statuses).toEqual([]);
    });
    await waitFor(() => {
      expect(locationCapture?.search ?? "").not.toContain("status=running");
    });
  });

  it("starts in by-initiative without polluting the URL", async () => {
    renderPage();
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.viewToggleByInitiative),
      ).toHaveAttribute("aria-selected", "true");
    });
    expect(locationCapture?.search ?? "").not.toContain("view=");
  });

  it("persists the view mode on the URL when toggled to by-phase", async () => {
    renderPage();
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.body),
      ).toBeInTheDocument();
    });
    await userEvent.click(
      screen.getByTestId(selectors.operationsCenter.viewToggleByPhase),
    );
    await waitFor(() => {
      expect(locationCapture?.search ?? "").toContain("view=by-phase");
    });
    expect(
      screen.getByTestId(selectors.operationsCenter.byPhaseBoard),
    ).toBeInTheDocument();
  });

  it("hydrates the by-phase view from the URL on mount", async () => {
    renderPage("/operations?view=by-phase");
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.byPhaseBoard),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByTestId(selectors.operationsCenter.viewToggleByPhase),
    ).toHaveAttribute("aria-selected", "true");
  });

  it("removes view= from the URL when toggling back to by-initiative", async () => {
    renderPage("/operations?view=by-phase");
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.operationsCenter.byPhaseBoard),
      ).toBeInTheDocument();
    });
    await userEvent.click(
      screen.getByTestId(selectors.operationsCenter.viewToggleByInitiative),
    );
    await waitFor(() => {
      expect(locationCapture?.search ?? "").not.toContain("view=");
    });
  });
});

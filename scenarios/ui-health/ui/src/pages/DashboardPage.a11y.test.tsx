import { describe, it, vi, beforeEach } from "vitest";

import { renderWithProviders, makeHealthResponse } from "../test-utils";
import { expectNoA11yViolations } from "../test-utils/a11y";
import { selectors } from "../consts/selectors";

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return {
    ...actual,
    fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
  };
});

vi.mock("../api/search", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/search")>();
  return {
    ...actual,
    searchStatus: vi.fn().mockResolvedValue({
      available: true,
      ollama: true,
      qdrant: true,
      indexedCount: 0,
      lastReconcileAt: "",
      lastReconcileOutcome: "",
    }),
  };
});

import { DashboardPage } from "./DashboardPage";

beforeEach(() => {
  window.localStorage.clear();
});

describe("DashboardPage accessibility", () => {
  it("has no axe violations once the health card loads", async () => {
    const { container, findByTestId } = renderWithProviders(<DashboardPage />);
    const card = await findByTestId(selectors.dashboard.apiStatus.card);
    // Wait for the card body to populate (post fetchHealth resolve).
    const { waitFor } = await import("@testing-library/react");
    await waitFor(() => {
      if (!card.querySelector("dl")) throw new Error("health card body not ready");
    });
    await expectNoA11yViolations(container);
  });
});

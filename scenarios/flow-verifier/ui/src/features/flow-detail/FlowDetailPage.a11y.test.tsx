import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, waitFor, screen } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  return {
    ...actual,
    fetchFlowDetail: vi.fn(),
    fetchRuns: vi.fn(),
  };
});

vi.mock("./StateGraph", () => ({
  StateGraph: () => <div role="img" aria-label="state-graph-stub" />,
}));

import { FlowDetailPage } from "./FlowDetailPage";

describe("FlowDetailPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    const { fetchFlowDetail, fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockResolvedValue({
      flowId: "notes.attachment-upload.ui",
      kind: "temporal",
      contractPath: "ui/src/features/notes/flow/flow.json",
      language: "ts",
      schemaVersion: 6,
      initialState: "a",
      states: [
        { id: "a", quint: "A", initial: true },
        { id: "b", quint: "B" },
      ],
      events: [{ id: "go", quint: "Go" }],
      transitions: [{ from: "a", event: "go", to: "b", wantError: false }],
      traces: [],
      invariants: [],
      model: { module: "M", seed: "0", maxSteps: 1, traceCount: 0, verify: { invariants: [] } },
      runtime: {},
      report: "",
    });
    vi.mocked(fetchRuns).mockResolvedValue([]);
  });

  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <Routes>
        <Route path="/flows/:flowId" element={<FlowDetailPage />} />
      </Routes>,
      { routerEntries: ["/flows/notes.attachment-upload.ui"] },
    );
    await waitFor(() =>
      // wait for fetch to resolve so the body actually mounts
      expect(screen.getByTestId("flow-detail-page")).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});

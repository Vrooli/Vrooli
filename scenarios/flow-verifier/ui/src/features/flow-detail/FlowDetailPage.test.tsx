/**
 * FlowDetailPage tests.
 *
 * Mocks the inventory API boundary and StateGraph so we exercise the
 * page's job — routing → fetch → tab switching → history rendering —
 * without dragging in the React Flow canvas.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  return {
    ...actual,
    fetchFlowDetail: vi.fn(),
    fetchRuns: vi.fn(),
  };
});

vi.mock("./StateGraph", () => ({
  StateGraph: ({ activeState }: { activeState?: string }) => (
    <div data-testid="state-graph-stub" data-active={activeState ?? ""} />
  ),
}));

import { FlowDetailPage } from "./FlowDetailPage";
import type { FlowDetail, RunRow } from "../../api/inventory";

const detail: FlowDetail = {
  flowId: "notes.attachment-upload.ui",
  contractPath: "ui/src/features/notes/flow/flow.json",
  language: "ts",
  schemaVersion: 6,
  initialState: "draft",
  states: [
    { id: "draft", quint: "Draft", initial: true },
    { id: "uploading", quint: "Uploading" },
    { id: "uploaded", quint: "Uploaded" },
  ],
  events: [{ id: "begin", quint: "Begin" }, { id: "complete", quint: "Complete" }],
  transitions: [
    { from: "draft", event: "begin", to: "uploading", wantError: false },
    { from: "uploading", event: "complete", to: "uploaded", wantError: false },
  ],
  traces: [
    {
      name: "happy-path",
      initial: "draft",
      steps: [
        { event: "begin", want: "uploading", wantError: false },
        { event: "complete", want: "uploaded", wantError: false },
      ],
    },
  ],
  invariants: [],
  report: "flow: notes.attachment-upload.ui\n",
};

const failingRun: RunRow = {
  id: "run-fail-1",
  flowId: "notes.attachment-upload.ui",
  flowPath: "ui/src/features/notes/flow/flow.json",
  root: ".",
  mode: "check",
  status: "failed",
  startedAt: "2026-05-10T11:59:58Z",
  finishedAt: "2026-05-10T12:00:00Z",
  durationMs: 2000,
  counterexample: JSON.stringify({
    states: [
      { state: "draft" },
      { state: "uploaded", event: "begin" },
    ],
  }),
};

const passingRun: RunRow = {
  ...failingRun,
  id: "run-pass-1",
  status: "passed",
  counterexample: undefined,
};

const renderAt = (route: string) =>
  renderWithProviders(
    <Routes>
      <Route path="/flows/:flowId" element={<FlowDetailPage />} />
      <Route path="/flows" element={<FlowDetailPage />} />
    </Routes>,
    { routerEntries: [route] },
  );

describe("FlowDetailPage", () => {
  beforeEach(async () => {
    const { fetchFlowDetail, fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockReset();
    vi.mocked(fetchRuns).mockReset();
  });
  afterEach(() => cleanup());

  it("shows missing-id state when no :flowId is in the route", () => {
    renderAt("/flows");
    expect(screen.getByTestId("flow-detail-missing")).toBeInTheDocument();
  });

  it("renders the loading state while the flow detail is in flight", async () => {
    const { fetchFlowDetail } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockReturnValue(new Promise(() => {}));
    renderAt("/flows/notes.attachment-upload.ui");
    expect(screen.getByTestId("flow-detail-loading")).toBeInTheDocument();
  });

  it("renders the error state and a back link when fetch fails", async () => {
    const { fetchFlowDetail } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockRejectedValue(new Error("boom"));
    renderAt("/flows/notes.attachment-upload.ui");
    await waitFor(() =>
      expect(screen.getByTestId("flow-detail-error")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("flow-detail-back")).toHaveAttribute("href", "/flows");
  });

  it("renders the header, defaults to the Graph tab, and shows StateGraph", async () => {
    const { fetchFlowDetail } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockResolvedValue(detail);
    renderAt("/flows/notes.attachment-upload.ui");
    await waitFor(() =>
      expect(screen.getByTestId("flow-detail-page")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("flow-detail-id")).toHaveTextContent(
      "notes.attachment-upload.ui",
    );
    expect(screen.getByTestId("flow-detail-lang")).toHaveTextContent("ts");
    expect(screen.getByTestId("flow-detail-tab-graph")).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByTestId("state-graph-stub")).toBeInTheDocument();
  });

  it("switches to the Traces tab and mounts the trace player", async () => {
    const { fetchFlowDetail } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockResolvedValue(detail);
    const user = userEvent.setup();
    renderAt("/flows/notes.attachment-upload.ui");
    await waitFor(() =>
      expect(screen.getByTestId("flow-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("flow-detail-tab-traces"));
    expect(screen.getByTestId("flow-detail-tab-traces")).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByTestId("trace-player")).toBeInTheDocument();
  });

  it("renders the history list and counterexample diff on Inspect", async () => {
    const { fetchFlowDetail, fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockResolvedValue(detail);
    vi.mocked(fetchRuns).mockResolvedValue([failingRun, passingRun]);
    const user = userEvent.setup();
    renderAt("/flows/notes.attachment-upload.ui");
    await waitFor(() =>
      expect(screen.getByTestId("flow-detail-page")).toBeInTheDocument(),
    );

    await user.click(screen.getByTestId("flow-detail-tab-history"));
    await waitFor(() =>
      expect(screen.getByTestId("flow-history")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("flow-history-row-run-fail-1")).toBeInTheDocument();
    expect(screen.getByTestId("flow-history-row-run-pass-1")).toBeInTheDocument();

    // passed runs don't expose an Inspect action; failing ones do.
    expect(screen.queryByTestId("flow-history-inspect-run-pass-1")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("flow-history-inspect-run-fail-1"));
    expect(screen.getByTestId("inspector-desktop")).toBeInTheDocument();
    expect(screen.getByTestId("ce-diff")).toBeInTheDocument();
  });

  it("shows an empty hint when no runs exist yet for the flow", async () => {
    const { fetchFlowDetail, fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchFlowDetail).mockResolvedValue(detail);
    vi.mocked(fetchRuns).mockResolvedValue([]);
    const user = userEvent.setup();
    renderAt("/flows/notes.attachment-upload.ui");
    await waitFor(() =>
      expect(screen.getByTestId("flow-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("flow-detail-tab-history"));
    await waitFor(() =>
      expect(screen.getByTestId("flow-history-empty")).toBeInTheDocument(),
    );
  });
});

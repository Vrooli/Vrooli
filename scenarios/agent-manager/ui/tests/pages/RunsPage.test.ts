import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { RunsPage } from "../../src/pages/RunsPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { RunStatus } from "../../src/types.js";
import { makeRun } from "../testutil/runs.js";

const state = vi.hoisted(() => ({
  useRunsPageState: vi.fn(),
  useSelectedRunController: vi.fn(),
}));

vi.mock("../../src/hooks/useRunsPageState.js", () => ({ useRunsPageState: state.useRunsPageState }));
vi.mock("../../src/hooks/useSelectedRunController.js", () => ({ useSelectedRunController: state.useSelectedRunController }));
vi.mock("../../src/components/RunDetail.js", () => ({ RunDetail: ({ taskTitle }: { taskTitle: string }) => createElement("div", { "data-testid": "run-detail" }, taskTitle) }));

function noop() {}

function setupControllerState() {
  state.useRunsPageState.mockReturnValue({
    searchQuery: "", setSearchQuery: noop, statusFilter: "all", setStatusFilter: noop,
    sortBy: "newest", setSortBy: noop, selectionMode: false, setSelectionMode: noop,
    selectedRunIds: new Set(), setSelectedRunIds: noop, investigateModalOpen: false,
    setInvestigateModalOpen: noop, investigateLoading: false, setInvestigateLoading: noop,
    investigateError: null, setInvestigateError: noop, toggleSelectionMode: noop,
    clearSelection: noop, handleRunCheckboxChange: noop,
  });
  state.useSelectedRunController.mockReturnValue({
    selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [],
    eventsLoading: false, diffLoading: false, resolvedRuns: [], getTaskById: () => undefined,
    getTaskTitle: () => "Unknown Task", loadRunDetails: noop,
  });
}

afterEach(() => vi.resetAllMocks());

test("RunsPage exposes its empty, API-error, refresh, and no-selection states", () => {
  setupControllerState();
  const refresh = vi.fn();
  const runEventStore = { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } };
  renderWithProviders(createElement(RunsPage, {
    runs: [], tasks: [], profiles: [], loading: false, error: "Runs API unavailable", onRefresh: refresh,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: runEventStore as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }));
  assert.ok(screen.getByText("No Runs Yet"));
  assert.ok(screen.getByText("Start a run from the Tasks tab"));
  assert.ok(screen.getByText("Runs API unavailable"));
  assert.ok(screen.getByText("Select a run to view details"));
  fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
  assert.equal(refresh.mock.calls.length, 1);
});

test("RunsPage wires a selected review run into its detail surface", () => {
  setupControllerState();
  const selected = makeRun({ id: "review-run", taskId: "task-1", status: RunStatus.NEEDS_REVIEW });
  state.useSelectedRunController.mockReturnValue({
    selectedRun: selected, setSelectedRun: noop, selectedRunId: selected.id, diff: null, events: [],
    eventsLoading: false, diffLoading: false, resolvedRuns: [selected], getTaskById: () => undefined,
    getTaskTitle: () => "Review task", loadRunDetails: noop,
  });
  const runEventStore = { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } };
  renderWithProviders(createElement(RunsPage, {
    runs: [selected], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: runEventStore as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs/review-run?tab=diff"] });
  assert.equal(screen.getByTestId("run-detail").textContent, "Review task");
  assert.equal(screen.getAllByText("Review task").length, 2);
});

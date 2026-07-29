import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { RunsPage } from "../../src/pages/RunsPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { ApprovalState, ExecutionMode, RunStatus } from "../../src/types.js";
import { makeRun } from "../testutil/runs.js";

const state = vi.hoisted(() => ({
  useRunsPageState: vi.fn(),
  useSelectedRunController: vi.fn(),
}));

vi.mock("../../src/hooks/useRunsPageState.js", () => ({ useRunsPageState: state.useRunsPageState }));
vi.mock("../../src/hooks/useSelectedRunController.js", () => ({ useSelectedRunController: state.useSelectedRunController }));
vi.mock("../../src/components/RunDetail.js", () => ({ RunDetail: ({ taskTitle, run, onApplyInvestigation, onInvestigate, onRetry, onApprove, onReject, onPartialApprove, onContinue, onDeleteMessage, onStop, onDelete }: {
  taskTitle: string; run: { id: string }; onApplyInvestigation: (id: string) => void; onInvestigate: (id: string) => void;
  onRetry: (run: { id: string }) => Promise<unknown>; onApprove: (request: unknown) => Promise<unknown>; onReject: (request: unknown) => Promise<unknown>;
  onPartialApprove: (fileIds: string[], actor?: string, message?: string) => Promise<unknown>; onContinue: (message: string, attachmentIds?: string[]) => Promise<unknown>;
  onDeleteMessage: (eventId: string) => Promise<unknown>; onStop: (run: { id: string }) => Promise<unknown>; onDelete: (run: { id: string }) => void;
}) => createElement("div", { "data-testid": "run-detail" }, taskTitle,
  createElement("button", { onClick: () => onApplyInvestigation(run.id) }, "Apply from detail"),
  createElement("button", { onClick: () => onInvestigate(run.id) }, "Investigate from detail"),
  createElement("button", { onClick: () => onRetry(run) }, "Retry from detail"),
  createElement("button", { onClick: () => onApprove({ actor: "reviewer" }) }, "Approve from detail"),
  createElement("button", { onClick: () => onReject({ reason: "needs work" }) }, "Reject from detail"),
  createElement("button", { onClick: () => onPartialApprove(["file-1"], "reviewer", "keep this") }, "Partial approve from detail"),
  createElement("button", { onClick: () => onContinue("continue carefully", ["attachment-1"]) }, "Continue from detail"),
  createElement("button", { onClick: () => onDeleteMessage("message-1") }, "Delete message from detail"),
  createElement("button", { onClick: () => onStop(run) }, "Stop from detail"),
  createElement("button", { onClick: () => onDelete(run) }, "Delete from detail"),
) }));

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
	assert.match(screen.getByTestId("run-detail").textContent ?? "", /Review task/);
  assert.equal(screen.getAllByText("Review task").length, 2);
});

test("RunsPage performs list lifecycle actions and preserves the deliberate delete confirmation", async () => {
  setupControllerState();
  const run = makeRun({
    id: "lifecycle-run",
    taskId: "task-1",
    status: RunStatus.RUNNING,
    actions: { canStop: true, canDelete: true, canResumeFromFailure: true },
  });
  const stopped = vi.fn().mockResolvedValue(undefined);
  const deleted = vi.fn().mockResolvedValue(undefined);
  const refresh = vi.fn();
  const toggleSelectionMode = vi.fn();
  state.useRunsPageState.mockReturnValue({
    ...state.useRunsPageState.mock.results[0]?.value,
    searchQuery: "", setSearchQuery: noop, statusFilter: "all", setStatusFilter: noop,
    sortBy: "newest", setSortBy: noop, selectionMode: true, setSelectionMode: noop,
    selectedRunIds: new Set([run.id]), setSelectedRunIds: noop, investigateModalOpen: false,
    setInvestigateModalOpen: noop, investigateLoading: false, setInvestigateLoading: noop,
    investigateError: null, setInvestigateError: noop, toggleSelectionMode,
    clearSelection: noop, handleRunCheckboxChange: noop,
  });
  state.useSelectedRunController.mockReturnValue({
    selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [],
    eventsLoading: false, diffLoading: false, resolvedRuns: [run], getTaskById: () => undefined,
    getTaskTitle: () => "Lifecycle task", loadRunDetails: noop,
  });
  const runEventStore = { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } };
  vi.stubGlobal("confirm", vi.fn(() => true));
  renderWithProviders(createElement(RunsPage, {
    runs: [run], tasks: [], profiles: [], loading: false, error: null, onRefresh: refresh,
    onStopRun: stopped, onDeleteRun: deleted, onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: runEventStore as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });

  fireEvent.click(screen.getByRole("button", { name: "Stop run Lifecycle task" }));
  await vi.waitFor(() => assert.equal(stopped.mock.calls[0]?.[0], run.id));
  assert.equal(refresh.mock.calls.length, 1);
  fireEvent.click(screen.getByRole("button", { name: "Delete run Lifecycle task" }));
  assert.ok(screen.getByRole("heading", { name: "Delete Run" }));
  fireEvent.click(screen.getByRole("button", { name: "Delete" }));
  await vi.waitFor(() => assert.equal(deleted.mock.calls[0]?.[0], run.id));
  assert.equal(toggleSelectionMode.mock.calls.length, 0);
});

test("RunsPage launches a batch investigation from explicit selected source runs", async () => {
  const source = makeRun({ id: "source-run", taskId: "task-1", actions: { ...makeRun().actions, canInvestigate: true } });
  const created = makeRun({ id: "investigation-run" });
  const setInvestigateModalOpen = vi.fn(); const clearSelection = vi.fn(); const setSelectionMode = vi.fn(); const snapshot = vi.fn();
  state.useRunsPageState.mockReturnValue({
    searchQuery: "", setSearchQuery: noop, statusFilter: "all", setStatusFilter: noop, sortBy: "newest", setSortBy: noop,
    selectionMode: true, setSelectionMode, selectedRunIds: new Set([source.id]), setSelectedRunIds: noop,
    investigateModalOpen: true, setInvestigateModalOpen, investigateLoading: false, setInvestigateLoading: noop,
    investigateError: null, setInvestigateError: noop, toggleSelectionMode: noop, clearSelection, handleRunCheckboxChange: noop,
  });
  state.useSelectedRunController.mockReturnValue({ selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false, resolvedRuns: [source], getTaskById: () => undefined, getTaskTitle: () => "Source task", loadRunDetails: noop });
  const investigate = vi.fn(async () => created);
  renderWithProviders(createElement(RunsPage, {
    runs: [source], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(), onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(),
    onInvestigateRuns: investigate, onApplyInvestigation: vi.fn(), onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: snapshot, eventsGapFilled: noop } } as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });
  fireEvent.click(screen.getByRole("radio", { name: /Deep/ }));
  fireEvent.click(screen.getByRole("button", { name: /Agent crashed/ }));
  fireEvent.click(screen.getByRole("button", { name: "Start Investigation" }));
  await vi.waitFor(() => assert.equal(investigate.mock.calls.length, 1));
  assert.deepEqual(investigate.mock.calls[0]?.[0], ["source-run"]); assert.equal(investigate.mock.calls[0]?.[2], "deep");
  assert.match(investigate.mock.calls[0]?.[1] ?? "", /agent crashed/i);
  assert.deepEqual(snapshot.mock.calls, [[created]]); assert.deepEqual(clearSelection.mock.calls, [[]]);
  assert.deepEqual(setInvestigateModalOpen.mock.calls.at(-1), [false]);
});

test("RunsPage keeps an investigation modal open and explains a create failure", async () => {
  const source = makeRun({ id: "source-run", taskId: "task-1", actions: { ...makeRun().actions, canInvestigate: true } });
  const setInvestigateModalOpen = vi.fn();
  const setInvestigateError = vi.fn();
  state.useRunsPageState.mockReturnValue({
    searchQuery: "", setSearchQuery: noop, statusFilter: "all", setStatusFilter: noop, sortBy: "newest", setSortBy: noop,
    selectionMode: true, setSelectionMode: noop, selectedRunIds: new Set([source.id]), setSelectedRunIds: noop,
    investigateModalOpen: true, setInvestigateModalOpen, investigateLoading: false, setInvestigateLoading: noop,
    investigateError: null, setInvestigateError, toggleSelectionMode: noop, clearSelection: noop, handleRunCheckboxChange: noop,
  });
  state.useSelectedRunController.mockReturnValue({ selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false, resolvedRuns: [source], getTaskById: () => undefined, getTaskTitle: () => "Source task", loadRunDetails: noop });
  const investigate = vi.fn(async () => { throw new Error("investigation runner unavailable"); });
  renderWithProviders(createElement(RunsPage, {
    runs: [source], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(), onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(),
    onInvestigateRuns: investigate, onApplyInvestigation: vi.fn(), onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });
  fireEvent.click(screen.getByRole("button", { name: "Start Investigation" }));
  await vi.waitFor(() => assert.equal(investigate.mock.calls.length, 1));
  assert.deepEqual(setInvestigateError.mock.calls.at(-1), ["investigation runner unavailable"]);
  assert.equal(setInvestigateModalOpen.mock.calls.length, 0);
});

test("RunsPage resumes a failed run with explicit operator guidance and updates durable state", async () => {
  const failed = makeRun({ id: "failed-run", taskId: "task-1", actions: { ...makeRun().actions, canResumeFromFailure: true } });
  const resumed = makeRun({ id: "resumed-run" }); const snapshot = vi.fn(); const refresh = vi.fn();
  setupControllerState();
  state.useSelectedRunController.mockReturnValue({ selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false, resolvedRuns: [failed], getTaskById: () => undefined, getTaskTitle: () => "Failed task", loadRunDetails: noop });
  const resume = vi.fn(async () => resumed);
  renderWithProviders(createElement(RunsPage, {
    runs: [failed], tasks: [], profiles: [], loading: false, error: null, onRefresh: refresh,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(), onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: resume, onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: snapshot, eventsGapFilled: noop } } as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });
  fireEvent.click(screen.getByRole("button", { name: "Resume run Failed task from failure" }));
  assert.ok(screen.getByRole("heading", { name: "Resume from Failure" }));
  fireEvent.change(screen.getByLabelText("Additional Guidance (optional)"), { target: { value: "  use the existing migration  " } });
  fireEvent.click(screen.getByRole("button", { name: "Resume Run" }));
  await vi.waitFor(() => assert.deepEqual(resume.mock.calls, [["failed-run", "use the existing migration", undefined]]));
  assert.deepEqual(snapshot.mock.calls, [[resumed]]); assert.equal(refresh.mock.calls.length, 1);
});

test("RunsPage opens detail-originated apply investigation and hands off the created apply run", async () => {
  const investigation = makeRun({ id: "investigation-run", status: RunStatus.COMPLETE });
  const created = makeRun({ id: "apply-run" }); const snapshot = vi.fn();
  setupControllerState();
  state.useSelectedRunController.mockReturnValue({ selectedRun: investigation, setSelectedRun: noop, selectedRunId: investigation.id, diff: null, events: [], eventsLoading: false, diffLoading: false, resolvedRuns: [investigation], getTaskById: () => undefined, getTaskTitle: () => "Investigation", loadRunDetails: noop });
  const apply = vi.fn(async () => created);
  renderWithProviders(createElement(RunsPage, {
    runs: [investigation], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(), onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: apply,
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: snapshot, eventsGapFilled: noop } } as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs/investigation-run"] });
  fireEvent.click(screen.getByRole("button", { name: "Apply from detail" }));
  assert.ok(screen.getByRole("heading", { name: "Apply Investigation Recommendations" }));
  fireEvent.change(screen.getByLabelText("Additional Context for Apply Agent"), { target: { value: "  apply evidence fix  " } });
  fireEvent.click(screen.getByRole("button", { name: "Apply Investigation" }));
  await vi.waitFor(() => assert.deepEqual(apply.mock.calls, [["investigation-run", [], "apply evidence fix", undefined]]));
  assert.deepEqual(snapshot.mock.calls, [[created]]);
});

test("RunsPage presents distinct status indicators, including rejected approval", () => {
  setupControllerState();
  const runs = [
    makeRun({ id: "complete", status: RunStatus.COMPLETE }),
    makeRun({ id: "failed", status: RunStatus.FAILED }),
    makeRun({ id: "running", status: RunStatus.RUNNING }),
    makeRun({ id: "review", status: RunStatus.NEEDS_REVIEW }),
    makeRun({ id: "cancelled", status: RunStatus.CANCELLED, approvalState: ApprovalState.REJECTED }),
  ];
  state.useSelectedRunController.mockReturnValue({ selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false, resolvedRuns: runs, getTaskById: () => undefined, getTaskTitle: (id: string) => id, loadRunDetails: noop });
  renderWithProviders(createElement(RunsPage, {
    runs, tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(), onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(), onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });
  for (const title of ["Complete", "Failed", "Running", "Needs Review", "Rejected"]) {
    assert.ok(screen.getByTitle(title));
  }
});

test("RunsPage retains the delete dialog and reports an API failure", async () => {
  setupControllerState();
  const run = makeRun({ id: "undeletable", taskId: "task-1", actions: { ...makeRun().actions, canDelete: true } });
  state.useSelectedRunController.mockReturnValue({ selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false, resolvedRuns: [run], getTaskById: () => undefined, getTaskTitle: () => "Undeletable task", loadRunDetails: noop });
  renderWithProviders(createElement(RunsPage, {
    runs: [run], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(async () => { throw new Error("run history is retained"); }), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(), onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(), onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never, wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });
  fireEvent.click(screen.getByRole("button", { name: "Delete run Undeletable task" }));
  fireEvent.click(screen.getByRole("button", { name: "Delete" }));
  await vi.waitFor(() => assert.ok(screen.getByText("run history is retained")));
  assert.ok(screen.getByRole("heading", { name: "Delete Run" }));
});

test("RunsPage propagates detail actions and refreshes its event snapshot after transcript mutations", async () => {
  const run = makeRun({ id: "detail-run", taskId: "task-1", actions: { ...makeRun().actions, canStop: true, canDelete: true } });
  const retryCreated = makeRun({ id: "retry-created" });
  const continued = makeRun({ ...run, id: run.id });
  const setSelectedRun = vi.fn();
  const loadRunDetails = vi.fn();
  const refresh = vi.fn();
  const snapshot = vi.fn();
  const eventsGapFilled = vi.fn();
  const getEvents = vi.fn(async () => []);
  const approve = vi.fn(async () => ({ remaining: 0 }));
  const reject = vi.fn(async () => undefined);
  const partialApprove = vi.fn(async () => ({ remaining: 0 }));
  const retry = vi.fn(async () => retryCreated);
  const continueRun = vi.fn(async () => continued);
  const deleteMessage = vi.fn(async () => undefined);
  const stop = vi.fn(async () => undefined);
  setupControllerState();
  state.useSelectedRunController.mockReturnValue({
    selectedRun: run, setSelectedRun, selectedRunId: run.id, diff: null, events: [], eventsLoading: false, diffLoading: false,
    resolvedRuns: [run], getTaskById: () => undefined, getTaskTitle: () => "Detail task", loadRunDetails,
  });
  vi.stubGlobal("confirm", vi.fn(() => true));
  renderWithProviders(createElement(RunsPage, {
    runs: [run], tasks: [], profiles: [], loading: false, error: null, onRefresh: refresh,
    onStopRun: stop, onDeleteRun: vi.fn(), onRetryRun: retry, onGetRun: vi.fn(), onGetEvents: getEvents, onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: approve, onRejectRun: reject, onPartialApproveRun: partialApprove, onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: continueRun, onDeleteRunMessage: deleteMessage,
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: { [run.id]: 7n } }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: snapshot, eventsGapFilled } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs/detail-run"] });

  fireEvent.click(screen.getByRole("button", { name: "Retry from detail" }));
  await vi.waitFor(() => assert.deepEqual(retry.mock.calls, [[run]]));
  assert.deepEqual(snapshot.mock.calls[0], [retryCreated]);
  assert.deepEqual(loadRunDetails.mock.calls, [[retryCreated]]);

  fireEvent.click(screen.getByRole("button", { name: "Approve from detail" }));
  fireEvent.click(screen.getByRole("button", { name: "Reject from detail" }));
  fireEvent.click(screen.getByRole("button", { name: "Partial approve from detail" }));
  await vi.waitFor(() => assert.deepEqual(approve.mock.calls, [[run.id, { actor: "reviewer" }]]));
  assert.deepEqual(reject.mock.calls, [[run.id, { reason: "needs work" }]]);
  assert.deepEqual(partialApprove.mock.calls, [[run.id, ["file-1"], "reviewer", "keep this"]]);
  assert.ok(setSelectedRun.mock.calls.filter(([value]) => value === null).length >= 3);
  assert.equal(refresh.mock.calls.length, 3);

  fireEvent.click(screen.getByRole("button", { name: "Continue from detail" }));
  fireEvent.click(screen.getByRole("button", { name: "Delete message from detail" }));
  await vi.waitFor(() => assert.deepEqual(continueRun.mock.calls, [[run.id, "continue carefully", ["attachment-1"]]]));
  assert.deepEqual(deleteMessage.mock.calls, [[run.id, "message-1"]]);
  assert.deepEqual(snapshot.mock.calls.at(-1), [continued]);
  assert.deepEqual(getEvents.mock.calls, [
    [run.id, { afterSequence: 7n }],
    [run.id, { afterSequence: 7n }],
  ]);
  await vi.waitFor(() => assert.equal(eventsGapFilled.mock.calls.length, 2));
  assert.deepEqual(eventsGapFilled.mock.calls, [[run.id, []], [run.id, []]]);

  fireEvent.click(screen.getByRole("button", { name: "Stop from detail" }));
  await vi.waitFor(() => assert.deepEqual(stop.mock.calls, [[run.id]]));
  assert.equal(refresh.mock.calls.length, 4);
  fireEvent.click(screen.getByRole("button", { name: "Delete from detail" }));
  assert.ok(screen.getByRole("heading", { name: "Delete Run" }));
});

test("RunsPage filters, sorts, and forwards row selection using the visible run order", () => {
  const oldest = makeRun({ id: "oldest-complete", taskId: "alpha-task", status: RunStatus.COMPLETE });
  const newest = makeRun({ id: "newest-complete", taskId: "beta-task", status: RunStatus.COMPLETE });
  const excluded = makeRun({ id: "failed-run", taskId: "beta-task", status: RunStatus.FAILED });
  const checkboxChange = vi.fn();
  const loadRunDetails = vi.fn();
  state.useRunsPageState.mockReturnValue({
    searchQuery: "beta", setSearchQuery: noop,
    statusFilter: String(RunStatus.COMPLETE), setStatusFilter: noop,
    sortBy: "oldest", setSortBy: noop,
    selectionMode: true, setSelectionMode: noop,
    selectedRunIds: new Set(), setSelectedRunIds: noop,
    investigateModalOpen: false, setInvestigateModalOpen: noop,
    investigateLoading: false, setInvestigateLoading: noop,
    investigateError: null, setInvestigateError: noop,
    toggleSelectionMode: noop, clearSelection: noop, handleRunCheckboxChange: checkboxChange,
  });
  state.useSelectedRunController.mockReturnValue({
    selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [],
    eventsLoading: false, diffLoading: false, resolvedRuns: [newest, excluded, oldest],
    getTaskById: () => undefined,
    getTaskTitle: (taskId: string) => taskId === "beta-task" ? "Beta migration" : "Alpha migration",
    loadRunDetails,
  });
  renderWithProviders(createElement(RunsPage, {
    runs: [newest, excluded, oldest], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });

  assert.ok(screen.getByText("Beta migration"));
  assert.equal(screen.queryByText("Alpha migration"), null);
  assert.equal(screen.queryByText("failed-run"), null);
  const row = screen.getByRole("button", { name: /Beta migration/ });
  fireEvent.click(row);
  assert.deepEqual(loadRunDetails.mock.calls.at(-1), [newest]);

  const checkbox = screen.getByRole("checkbox");
  fireEvent.click(checkbox, { shiftKey: true });
  assert.deepEqual(checkboxChange.mock.calls, [[newest.id, 0, true, [newest]]]);
});

test("RunsPage keeps the destructive confirmation open and reports a delete failure", async () => {
  setupControllerState();
  const run = makeRun({ id: "undeletable-run", taskId: "task-1", actions: { ...makeRun().actions, canDelete: true } });
  state.useSelectedRunController.mockReturnValue({
    selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false,
    resolvedRuns: [run], getTaskById: () => undefined, getTaskTitle: () => "Protected evidence", loadRunDetails: noop,
  });
  const remove = vi.fn(async () => { throw new Error("retention policy prevented deletion"); });
  renderWithProviders(createElement(RunsPage, {
    runs: [run], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: remove, onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });

  fireEvent.click(screen.getByRole("button", { name: "Delete run Protected evidence" }));
  fireEvent.click(screen.getByRole("button", { name: "Delete" }));
  await vi.waitFor(() => assert.deepEqual(remove.mock.calls, [[run.id]]));
  assert.ok(screen.getByRole("heading", { name: "Delete Run" }));
  await vi.waitFor(() => assert.ok(screen.getByText("retention policy prevented deletion")));
});

test("RunsPage retains apply and resume workflows when their creation requests fail", async () => {
  const investigation = makeRun({ id: "investigation-run", status: RunStatus.COMPLETE });
  const failed = makeRun({ id: "failed-run", taskId: "task-2", actions: { ...makeRun().actions, canResumeFromFailure: true } });
  setupControllerState();
  state.useSelectedRunController.mockReturnValue({
    selectedRun: investigation, setSelectedRun: noop, selectedRunId: investigation.id, diff: null, events: [], eventsLoading: false, diffLoading: false,
    resolvedRuns: [investigation, failed], getTaskById: () => undefined, getTaskTitle: (id: string) => id === "task-2" ? "Recoverable task" : "Investigation", loadRunDetails: noop,
  });
  const apply = vi.fn(async () => { throw new Error("apply runner unavailable"); });
  const resume = vi.fn(async () => { throw new Error("resume runner unavailable"); });
  renderWithProviders(createElement(RunsPage, {
    runs: [investigation, failed], tasks: [], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: apply,
    onResumeFromFailedRun: resume, onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs/investigation-run"] });

  fireEvent.click(screen.getByRole("button", { name: "Apply from detail" }));
  fireEvent.click(screen.getByRole("button", { name: "Apply Investigation" }));
  await vi.waitFor(() => assert.deepEqual(apply.mock.calls, [[investigation.id, [], undefined, undefined]]));
  await vi.waitFor(() => assert.ok(screen.getByText("apply runner unavailable")));

  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  fireEvent.click(screen.getByRole("button", { name: "Resume run Recoverable task from failure" }));
  fireEvent.click(screen.getByRole("button", { name: "Resume Run" }));
  await vi.waitFor(() => assert.deepEqual(resume.mock.calls, [[failed.id, undefined, undefined]]));
  await vi.waitFor(() => assert.ok(screen.getByText("resume runner unavailable")));
});

test("RunsPage forwards detail review, retry, and transcript actions into the durable run store", async () => {
  const run = makeRun({ id: "reviewable-run", taskId: "task-1", status: RunStatus.NEEDS_REVIEW });
  const retried = makeRun({ id: "retried-run", taskId: "task-1" });
  const continued = makeRun({ ...run, status: RunStatus.RUNNING });
  const setSelectedRun = vi.fn();
  const loadRunDetails = vi.fn();
  const snapshot = vi.fn();
  const gapFilled = vi.fn();
  const refresh = vi.fn();
  const approve = vi.fn().mockResolvedValue({});
  const reject = vi.fn().mockResolvedValue(undefined);
  const partial = vi.fn().mockResolvedValue({ remaining: 0 });
  const retry = vi.fn().mockResolvedValue(retried);
  const continueRun = vi.fn().mockResolvedValue(continued);
  const deleteMessage = vi.fn().mockResolvedValue(undefined);
  const events = [{ id: "fresh-event" }];

  setupControllerState();
  state.useSelectedRunController.mockReturnValue({
    selectedRun: run, setSelectedRun, selectedRunId: run.id, diff: null, events: [], eventsLoading: false, diffLoading: false,
    resolvedRuns: [run], getTaskById: () => undefined, getTaskTitle: () => "Reviewable task", loadRunDetails,
  });
  renderWithProviders(createElement(RunsPage, {
    runs: [run], tasks: [], profiles: [], loading: false, error: null, onRefresh: refresh,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: retry, onGetRun: vi.fn(), onGetEvents: vi.fn().mockResolvedValue(events), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: approve, onRejectRun: reject, onPartialApproveRun: partial, onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: continueRun, onDeleteRunMessage: deleteMessage,
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: { [run.id]: 4n } }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: snapshot, eventsGapFilled: gapFilled } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs/reviewable-run"] });

  fireEvent.click(screen.getByRole("button", { name: "Retry from detail" }));
  await vi.waitFor(() => assert.deepEqual(retry.mock.calls, [[run]]));
  assert.deepEqual(snapshot.mock.calls.at(-1), [retried]);
  assert.deepEqual(loadRunDetails.mock.calls.at(-1), [retried]);

  fireEvent.click(screen.getByRole("button", { name: "Approve from detail" }));
  fireEvent.click(screen.getByRole("button", { name: "Reject from detail" }));
  fireEvent.click(screen.getByRole("button", { name: "Partial approve from detail" }));
  await vi.waitFor(() => assert.equal(refresh.mock.calls.length, 3));
  assert.deepEqual(approve.mock.calls, [[run.id, { actor: "reviewer" }]]);
  assert.deepEqual(reject.mock.calls, [[run.id, { reason: "needs work" }]]);
  assert.deepEqual(partial.mock.calls, [[run.id, ["file-1"], "reviewer", "keep this"]]);
  assert.equal(setSelectedRun.mock.calls.filter(([value]) => value === null).length, 3);

  fireEvent.click(screen.getByRole("button", { name: "Continue from detail" }));
  fireEvent.click(screen.getByRole("button", { name: "Delete message from detail" }));
  await vi.waitFor(() => assert.deepEqual(continueRun.mock.calls, [[run.id, "continue carefully", ["attachment-1"]]]));
  await vi.waitFor(() => assert.deepEqual(deleteMessage.mock.calls, [[run.id, "message-1"]]));
  assert.deepEqual(gapFilled.mock.calls, [[run.id, events], [run.id, events]]);
});

test("RunsPage makes stop cancellation and stop failures non-destructive", async () => {
  setupControllerState();
  const run = makeRun({ id: "stoppable-run", taskId: "task-1", actions: { ...makeRun().actions, canStop: true } });
  state.useSelectedRunController.mockReturnValue({
    selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false,
    resolvedRuns: [run], getTaskById: () => undefined, getTaskTitle: () => "Stoppable task", loadRunDetails: noop,
  });
  const stop = vi.fn(async () => { throw new Error("runner no longer accepts stop requests"); });
  const refresh = vi.fn();
  const consoleError = vi.spyOn(console, "error").mockImplementation(noop);
  vi.stubGlobal("confirm", vi.fn(() => false));
  renderWithProviders(createElement(RunsPage, {
    runs: [run], tasks: [], profiles: [], loading: false, error: null, onRefresh: refresh,
    onStopRun: stop, onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });

  fireEvent.click(screen.getByRole("button", { name: "Stop run Stoppable task" }));
  assert.equal(stop.mock.calls.length, 0);
  vi.stubGlobal("confirm", vi.fn(() => true));
  fireEvent.click(screen.getByRole("button", { name: "Stop run Stoppable task" }));
  await vi.waitFor(() => assert.deepEqual(stop.mock.calls, [[run.id]]));
  assert.equal(refresh.mock.calls.length, 0);
  assert.match(consoleError.mock.calls[0]?.[0] ?? "", /Failed to stop run/);
  consoleError.mockRestore();
});

test("RunsPage supplies investigation defaults and clears transient modal state on close", async () => {
  const source = makeRun({ id: "source-defaults", taskId: "task-defaults" });
  const setInvestigateModalOpen = vi.fn();
  const setInvestigateError = vi.fn();
  state.useRunsPageState.mockReturnValue({
    searchQuery: "", setSearchQuery: noop, statusFilter: "all", setStatusFilter: noop, sortBy: "newest", setSortBy: noop,
    selectionMode: true, setSelectionMode: noop, selectedRunIds: new Set([source.id]), setSelectedRunIds: noop,
    investigateModalOpen: true, setInvestigateModalOpen, investigateLoading: false, setInvestigateLoading: noop,
    investigateError: "previous run failed", setInvestigateError, toggleSelectionMode: noop, clearSelection: noop, handleRunCheckboxChange: noop,
  });
  state.useSelectedRunController.mockReturnValue({
    selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false,
    resolvedRuns: [source], getTaskById: () => undefined, getTaskTitle: () => "Source defaults", loadRunDetails: noop,
  });
  renderWithProviders(createElement(RunsPage, {
    runs: [source], tasks: [{ id: source.taskId, projectRoot: "/workspace/agent-manager", scopePath: "api:ui:docs" }] as never[], profiles: [], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });

  assert.ok(screen.getByText("previous run failed"));
  assert.equal((screen.getByLabelText("Project Root") as HTMLInputElement).value, "/workspace/agent-manager");
  fireEvent.click(screen.getByRole("button", { name: "Close" }));
  await vi.waitFor(() => assert.deepEqual(setInvestigateModalOpen.mock.calls.at(-1), [false]));
  assert.deepEqual(setInvestigateError.mock.calls.at(-1), [null]);
});

test("RunsPage communicates pending, starting, and cancelled statuses plus profile fallbacks", () => {
  setupControllerState();
  const runs = [
    makeRun({ id: "pending-status", status: RunStatus.PENDING, agentProfileId: "known", executionMode: ExecutionMode.INTERACTIVE }),
    makeRun({ id: "starting-status", status: RunStatus.STARTING, agentProfileId: "missing" }),
    makeRun({ id: "cancelled-status", status: RunStatus.CANCELLED }),
  ];
  state.useSelectedRunController.mockReturnValue({
    selectedRun: null, setSelectedRun: noop, selectedRunId: null, diff: null, events: [], eventsLoading: false, diffLoading: false,
    resolvedRuns: runs, getTaskById: () => undefined, getTaskTitle: (id: string) => id, loadRunDetails: noop,
  });
  renderWithProviders(createElement(RunsPage, {
    runs, tasks: [], profiles: [{ id: "known", name: "Reliable profile" }] as never[], loading: false, error: null, onRefresh: noop,
    onStopRun: vi.fn(), onDeleteRun: vi.fn(), onRetryRun: vi.fn(), onGetRun: vi.fn(), onGetEvents: vi.fn(), onGetDiff: vi.fn(), onGetTask: vi.fn(),
    onApproveRun: vi.fn(), onRejectRun: vi.fn(), onPartialApproveRun: vi.fn(), onInvestigateRuns: vi.fn(), onApplyInvestigation: vi.fn(),
    onResumeFromFailedRun: vi.fn(), onContinueRun: vi.fn(), onDeleteRunMessage: vi.fn(),
    runEventStore: { state: { runsById: {}, lastSequenceByRunId: {} }, actions: { subscribeRun: noop, unsubscribeRun: noop, runSnapshotLoaded: noop, eventsGapFilled: noop } } as never,
    wsSubscribe: noop, wsUnsubscribe: noop,
  }), { initialEntries: ["/runs"] });

  for (const title of ["Pending", "Starting", "Cancelled"]) assert.ok(screen.getByTitle(title));
  assert.ok(screen.getByTestId("interactive-badge"));
  assert.ok(screen.getByText(/Reliable profile/));
  assert.ok(screen.getAllByText(/Unknown Profile/).length >= 1);
});

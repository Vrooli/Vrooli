import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { WorkflowExecutionStatus } from "@vrooli/proto-types/agent-manager/v1/domain/workflow_pb";
import { WorkflowsPage } from "../../src/pages/WorkflowsPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const state = vi.hoisted(() => ({ useWorkflowExecutions: vi.fn() }));
vi.mock("../../src/hooks/useApi.js", () => ({ useWorkflowExecutions: state.useWorkflowExecutions }));

const execution = {
  id: "workflow-12345678",
  workflowKey: "agent-manager/investigate",
  status: WorkflowExecutionStatus.WAITING,
  currentNodeId: "investigate",
  version: 4n,
  depth: 1,
  definitionDigest: "sha256:abc",
  parentExecutionId: "parent-12345678",
  parentAttemptId: "attempt-12345678",
  budgetUsage: { turns: 2, tokens: 100, costUsd: 1.25, nodeAttempts: 1, children: 0, retries: 0 },
};

function hook(overrides: Record<string, unknown> = {}) {
  return {
    data: [execution], loading: false, error: null, refetch: vi.fn(),
    getTrace: vi.fn(async () => ({ attempts: [{ id: "attempt-1", nodeId: "investigate", ordinal: 1, profileIdentity: "investigator", strategy: "run", status: "waiting", runId: "run-12345678", conversationId: "conversation-12345678", sourceAttemptId: "", childExecutionId: "", inputSnapshotSizeBytes: 32n, inputSnapshotDigest: "sha256:abcdef" }], journal: [{ id: "journal-1", sequence: 1n, kind: "wait", nodeId: "investigate", attemptId: "attempt-1", payloadSizeBytes: 24n }] })),
    control: vi.fn(async () => undefined), signal: vi.fn(async () => undefined), ...overrides,
  };
}

afterEach(() => vi.resetAllMocks());

test("WorkflowsPage presents metadata-only trace evidence and controls the selected execution", async () => {
  const user = userEvent.setup();
  const workflows = hook();
  state.useWorkflowExecutions.mockReturnValue(workflows);
  renderWithProviders(createElement(WorkflowsPage));

  await waitFor(() => assert.ok(screen.getByText("Attempts and identity")));
  assert.ok(screen.getByText(/Routine inspection is metadata-only/));
  assert.ok(screen.getByText("investigator"));
  assert.ok(screen.getByText(/wait · node investigate/));
  await user.click(screen.getByRole("button", { name: "Retry" }));
  await waitFor(() => assert.equal(workflows.control.mock.calls[0]?.[1], "retry"));
  await user.type(screen.getByLabelText("Signal name"), "continue");
  fireEvent.change(screen.getByLabelText("JSON payload"), { target: { value: '{"approved":true}' } });
  await user.click(screen.getByRole("button", { name: "Signal" }));
  await waitFor(() => assert.deepEqual(workflows.signal.mock.calls[0]?.slice(1), ["continue", { approved: true }]));
});

test("WorkflowsPage keeps empty, loading, and trace failures legible", async () => {
  state.useWorkflowExecutions.mockReturnValue({ ...hook(), data: [], loading: false });
  const empty = renderWithProviders(createElement(WorkflowsPage));
  assert.ok(screen.getByText(/No workflow executions yet/));
  empty.unmount();
  state.useWorkflowExecutions.mockReturnValue({ ...hook(), data: [], loading: true, error: "workflow API unavailable" });
  renderWithProviders(createElement(WorkflowsPage));
  assert.ok(screen.getByText("Loading workflow history…"));
  assert.ok(screen.getByRole("alert").textContent?.includes("workflow API unavailable"));
});

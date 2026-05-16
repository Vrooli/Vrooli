import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import {
  agentSessionStoreInitialState,
  backlogStoreInitialState,
  captureStoreInitialState,
  executionStoreInitialState,
  useAgentSessionStore,
  useBacklogStore,
  useCaptureStore,
  useExecutionStore,
} from "../../../../stores";
import { useSnoozeStore } from "../../../../stores/snooze-store";
import type { AgentSession, BacklogItem, Capture, ExecutionRecord } from "../../../../types";
import { SidebarTabs } from "./SidebarTabs";

const noop = () => {};

function makeBacklog(overrides: Partial<BacklogItem> = {}): BacklogItem {
  return {
    name: "plan-ready",
    title: "Plan ready",
    description: "",
    kind: "idea",
    status: "ready",
    priority: 2,
    tags: [],
    suggestedSkills: [],
    created: "2026-05-01T12:00:00Z",
    updated: "2026-05-01T12:00:00Z",
    ...overrides,
  };
}

function makeCapture(overrides: Partial<Capture> = {}): Capture {
  return {
    id: "cap-1",
    text: "Capture this",
    attachments: [],
    created: "2026-05-01T12:00:00Z",
    status: "classifying",
    classification: null,
    ...overrides,
  };
}

function makeExecution(overrides: Partial<ExecutionRecord> = {}): ExecutionRecord {
  return {
    executionId: "exec-1",
    backlogKind: "idea",
    backlogName: "run-review",
    status: "needs_review",
    mode: "manual",
    startedAt: "2026-05-01T12:00:00Z",
    createdAt: "2026-05-01T12:00:00Z",
    ...overrides,
  } as ExecutionRecord;
}

function makeSession(overrides: Partial<AgentSession> = {}): AgentSession {
  return {
    id: "sess-1",
    title: "Operator decision",
    kind: "swarm_operations",
    status: "waiting_for_user",
    skillId: "swarm-manager-operations-session",
    taskId: "task-1",
    runId: "run-1",
    profileKey: "swarm-manager/default",
    createdAt: "2026-05-01T12:00:00Z",
    updatedAt: "2026-05-01T12:00:00Z",
    messages: [],
    proposals: [],
    artifacts: [],
    ...overrides,
  };
}

function renderTabs() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  queryClient.setQueryData(["backlog-summary"], {
    feedback: {
      items: [
        { kind: "idea", name: "needs-answer", pending_decisions: 2 },
      ],
    },
    maturity: {
      items: [
        { kind: "idea", name: "plan-ready", ready: true, pending_items: 0 },
      ],
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <SidebarTabs activeTab="activity" onTabChange={noop} />
    </QueryClientProvider>,
  );
}

describe("SidebarTabs", () => {
  beforeEach(() => {
    useBacklogStore.setState({
      ...backlogStoreInitialState,
      items: [],
      blockingMap: {},
      status: "success",
    });
    useCaptureStore.setState({
      ...captureStoreInitialState,
      captures: [],
      status: "success",
    });
    useExecutionStore.setState({
      ...executionStoreInitialState,
      items: [],
      status: "success",
    });
    useAgentSessionStore.setState({
      ...agentSessionStoreInitialState,
      sessions: [],
      status: "success",
    });
    useSnoozeStore.setState({ entries: new Map() });
  });

  it("shows actionable entity badges for backlog, captures, executions, and sessions", () => {
    useBacklogStore.setState({
      items: [
        makeBacklog(),
        makeBacklog({ name: "needs-answer", title: "Needs answer", status: "ready" }),
        makeBacklog({ name: "archived-done", title: "Archived done", status: "completed", archivedAt: "2026-05-01T13:00:00Z" }),
      ],
    });
    useCaptureStore.setState({
      captures: [
        makeCapture(),
        makeCapture({
          id: "cap-2",
          status: "classified",
          classification: {
            classifiedAt: "2026-05-01T12:05:00Z",
            items: [{ kind: "fix", title: "Fix it", description: "", priority: 3, tags: [], confidence: 0.9 }],
          },
        }),
        makeCapture({
          id: "cap-3",
          status: "classified",
          classification: { classifiedAt: "2026-05-01T12:06:00Z", items: [] },
        }),
      ],
    });
    useExecutionStore.setState({
      items: [
        makeExecution(),
        makeExecution({ executionId: "exec-2", status: "running" }),
      ],
    });
    useAgentSessionStore.setState({
      sessions: [
        makeSession(),
        makeSession({ id: "sess-2", status: "proposal_ready" }),
        makeSession({ id: "sess-3", status: "running" }),
      ],
    });

    renderTabs();

    expect(within(screen.getByTestId("sidebar-tab-backlog")).getByText("3")).toBeInTheDocument();
    expect(within(screen.getByTestId("sidebar-tab-captures")).getByText("2")).toBeInTheDocument();
    expect(within(screen.getByTestId("sidebar-tab-executions")).getByText("1")).toBeInTheDocument();
    expect(within(screen.getByTestId("sidebar-tab-sessions")).getByText("2")).toBeInTheDocument();
  });

  it("does not badge non-actionable tabs or raw active sessions", () => {
    useAgentSessionStore.setState({
      sessions: [makeSession({ status: "running" })],
    });

    renderTabs();

    expect(within(screen.getByTestId("sidebar-tab-activity")).queryByText(/\d+/)).toBeNull();
    expect(within(screen.getByTestId("sidebar-tab-operatingModes")).queryByText(/\d+/)).toBeNull();
    expect(within(screen.getByTestId("sidebar-tab-sessions")).queryByText(/\d+/)).toBeNull();
  });
});

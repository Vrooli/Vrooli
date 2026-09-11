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
import { SIDEBAR_TABS } from "./types";

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
  queryClient.setQueryData(["next-actions-feed"], {
    entries: [{ entity_kind: "backlog_item" }, { entity_kind: "backlog_item" }],
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <SidebarTabs activeTab="goals" onTabChange={noop} />
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

    expect(within(screen.getByTestId("sidebar-tab-backlog")).getByText("2")).toBeInTheDocument();
    // The classifying capture is machine work; only the one with proposals waits on the operator.
    expect(within(screen.getByTestId("sidebar-tab-captures")).getByText("1")).toBeInTheDocument();
    expect(within(screen.getByTestId("sidebar-tab-executions")).getByText("1")).toBeInTheDocument();
    expect(within(screen.getByTestId("sidebar-tab-sessions")).getByText("2")).toBeInTheDocument();
  });

  it("explains what each badge counts", () => {
    useCaptureStore.setState({
      captures: [
        makeCapture({
          status: "classified",
          classification: {
            classifiedAt: "2026-05-01T12:05:00Z",
            items: [{ kind: "fix", title: "Fix it", description: "", priority: 3, tags: [], confidence: 0.9 }],
          },
        }),
      ],
    });
    useAgentSessionStore.setState({ sessions: [makeSession(), makeSession({ id: "sess-2" })] });

    renderTabs();

    expect(within(screen.getByTestId("sidebar-tab-backlog")).getByText("2 backlog items have a next step for you")).toBeInTheDocument();
    expect(within(screen.getByTestId("sidebar-tab-captures")).getByText("1 capture has proposals to review")).toBeInTheDocument();
    expect(within(screen.getByTestId("sidebar-tab-sessions")).getByText("2 sessions are waiting on you")).toBeInTheDocument();
  });

  it("does not badge captures that are still being classified", () => {
    useCaptureStore.setState({ captures: [makeCapture({ status: "classifying" })] });

    renderTabs();

    expect(screen.queryByTestId("sidebar-tab-captures-badge")).toBeNull();
  });

  it("does not badge non-actionable tabs or raw active sessions", () => {
    useAgentSessionStore.setState({
      sessions: [makeSession({ status: "running" })],
    });

    renderTabs();

    expect(within(screen.getByTestId("sidebar-tab-goals")).queryByText(/\d+/)).toBeNull();
    expect(within(screen.getByTestId("sidebar-tab-sessions")).queryByText(/\d+/)).toBeNull();
  });

  it("renders an entity icon for every tab", () => {
    renderTabs();

    for (const tab of SIDEBAR_TABS) {
      expect(screen.getByTestId(`sidebar-tab-${tab}`).querySelector("svg")).toBeInTheDocument();
    }
  });
});

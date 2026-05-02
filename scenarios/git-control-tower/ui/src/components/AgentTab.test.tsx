import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AgentTab } from "./AgentTab";
import { renderWithQueryClient } from "../test-utils";
import type {
  AgentRun,
  AgentRunDiffResponse,
  AgentRunEventsResponse,
  AgentRunListResponse,
  AgentProfileListResponse,
  ScenarioEnvelopeData,
} from "../lib/api";

const mocks = vi.hoisted(() => ({
  fetchExternalUrl: vi.fn(),
  createRunMutate: vi.fn(),
  continueRunMutate: vi.fn(),
  approveMutate: vi.fn(),
  rejectMutate: vi.fn(),
  stopMutate: vi.fn(),
  onActiveRunIdChange: vi.fn(),
  onAddContext: vi.fn(),
  onRemoveContext: vi.fn(),
  onClearContext: vi.fn(),
  profiles: undefined as AgentProfileListResponse | undefined,
  envelope: undefined as ScenarioEnvelopeData | undefined,
  runs: undefined as AgentRunListResponse | undefined,
  activeRun: null as AgentRun | null,
  runEvents: undefined as AgentRunEventsResponse | undefined,
  runEventsError: null as Error | null,
  runDiff: undefined as AgentRunDiffResponse | undefined,
}));

vi.mock("../lib/api-internals", () => ({
  fetchExternalUrl: (...args: unknown[]) => mocks.fetchExternalUrl(...args),
}));

vi.mock("./ContextPickerPopover", () => ({
  ContextPickerPopover: ({ onAddContext }: { onAddContext: (item: unknown) => void }) => (
    <button
      type="button"
      onClick={() => onAddContext({ id: "ctx-added", kind: "change-summary", label: "Added context", markdown: "details" })}
    >
      Add context
    </button>
  ),
}));

vi.mock("./ContextPreviewPopover", () => ({
  ContextPreviewPopover: ({ item, onRemove }: { item: { id: string; label: string }; onRemove: (id: string) => void }) => (
    <button type="button" onClick={() => onRemove(item.id)}>
      {item.label}
    </button>
  ),
}));

vi.mock("../lib/hooks", () => ({
  useAgentProfiles: () => query(mocks.profiles),
  useScenarioEnvelope: () => query(mocks.envelope),
  useAgentRuns: () => query(mocks.runs),
  useAgentRun: () => query(mocks.activeRun),
  useAgentRunEvents: () => ({
    ...query(mocks.runEvents),
    isLoading: false,
    error: mocks.runEventsError,
  }),
  useAgentRunDiff: () => query(mocks.runDiff),
  useCreateAgentRun: () => mutation(mocks.createRunMutate),
  useContinueAgentRun: () => mutation(mocks.continueRunMutate),
  useApproveAgentRun: () => mutation(mocks.approveMutate),
  useRejectAgentRun: () => mutation(mocks.rejectMutate),
  useStopAgentRun: () => mutation(mocks.stopMutate),
}));

function query<T>(data: T) {
  return {
    data,
    isLoading: false,
    isPending: false,
    isError: false,
    error: null,
  };
}

function mutation(mutate: ReturnType<typeof vi.fn>) {
  return {
    mutate,
    isPending: false,
  };
}

function run(overrides: Partial<AgentRun> = {}): AgentRun {
  return {
    id: "run-1",
    status: "complete",
    createdAt: "2026-05-01T12:00:00Z",
    actions: {
      canStop: false,
      canRetry: false,
      canContinue: false,
      canApprove: false,
      canReject: false,
    },
    ...overrides,
  };
}

function renderAgentTab(overrides: Partial<Parameters<typeof AgentTab>[0]> = {}) {
  return renderWithQueryClient(
    <AgentTab
      scenarioSlug="git-control-tower"
      repoId="repo-1"
      agentManagerAvailable
      workspaceSandboxAvailable
      contextItems={[]}
      onAddContext={mocks.onAddContext}
      onRemoveContext={mocks.onRemoveContext}
      onClearContext={mocks.onClearContext}
      testGenieAvailable
      tidinessAvailable
      auditorAvailable
      visualCaptureAvailable
      activeRunId={null}
      onActiveRunIdChange={mocks.onActiveRunIdChange}
      {...overrides}
    />,
  );
}

describe("AgentTab", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchExternalUrl.mockResolvedValue("");
    mocks.profiles = { profiles: [], total: 0 };
    mocks.envelope = {
      name: "git-control-tower",
      displayName: "Git Control Tower",
      description: "Agent-friendly git repository control plane.",
      path: "/scenarios/git-control-tower",
      tags: ["git", "review"],
      dependencies: {
        scenarios: {},
        resources: {},
      },
      lifecycle: {
        testCommand: "make test",
      },
    };
    mocks.runs = { runs: [], total: 0 };
    mocks.activeRun = null;
    mocks.runEvents = { events: [] };
    mocks.runEventsError = null;
    mocks.runDiff = { runId: "run-1", files: [] };
  });

  it("shows the unavailable state without hiding the required setup action", () => {
    renderAgentTab({ agentManagerAvailable: false });

    expect(screen.getByText("Agent Manager is not available")).toBeInTheDocument();
    expect(screen.getByText(/start the agent-manager scenario/i)).toBeInTheDocument();
  });

  it("auto-selects an active run from history when no run is selected", async () => {
    mocks.runs = {
      total: 2,
      runs: [
        run({ id: "run-active", status: "running", promptPreview: "Fix failing tests" }),
        run({ id: "run-done", status: "complete", promptPreview: "Previous task" }),
      ],
    };

    renderAgentTab();

    await waitFor(() => {
      expect(mocks.onActiveRunIdChange).toHaveBeenCalledWith("run-active");
    });
  });

  it("renders review messages, diff, sandbox link, and approval actions", async () => {
    mocks.runs = {
      total: 1,
      runs: [run({ id: "run-1", status: "needs_review", promptPreview: "Review app shell" })],
    };
    mocks.activeRun = run({
      id: "run-1",
      status: "needs_review",
      approvalState: "pending",
      sandboxId: "sandbox-1",
      promptPreview: "Review app shell",
      startedAt: "2026-05-01T12:00:00Z",
      endedAt: "2026-05-01T12:01:30Z",
      summary: {
        tokensUsed: 12500,
        turnsUsed: 4,
        costEstimate: 0.42,
        filesModified: ["src/App.tsx"],
      },
      actions: {
        canStop: false,
        canRetry: false,
        canContinue: false,
        canApprove: true,
        canReject: true,
      },
    });
    mocks.runEvents = {
      events: [
        {
          id: "evt-1",
          runId: "run-1",
          sequence: 1,
          eventType: "message",
          timestamp: "2026-05-01T12:00:01Z",
          data: { role: "user", content: "Please review the panel routing" },
        },
        {
          id: "evt-2",
          runId: "run-1",
          sequence: 2,
          eventType: "message",
          timestamp: "2026-05-01T12:00:02Z",
          data: { role: "assistant", content: "I found one issue and patched it." },
        },
        {
          id: "evt-3",
          runId: "run-1",
          sequence: 3,
          eventType: "tool_call",
          timestamp: "2026-05-01T12:00:03Z",
          data: { name: "pnpm test" },
        },
        {
          id: "evt-4",
          runId: "run-1",
          sequence: 4,
          eventType: "tool_result",
          timestamp: "2026-05-01T12:00:04Z",
          data: { result: "268 tests passed" },
        },
      ],
    };
    mocks.runDiff = {
      runId: "run-1",
      files: [
        {
          path: "src/App.tsx",
          changeType: "modified",
          additions: 12,
          deletions: 3,
          patch: "@@ patched",
        },
      ],
    };

    renderAgentTab({ activeRunId: "run-1" });

    expect(await screen.findByText("Please review the panel routing")).toBeInTheDocument();
    expect(screen.getByText("I found one issue and patched it.")).toBeInTheDocument();
    expect(screen.getByText("Summary")).toBeInTheDocument();
    expect(screen.getByText("12.5K tokens")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /review in sandbox/i })).toHaveAttribute(
      "href",
      expect.stringContaining("sandbox=sandbox-1"),
    );
    expect(screen.getByText("src/App.tsx")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /used 1 tool/i }));
    expect(screen.getByText("pnpm test")).toBeInTheDocument();
    expect(screen.getByText("268 tests passed")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /approve/i }));
    expect(mocks.approveMutate).toHaveBeenCalledWith({
      runId: "run-1",
      request: {},
    });
  });

  it("creates a first run with scenario envelope and selected context", async () => {
    const contextItem = {
      id: "test-failure-1",
      kind: "test-failure" as const,
      label: "Smoke failure",
      markdown: "Iframe bridge did not signal ready.",
    };
    mocks.createRunMutate.mockImplementation((_request, options) => {
      options.onSuccess({ runId: "run-created", taskId: "task-1" });
    });

    renderAgentTab({
      contextItems: [contextItem],
    });

    fireEvent.change(screen.getByPlaceholderText("Ask the agent to fix something..."), {
      target: { value: "Fix the smoke failure" },
    });
    fireEvent.click(screen.getByRole("button", { name: /smoke failure/i }));
    expect(mocks.onRemoveContext).toHaveBeenCalledWith("test-failure-1");

    fireEvent.click(screen.getByRole("button", { name: /add context/i }));
    expect(mocks.onAddContext).toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /send message/i }));

    await waitFor(() => {
      expect(mocks.createRunMutate).toHaveBeenCalled();
    });
    const firstCreateCall = mocks.createRunMutate.mock.calls[0];
    if (!firstCreateCall) {
      throw new Error("expected create-run mutation to be called");
    }
    const [request] = firstCreateCall;
    expect(request.scenarioSlug).toBe("git-control-tower");
    expect(request.profileKey).toBe("git-control-tower-reviewer");
    expect(request.prompt).toContain("Fix the smoke failure");
    expect(request.prompt).toContain("Git Control Tower");
    expect(request.prompt).toContain("Iframe bridge did not signal ready.");
    expect(mocks.onActiveRunIdChange).toHaveBeenCalledWith("run-created");
    expect(mocks.onClearContext).toHaveBeenCalled();
  });
});

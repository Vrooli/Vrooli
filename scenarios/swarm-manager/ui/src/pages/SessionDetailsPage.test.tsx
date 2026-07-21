import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";
import type { AgentSession } from "../types";
import { SessionDetailsPage } from "./SessionDetailsPage";
import { createTestQueryClient, installMatchMediaMock, renderWithProviders } from "../test-utils";
import { stageContextForSession } from "../components/session/context/pending-session-context";

const storeMock = vi.hoisted(() => {
  const initialState = {
    sessions: [],
    status: "idle",
    error: null,
    isMutating: false,
    isRefreshing: false,
    loadSession: vi.fn(),
    startSession: vi.fn(),
    continueSession: vi.fn(),
    uploadSessionAttachments: vi.fn().mockResolvedValue([]),
    refreshSession: vi.fn(),
    cancelSession: vi.fn(),
    deleteSession: vi.fn(),
    applyProposal: vi.fn(),
    fetchSessions: vi.fn().mockResolvedValue(undefined),
  };
  let state: Record<string, unknown> = { ...initialState };
  const makeUseStore = (storeState: Record<string, unknown>) =>
    (selector: (value: Record<string, unknown>) => unknown) => selector(storeState);
  const useAgentSessionStore = Object.assign(
    (selector: (value: Record<string, unknown>) => unknown) => selector(state),
    {
      setState: (partial: Record<string, unknown>) => {
        state = { ...state, ...partial };
      },
      reset: () => {
        state = { ...initialState };
      },
    },
  );
  return {
    initialState,
    useAgentSessionStore,
    useBacklogStore: makeUseStore({
      fetchBacklog: vi.fn().mockResolvedValue(undefined),
      items: [{ kind: "execute", name: "starter-item", title: "Starter item", status: "new" }],
    }),
    useInitiativeStore: makeUseStore({ fetchInitiatives: vi.fn().mockResolvedValue(undefined), items: [] }),
    useCaptureStore: makeUseStore({ fetchCaptures: vi.fn().mockResolvedValue(undefined), captures: [] }),
    useExecutionStore: makeUseStore({ fetchExecutions: vi.fn().mockResolvedValue(undefined), items: [] }),
    useAgentActivitiesStore: makeUseStore({ refreshActivities: vi.fn().mockResolvedValue(undefined), activities: [] }),
    useScenariosStore: makeUseStore({ fetchScenarios: vi.fn().mockResolvedValue(undefined), scenarios: [] }),
  };
});

vi.mock("../stores", () => ({
  agentSessionStoreInitialState: storeMock.initialState,
  useAgentSessionStore: storeMock.useAgentSessionStore,
  useBacklogStore: storeMock.useBacklogStore,
  useInitiativeStore: storeMock.useInitiativeStore,
  useCaptureStore: storeMock.useCaptureStore,
  useExecutionStore: storeMock.useExecutionStore,
  useAgentActivitiesStore: storeMock.useAgentActivitiesStore,
  useScenariosStore: storeMock.useScenariosStore,
}));

vi.mock("@vrooli/api-base", () => ({
  buildApiUrl: (path: string) => path,
  buildWsUrl: (path: string) => path,
  resolveApiBase: () => "http://localhost",
}));

beforeAll(() => {
  installMatchMediaMock();
});

vi.mock("../hooks/useStorePolling", () => ({
  useStorePolling: vi.fn(),
}));

vi.mock("../hooks/useAgentSessionPolling", () => ({
  useAgentSessionPolling: vi.fn(),
}));

const navigateMock = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => navigateMock };
});

const SESSION: AgentSession = {
  id: "sess_meta",
  title: "Plan quality work",
  kind: "meta_orchestration",
  status: "proposal_ready",
  skillId: "swarm-manager-meta-orchestrator",
  taskId: "task-meta",
  runId: "run-meta",
  profileKey: "swarm-manager/default",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:10:00Z",
  messages: [
    { id: "msg-1", role: "user", content: "Plan it.", createdAt: "2026-05-01T12:00:00Z", attachmentIds: [] },
    { id: "msg-2", role: "assistant", content: "**On it.**", createdAt: "2026-05-01T12:01:00Z", attachmentIds: [] },
  ],
  proposals: [
    {
      id: "prop-1",
      kind: "backlog_batch_import",
      status: "ready",
      summary: "Apply this plan.",
      payloadJson: "{}",
      createdAt: "2026-05-01T12:05:00Z",
      updatedAt: "2026-05-01T12:09:00Z",
    },
  ],
  artifacts: [
    {
      id: "art-1",
      sessionId: "sess_meta",
      artifactType: "initiative",
      action: "created",
      entityRef: "quality-gates",
      title: "Quality gates",
      createdAt: "2026-05-01T12:11:00Z",
    },
  ],
  proposalTarget: { type: "initiative", ref: "quality-gates", name: "Quality gates" },
};

const DRAFT_SESSION: AgentSession = {
  ...SESSION,
  id: "sess_draft",
  title: "Draft planning session",
  status: "draft",
  taskId: undefined,
  runId: undefined,
  messages: [],
  proposals: [],
  artifacts: [],
};

function renderPage(initialPath = "/sessions/sess_meta") {
  const queryClient = createTestQueryClient();
  queryClient.setQueryData(["embedded-service-url", "agent-manager"], "https://agent.test");
  queryClient.setQueryData(["embedded-service-url", "prompt-manager"], "https://prompt.test");

  return renderWithProviders(
    <Routes>
      <Route path="/sessions/:sessionId" element={<SessionDetailsPage />} />
    </Routes>,
    { initialEntries: [initialPath], queryClient },
  );
}

async function activateTab(name: string | RegExp) {
  await userEvent.click(screen.getByRole("tab", { name }));
}

describe("SessionDetailsPage", () => {
  beforeEach(() => {
    installMatchMediaMock(false);
    window.localStorage.clear();
    navigateMock.mockReset();
    storeMock.useAgentSessionStore.reset();
    storeMock.useAgentSessionStore.setState({
      ...storeMock.initialState,
      sessions: [SESSION],
      status: "success",
    });
  });

  it("groups proposals with artifacts instead of rendering a separate proposal tab", async () => {
    renderPage();

    expect(screen.getByText("Plan quality work")).toBeInTheDocument();
    expect(screen.getByText("Plan it.")).toBeInTheDocument();
    expect(screen.getByText("On it.")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: /Proposals/ })).toBeNull();
    await activateTab(/Artifacts 2/);
    expect(screen.getByText("Apply this plan.")).toBeInTheDocument();
    expect(screen.getByText("Quality gates")).toBeInTheDocument();
  });

  it("deep-links session metadata to Agent Manager and Prompt Manager", async () => {
    renderPage();

    await activateTab("Details");

    expect(screen.getByTestId("agent-session-skill-link")).toHaveAttribute(
      "href",
      "https://prompt.test/skills/swarm-manager-meta-orchestrator",
    );
    expect(screen.getByTestId("agent-session-run-link")).toHaveAttribute(
      "href",
      "https://agent.test/runs/run-meta",
    );
    expect(screen.getByTestId("agent-session-task-link")).toHaveAttribute(
      "href",
      "https://agent.test/tasks?taskId=task-meta",
    );
    expect(screen.getByTestId("agent-session-profile-link")).toHaveAttribute(
      "href",
      "https://agent.test/profiles?profileKey=swarm-manager%2Fdefault",
    );
  });

  it("renders assistant markdown as HTML", () => {
    renderPage();

    expect(screen.getByText("On it.").tagName).toBe("STRONG");
  });

  it("collapses and expands the desktop inspector", () => {
    renderPage();

    expect(screen.getByTestId("session-inspector")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("session-inspector-collapse"));
    expect(screen.queryByTestId("session-inspector")).toBeNull();

    fireEvent.click(screen.getByTestId("session-inspector-expand"));
    expect(screen.getByTestId("session-inspector")).toBeInTheDocument();
  });

  it("uses edge-to-edge independently scrollable desktop panes", () => {
    renderPage();

    const layout = screen.getByTestId("detail-page-layout");
    const desktopLayout = screen.getByTestId("session-desktop-layout");
    const conversation = screen.getByTestId("agent-session-conversation");
    const messages = screen.getByTestId("agent-session-messages");
    const inspector = screen.getByTestId("session-inspector");

    expect(layout).toHaveClass("md:h-full", "md:min-h-0", "md:overflow-hidden");
    expect(desktopLayout).toHaveClass("flex-1", "min-h-0");
    expect(conversation).toHaveClass("h-full", "border-r");
    expect(conversation).not.toHaveClass("rounded-lg");
    expect(messages).toHaveClass("overflow-y-auto");
    expect(inspector).toHaveClass("h-full", "border-l");
    expect(inspector).not.toHaveClass("rounded-lg");
    expect(screen.getByTestId("session-inspector-resize-handle")).toHaveAttribute("role", "separator");
  });

  it("defaults the desktop inspector to artifacts when a proposal is ready", () => {
    renderPage();

    expect(screen.getByRole("tab", { name: /Artifacts 2/ })).toHaveAttribute("data-state", "active");
  });

  it("uses full-page mobile tabs instead of stacked secondary sections", async () => {
    installMatchMediaMock(true);
    renderPage();

    expect(screen.getByRole("tab", { name: "Conversation" })).toHaveAttribute("data-state", "active");
    expect(screen.queryByTestId("detail-mobile-actions-fab")).toBeNull();
    expect(screen.getByTestId("session-header-actions")).toBeInTheDocument();
    expect(screen.getByText("Plan it.")).toBeVisible();
    expect(screen.queryByText("Apply this plan.")).toBeNull();
    expect(screen.getByTestId("agent-session-composer").closest(".fixed")).toBeInTheDocument();

    await activateTab(/Artifacts 2/);

    expect(screen.getByText("Apply this plan.")).toBeVisible();
  });

  it("puts mobile session actions behind the header ellipsis", async () => {
    installMatchMediaMock(true);
    const refreshSession = vi.fn().mockResolvedValue(SESSION);
    storeMock.useAgentSessionStore.setState({ refreshSession });

    renderPage();

    await userEvent.click(screen.getByTestId("session-header-actions"));
    expect(screen.getByTestId("session-actions-menu")).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("session-refresh"));

    await waitFor(() => expect(refreshSession).toHaveBeenCalledWith("sess_meta"));
  });

  it("sends a continuation on Ctrl+Enter", async () => {
    const continueSession = vi.fn().mockResolvedValue(SESSION);
    storeMock.useAgentSessionStore.setState({ continueSession });

    renderPage();

    const composer = screen.getByTestId("agent-session-composer");
    fireEvent.change(composer, { target: { value: "Next step" } });
    fireEvent.keyDown(composer, { key: "Enter", ctrlKey: true });

    await waitFor(() => {
      expect(continueSession).toHaveBeenCalledWith({ sessionId: "sess_meta", message: "Next step" });
    });
  });

  it("applies staged context and sends it with the next continuation [REQ:REQ-P1-010-SESSION-CONTEXT]", async () => {
    const continueSession = vi.fn().mockResolvedValue(SESSION);
    storeMock.useAgentSessionStore.setState({ continueSession });
    stageContextForSession("sess_meta", {
      type: "initiative",
      ref: "quality-gates",
      title: "Quality gates",
    });

    renderPage();

    expect((await screen.findAllByText("Quality gates")).length).toBeGreaterThan(0);

    const composer = screen.getByTestId("agent-session-composer");
    fireEvent.change(composer, { target: { value: "Use this context" } });
    fireEvent.keyDown(composer, { key: "Enter", ctrlKey: true });

    await waitFor(() => {
      expect(continueSession).toHaveBeenCalledWith({
        sessionId: "sess_meta",
        message: "Use this context",
        contextRefs: [{ type: "initiative", ref: "quality-gates" }],
      });
    });
  });

  it("shows starter suggestions for draft sessions and starts with the selected ready prompt", async () => {
    const startSession = vi.fn().mockResolvedValue({ ...DRAFT_SESSION, status: "running", runId: "run-draft" });
    storeMock.useAgentSessionStore.setState({ sessions: [DRAFT_SESSION], startSession });

    renderPage("/sessions/sess_draft");

    expect(screen.getByTestId("agent-session-starter-suggestions")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /Turn this idea into initiatives and backlog items/i }));

    expect(screen.getByTestId("agent-session-composer")).toHaveValue("Turn this idea into initiatives and backlog items.");

    fireEvent.keyDown(screen.getByTestId("agent-session-composer"), { key: "Enter", ctrlKey: true });

    await waitFor(() => {
      expect(startSession).toHaveBeenCalledWith({
        sessionId: "sess_draft",
        message: "Turn this idea into initiatives and backlog items.",
        contextRefs: [{ type: "startup_brief", ref: "startup_brief/meta_orchestration" }],
        autoContextPolicy: "none",
      });
    });
  });

  it("opens the context picker to the required tab for guided starter suggestions", async () => {
    storeMock.useAgentSessionStore.setState({
      sessions: [{ ...DRAFT_SESSION, kind: "swarm_operations", title: "Operations draft" }],
    });

    renderPage("/sessions/sess_draft");

    await userEvent.click(screen.getByRole("button", { name: /Help me drain workshop decisions for a backlog item/i }));

    expect(screen.getByTestId("session-context-picker")).toBeInTheDocument();
    expect(screen.getByTestId("session-context-tab-backlog_item")).toHaveAttribute("data-state", "active");
  });

  it("restores the composer draft and shows an alert when send fails", async () => {
    const continueSession = vi.fn().mockRejectedValue(new Error("agent offline"));
    storeMock.useAgentSessionStore.setState({ continueSession });

    renderPage();

    const composer = screen.getByTestId("agent-session-composer");
    fireEvent.change(composer, { target: { value: "Retry this" } });
    fireEvent.keyDown(composer, { key: "Enter", ctrlKey: true });

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("agent offline"));
    expect(screen.getByTestId("agent-session-composer")).toHaveValue("Retry this");
  });

  it("invokes refresh (from the menu) and cancel (inline primary) actions", async () => {
    const refreshSession = vi.fn().mockResolvedValue(SESSION);
    const cancelSession = vi.fn().mockResolvedValue(SESSION);
    storeMock.useAgentSessionStore.setState({ refreshSession, cancelSession });

    renderPage();

    fireEvent.click(screen.getByTestId("session-cancel"));
    await userEvent.click(screen.getByTestId("session-header-actions"));
    await userEvent.click(screen.getByTestId("session-refresh"));

    await waitFor(() => expect(refreshSession).toHaveBeenCalledWith("sess_meta"));
    await waitFor(() => expect(cancelSession).toHaveBeenCalledWith("sess_meta"));
  });

  it("keeps only Cancel inline; delete and refresh live in the header ellipsis menu", async () => {
    renderPage();

    expect(screen.getByTestId("session-cancel")).toBeInTheDocument();
    expect(screen.queryByTestId("session-refresh")).toBeNull();
    expect(screen.queryByTestId("session-delete-action")).toBeNull();

    await userEvent.click(screen.getByTestId("session-header-actions"));

    expect(screen.getByTestId("session-actions-menu")).toBeInTheDocument();
    expect(screen.getByTestId("session-refresh")).toBeInTheDocument();
    expect(screen.getByTestId("session-delete-action")).toHaveTextContent("Delete session");
  });

  it("confirms before deleting and navigates away on success", async () => {
    // Sessions default to the `simple` confirmation level (the over-confirm fix),
    // so the dialog has no type-the-name gate; confirm is enabled immediately.
    const deleteSession = vi.fn().mockResolvedValue(undefined);
    storeMock.useAgentSessionStore.setState({ deleteSession });
    renderPage();

    await userEvent.click(screen.getByTestId("session-header-actions"));
    await userEvent.click(screen.getByTestId("session-delete-action"));

    expect(screen.getByTestId("session-delete-dialog")).toBeInTheDocument();
    expect(screen.getByText(/Created backlog items, initiatives, captures, files, and agent activity records stay/)).toBeInTheDocument();
    expect(screen.getByTestId("session-delete-confirm")).toBeEnabled();

    await userEvent.click(screen.getByTestId("session-delete-confirm"));

    await waitFor(() => expect(deleteSession).toHaveBeenCalledWith("sess_meta"));
    expect(navigateMock).toHaveBeenCalled();
  });

  it("keeps the page open and shows an alert when delete fails", async () => {
    const deleteSession = vi.fn().mockRejectedValue(new Error("delete failed"));
    storeMock.useAgentSessionStore.setState({ deleteSession });
    renderPage();

    await userEvent.click(screen.getByTestId("session-header-actions"));
    await userEvent.click(screen.getByTestId("session-delete-action"));
    await userEvent.click(screen.getByTestId("session-delete-confirm"));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("delete failed"));
    expect(screen.getByTestId("session-delete-dialog")).toBeInTheDocument();
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it("shows mobile delete inside the header actions sheet", async () => {
    installMatchMediaMock(true);
    renderPage();

    await userEvent.click(screen.getByTestId("session-header-actions"));

    expect(screen.getByTestId("session-actions-menu")).toBeInTheDocument();
    expect(screen.getByTestId("session-delete-action")).toHaveTextContent("Delete session");
  });

  it("disables delete while a session mutation is in progress", async () => {
    storeMock.useAgentSessionStore.setState({ isMutating: true });
    renderPage();

    await userEvent.click(screen.getByTestId("session-header-actions"));

    expect(screen.getByTestId("session-delete-action")).toBeDisabled();
  });

  it("navigates to the artifact detail page on artifact click", async () => {
    renderPage();

    await activateTab(/Artifacts 2/);
    fireEvent.click(screen.getByTestId("agent-session-artifact"));

    expect(navigateMock).toHaveBeenCalledWith("/initiatives/quality-gates");
  });

  it("opens a proposal in its target proposal review tab", async () => {
    renderPage();

    await activateTab(/Artifacts 2/);
    fireEvent.click(screen.getByTestId("agent-session-proposal-artifact"));

    expect(navigateMock).toHaveBeenCalledWith("/initiatives/quality-gates?tab=proposals");
  });

  it("renders not-found when session is missing", async () => {
    const loadSession = vi.fn().mockRejectedValue(new Error("missing"));
    storeMock.useAgentSessionStore.setState({ sessions: [], status: "success", loadSession });

    renderPage("/sessions/missing");

    await waitFor(() => {
      expect(screen.getByText(/Session not found/)).toBeInTheDocument();
    });
  });
});

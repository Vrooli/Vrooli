import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";
import type { AgentSession } from "../types";
import { SessionDetailsPage } from "./SessionDetailsPage";
import { installMatchMediaMock, renderWithProviders } from "../test-utils";

const storeMock = vi.hoisted(() => {
  const initialState = {
    sessions: [],
    status: "idle",
    error: null,
    isMutating: false,
    isRefreshing: false,
    loadSession: vi.fn(),
    continueSession: vi.fn(),
    refreshSession: vi.fn(),
    cancelSession: vi.fn(),
    deleteSession: vi.fn(),
    applyProposal: vi.fn(),
  };
  let state: Record<string, unknown> = { ...initialState };
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
  return { initialState, useAgentSessionStore };
});

vi.mock("../stores", () => ({
  agentSessionStoreInitialState: storeMock.initialState,
  useAgentSessionStore: storeMock.useAgentSessionStore,
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
};

function renderPage(initialPath = "/sessions/sess_meta") {
  return renderWithProviders(
    <Routes>
      <Route path="/sessions/:sessionId" element={<SessionDetailsPage />} />
    </Routes>,
    { initialEntries: [initialPath] },
  );
}

async function activateTab(name: string | RegExp) {
  await userEvent.click(screen.getByRole("tab", { name }));
}

describe("SessionDetailsPage", () => {
  beforeEach(() => {
    installMatchMediaMock(false);
    navigateMock.mockReset();
    storeMock.useAgentSessionStore.reset();
    storeMock.useAgentSessionStore.setState({
      ...storeMock.initialState,
      sessions: [SESSION],
      status: "success",
    });
  });

  it("renders header, conversation, proposals, and artifacts", async () => {
    renderPage();

    expect(screen.getByText("Plan quality work")).toBeInTheDocument();
    expect(screen.getByText("Plan it.")).toBeInTheDocument();
    expect(screen.getByText("On it.")).toBeInTheDocument();
    expect(screen.getByText("Apply this plan.")).toBeInTheDocument();
    await activateTab(/Artifacts 1/);
    expect(screen.getByText("Quality gates")).toBeInTheDocument();
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

  it("defaults the desktop inspector to proposals when a proposal is ready", () => {
    renderPage();

    expect(screen.getByRole("tab", { name: /Proposals 1/ })).toHaveAttribute("data-state", "active");
  });

  it("uses full-page mobile tabs instead of stacked secondary sections", async () => {
    installMatchMediaMock(true);
    renderPage();

    expect(screen.getByRole("tab", { name: "Conversation" })).toHaveAttribute("data-state", "active");
    expect(screen.queryByTestId("detail-mobile-actions-fab")).toBeNull();
    expect(screen.getByTestId("session-mobile-header-actions")).toBeInTheDocument();
    expect(screen.getByText("Plan it.")).toBeVisible();
    expect(screen.queryByText("Apply this plan.")).toBeNull();
    expect(screen.getByTestId("agent-session-composer").parentElement?.parentElement).toHaveClass("fixed");

    await activateTab(/Proposals 1/);

    expect(screen.getByText("Apply this plan.")).toBeVisible();
  });

  it("puts mobile session actions behind the header ellipsis", async () => {
    installMatchMediaMock(true);
    const refreshSession = vi.fn().mockResolvedValue(SESSION);
    storeMock.useAgentSessionStore.setState({ refreshSession });

    renderPage();

    await userEvent.click(screen.getByTestId("session-mobile-header-actions"));
    expect(screen.getByTestId("session-mobile-actions-sheet")).toBeInTheDocument();

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

  it("invokes refresh and cancel actions", async () => {
    const refreshSession = vi.fn().mockResolvedValue(SESSION);
    const cancelSession = vi.fn().mockResolvedValue(SESSION);
    storeMock.useAgentSessionStore.setState({ refreshSession, cancelSession });

    renderPage();

    fireEvent.click(screen.getByTestId("session-refresh"));
    fireEvent.click(screen.getByTestId("session-cancel"));

    await waitFor(() => expect(refreshSession).toHaveBeenCalledWith("sess_meta"));
    await waitFor(() => expect(cancelSession).toHaveBeenCalledWith("sess_meta"));
  });

  it("keeps desktop delete in the header ellipsis menu", async () => {
    renderPage();

    expect(screen.getByTestId("session-refresh")).toBeInTheDocument();
    expect(screen.getByTestId("session-cancel")).toBeInTheDocument();
    expect(screen.queryByTestId("session-delete-action")).toBeNull();

    await userEvent.click(screen.getByTestId("session-desktop-header-actions"));

    expect(screen.getByTestId("session-desktop-actions-menu")).toBeInTheDocument();
    expect(screen.getByTestId("session-delete-action")).toHaveTextContent("Delete session");
  });

  it("requires strong confirmation before deleting and navigates away on success", async () => {
    const deleteSession = vi.fn().mockResolvedValue(undefined);
    storeMock.useAgentSessionStore.setState({ deleteSession });
    renderPage();

    await userEvent.click(screen.getByTestId("session-desktop-header-actions"));
    await userEvent.click(screen.getByTestId("session-delete-action"));

    expect(screen.getByTestId("session-delete-dialog")).toBeInTheDocument();
    expect(screen.getByText(/Created backlog items, initiatives, captures, files, and agent activity records stay/)).toBeInTheDocument();
    expect(screen.getByTestId("session-delete-confirm")).toBeDisabled();

    await userEvent.type(screen.getByPlaceholderText("Plan quality work"), "Plan quality work");
    expect(screen.getByTestId("session-delete-confirm")).toBeEnabled();
    await userEvent.click(screen.getByTestId("session-delete-confirm"));

    await waitFor(() => expect(deleteSession).toHaveBeenCalledWith("sess_meta"));
    expect(navigateMock).toHaveBeenCalled();
  });

  it("keeps the page open and shows an alert when delete fails", async () => {
    const deleteSession = vi.fn().mockRejectedValue(new Error("delete failed"));
    storeMock.useAgentSessionStore.setState({ deleteSession });
    renderPage();

    await userEvent.click(screen.getByTestId("session-desktop-header-actions"));
    await userEvent.click(screen.getByTestId("session-delete-action"));
    await userEvent.type(screen.getByPlaceholderText("Plan quality work"), "Plan quality work");
    await userEvent.click(screen.getByTestId("session-delete-confirm"));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("delete failed"));
    expect(screen.getByTestId("session-delete-dialog")).toBeInTheDocument();
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it("shows mobile delete inside the header actions sheet", async () => {
    installMatchMediaMock(true);
    renderPage();

    await userEvent.click(screen.getByTestId("session-mobile-header-actions"));

    expect(screen.getByTestId("session-mobile-actions-sheet")).toBeInTheDocument();
    expect(screen.getByTestId("session-delete-action")).toHaveTextContent("Delete session");
  });

  it("disables delete while a session mutation is in progress", async () => {
    storeMock.useAgentSessionStore.setState({ isMutating: true });
    renderPage();

    await userEvent.click(screen.getByTestId("session-desktop-header-actions"));

    expect(screen.getByTestId("session-delete-action")).toBeDisabled();
  });

  it("applies a proposal", async () => {
    const applyProposal = vi.fn().mockResolvedValue([]);
    storeMock.useAgentSessionStore.setState({ applyProposal });

    renderPage();

    fireEvent.click(screen.getByTestId("agent-session-apply-proposal"));

    await waitFor(() => {
      expect(applyProposal).toHaveBeenCalledWith("sess_meta", "prop-1");
    });
  });

  it("navigates to the artifact detail page on artifact click", async () => {
    renderPage();

    await activateTab(/Artifacts 1/);
    fireEvent.click(screen.getByTestId("agent-session-artifact"));

    expect(navigateMock).toHaveBeenCalledWith("/initiatives/quality-gates");
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

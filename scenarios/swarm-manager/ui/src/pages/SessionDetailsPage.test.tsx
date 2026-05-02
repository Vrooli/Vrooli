import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { agentSessionStoreInitialState, useAgentSessionStore } from "../stores";
import type { AgentSession } from "../types";
import { SessionDetailsPage } from "./SessionDetailsPage";
import { installMatchMediaMock, renderWithProviders } from "../test-utils";

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
    { id: "msg-2", role: "assistant", content: "On it.", createdAt: "2026-05-01T12:01:00Z", attachmentIds: [] },
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

describe("SessionDetailsPage", () => {
  beforeEach(() => {
    navigateMock.mockReset();
    useAgentSessionStore.setState({
      ...agentSessionStoreInitialState,
      sessions: [SESSION],
      status: "success",
    });
  });

  it("renders header, conversation, proposals, and artifacts", () => {
    renderPage();

    expect(screen.getByText("Plan quality work")).toBeInTheDocument();
    expect(screen.getByText("Plan it.")).toBeInTheDocument();
    expect(screen.getByText("On it.")).toBeInTheDocument();
    expect(screen.getByText("Apply this plan.")).toBeInTheDocument();
    expect(screen.getByText("Quality gates")).toBeInTheDocument();
  });

  it("sends a continuation on Ctrl+Enter", async () => {
    const continueSession = vi.fn().mockResolvedValue(SESSION);
    useAgentSessionStore.setState({ continueSession });

    renderPage();

    const composer = screen.getByTestId("agent-session-composer");
    fireEvent.change(composer, { target: { value: "Next step" } });
    fireEvent.keyDown(composer, { key: "Enter", ctrlKey: true });

    await waitFor(() => {
      expect(continueSession).toHaveBeenCalledWith({ sessionId: "sess_meta", message: "Next step" });
    });
  });

  it("invokes refresh and cancel actions", async () => {
    const refreshSession = vi.fn().mockResolvedValue(SESSION);
    const cancelSession = vi.fn().mockResolvedValue(SESSION);
    useAgentSessionStore.setState({ refreshSession, cancelSession });

    renderPage();

    fireEvent.click(screen.getByTestId("session-refresh"));
    fireEvent.click(screen.getByTestId("session-cancel"));

    await waitFor(() => expect(refreshSession).toHaveBeenCalledWith("sess_meta"));
    await waitFor(() => expect(cancelSession).toHaveBeenCalledWith("sess_meta"));
  });

  it("applies a proposal", async () => {
    const applyProposal = vi.fn().mockResolvedValue([]);
    useAgentSessionStore.setState({ applyProposal });

    renderPage();

    fireEvent.click(screen.getByTestId("agent-session-apply-proposal"));

    await waitFor(() => {
      expect(applyProposal).toHaveBeenCalledWith("sess_meta", "prop-1");
    });
  });

  it("navigates to the artifact detail page on artifact click", () => {
    renderPage();

    fireEvent.click(screen.getByTestId("agent-session-artifact"));

    expect(navigateMock).toHaveBeenCalledWith("/initiatives/quality-gates");
  });

  it("renders not-found when session is missing", async () => {
    const loadSession = vi.fn().mockRejectedValue(new Error("missing"));
    useAgentSessionStore.setState({ sessions: [], status: "success", loadSession });

    renderPage("/sessions/missing");

    await waitFor(() => {
      expect(screen.getByText(/Session not found/)).toBeInTheDocument();
    });
  });
});

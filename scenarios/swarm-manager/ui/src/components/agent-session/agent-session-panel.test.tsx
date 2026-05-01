import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { AgentSessionPanel } from "./agent-session-panel";
import { agentSessionStoreInitialState, useAgentSessionStore } from "../../stores";
import type { AgentSession, AgentSessionArtifact } from "../../types";

vi.mock("../ui/floating-panel", () => ({
  FloatingPanel: ({ children, isOpen, title }: { children: React.ReactNode; isOpen: boolean; title: string }) =>
    isOpen ? (
      <div data-testid="floating-panel">
        <h2>{title}</h2>
        {children}
      </div>
    ) : null,
}));

const ARTIFACT: AgentSessionArtifact = {
  id: "art-1",
  sessionId: "sess_1",
  artifactType: "initiative",
  action: "created",
  entityRef: "quality-gates",
  title: "Quality Gates",
  createdAt: "2026-05-01T12:03:00Z",
};

const SESSION: AgentSession = {
  id: "sess_1",
  title: "Plan quality gates",
  kind: "meta_orchestration",
  status: "proposal_ready",
  skillId: "swarm-manager-meta-orchestrator",
  runId: "run-1",
  taskId: "task-1",
  profileKey: "swarm-manager/default",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:02:00Z",
  messages: [
    {
      id: "msg-1",
      role: "user",
      content: "Plan this initiative.",
      attachmentIds: [],
      createdAt: "2026-05-01T12:00:00Z",
    },
    {
      id: "msg-2",
      role: "assistant",
      content: "Here is a proposal.",
      attachmentIds: [],
      createdAt: "2026-05-01T12:01:00Z",
    },
  ],
  proposals: [
    {
      id: "prop-1",
      kind: "backlog_batch_import",
      status: "ready",
      summary: "Create quality gate work.",
      payloadJson: "{\"items\":[]}",
      createdAt: "2026-05-01T12:02:00Z",
      updatedAt: "2026-05-01T12:02:00Z",
    },
  ],
  artifacts: [ARTIFACT],
};

function resetStore(overrides: Partial<ReturnType<typeof useAgentSessionStore.getState>> = {}) {
  const continueSession = vi.fn().mockResolvedValue(SESSION);
  const refreshSession = vi.fn().mockResolvedValue(SESSION);
  const cancelSession = vi.fn().mockResolvedValue({ ...SESSION, status: "canceled" });
  const applyProposal = vi.fn().mockResolvedValue([ARTIFACT]);
  const setActiveSession = vi.fn((session: AgentSession | null) => {
    useAgentSessionStore.setState({ activeSession: session });
  });

  useAgentSessionStore.setState({
    ...agentSessionStoreInitialState,
    sessions: [SESSION],
    activeSession: SESSION,
    status: "success",
    continueSession,
    refreshSession,
    cancelSession,
    applyProposal,
    setActiveSession,
    ...overrides,
  });

  return { continueSession, refreshSession, cancelSession, applyProposal, setActiveSession };
}

describe("AgentSessionPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders session metadata, messages, proposals, and artifacts", () => {
    render(<AgentSessionPanel />);

    expect(screen.getAllByText("Plan quality gates")).toHaveLength(2);
    expect(screen.getByText("Plan work")).toBeInTheDocument();
    expect(screen.getByText("Plan this initiative.")).toBeInTheDocument();
    expect(screen.getByText("Here is a proposal.")).toBeInTheDocument();
    expect(screen.getByText("Create quality gate work.")).toBeInTheDocument();
    expect(screen.getByText("Quality Gates")).toBeInTheDocument();
  });

  it("continues the active session from the composer", async () => {
    const { continueSession } = resetStore();
    render(<AgentSessionPanel />);

    fireEvent.change(screen.getByTestId("agent-session-composer"), {
      target: { value: "Please keep going." },
    });
    fireEvent.click(screen.getByTestId("agent-session-send"));

    await waitFor(() => {
      expect(continueSession).toHaveBeenCalledWith({
        sessionId: "sess_1",
        message: "Please keep going.",
      });
    });
  });

  it("applies a ready proposal", async () => {
    const { applyProposal } = resetStore();
    render(<AgentSessionPanel />);

    fireEvent.click(screen.getByTestId("agent-session-apply-proposal"));

    await waitFor(() => {
      expect(applyProposal).toHaveBeenCalledWith("sess_1", "prop-1");
    });
  });

  it("opens artifacts through the callback", () => {
    const onOpenArtifact = vi.fn();
    render(<AgentSessionPanel onOpenArtifact={onOpenArtifact} />);

    fireEvent.click(screen.getByTestId("agent-session-artifact"));

    expect(onOpenArtifact).toHaveBeenCalledWith(ARTIFACT);
  });

  it("does not render without an active session", () => {
    resetStore({ activeSession: null });

    render(<AgentSessionPanel />);

    expect(screen.queryByTestId("floating-panel")).toBeNull();
  });
});

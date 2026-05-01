import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { agentSessionStoreInitialState, useAgentSessionStore } from "../../stores";
import type { AgentSession } from "../../types";
import { AttributionChip } from "./AttributionChip";

const SESSION: AgentSession = {
  id: "sess_1",
  title: "Plan quality gates",
  kind: "meta_orchestration",
  status: "waiting_for_user",
  skillId: "swarm-manager-meta-orchestrator",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:00:00Z",
  messages: [],
  proposals: [],
  artifacts: [],
};

describe("AttributionChip", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAgentSessionStore.setState(agentSessionStoreInitialState);
  });

  it("opens the attributed session when session provenance exists", () => {
    const openSession = vi.fn();
    useAgentSessionStore.setState({
      ...agentSessionStoreInitialState,
      sessions: [SESSION],
      openSession,
    });

    render(
      <AttributionChip
        attribution={{
          type: "agent",
          runId: "run-1",
          profileKey: "swarm-manager/default",
          sessionId: "sess_1",
          sessionKind: "meta_orchestration",
        }}
      />,
    );

    fireEvent.click(screen.getByTestId("attribution-chip"));

    expect(screen.getByText("Created by Plan quality gates")).toBeInTheDocument();
    expect(openSession).toHaveBeenCalledWith("sess_1");
  });

  it("renders non-session agent provenance as read-only", () => {
    render(
      <AttributionChip
        attribution={{
          type: "agent",
          runId: "run-1",
          profileKey: "swarm-manager/default",
        }}
      />,
    );

    expect(screen.getByText("Created by agent:swarm-manager/default/run-1")).toBeInTheDocument();
    expect(screen.getByTestId("attribution-chip").tagName).toBe("SPAN");
  });

  it("renders operator provenance as read-only", () => {
    render(<AttributionChip attribution={{ type: "operator" }} />);

    expect(screen.getByText("Created by operator")).toBeInTheDocument();
  });
});

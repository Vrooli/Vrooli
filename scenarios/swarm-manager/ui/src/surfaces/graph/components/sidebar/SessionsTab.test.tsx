import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { agentSessionStoreInitialState, useAgentSessionStore } from "../../../../stores";
import type { AgentSession } from "../../../../types";
import { SessionsTab } from "./SessionsTab";

const navigateMock = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => navigateMock };
});
import { applySessionFilters, applySessionSort } from "./session-list-utils";
import type { SessionFilters, SortConfig } from "./types";

const BASE_FILTERS: SessionFilters = {
  statuses: [],
  kinds: [],
  activeOnly: false,
  hasProposals: false,
  hasAppliedArtifacts: false,
};

const RECENCY_SORT: SortConfig = { field: "recency", direction: "desc" };

const META_SESSION: AgentSession = {
  id: "sess_meta",
  title: "Plan quality work",
  kind: "meta_orchestration",
  status: "running",
  skillId: "swarm-manager-meta-orchestrator",
  taskId: "task-meta",
  runId: "run-meta",
  profileKey: "swarm-manager/default",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:10:00Z",
  messages: [
    { id: "msg-1", role: "user", content: "Plan it.", createdAt: "2026-05-01T12:00:00Z", attachmentIds: [] },
  ],
  proposals: [],
  artifacts: [],
};

const AUTHORING_SESSION: AgentSession = {
  ...META_SESSION,
  id: "sess_mode",
  title: "Author phased execution mode",
  kind: "operating_mode_authoring",
  status: "proposal_ready",
  skillId: "swarm-manager-operating-mode-authoring",
  taskId: "task-mode",
  runId: "run-mode",
  updatedAt: "2026-05-01T12:20:00Z",
  proposals: [
    {
      id: "prop-1",
      kind: "operating_mode_draft",
      status: "ready",
      summary: "Draft the mode.",
      payloadJson: "{\"mode\":\"phased\"}",
      createdAt: "2026-05-01T12:15:00Z",
      updatedAt: "2026-05-01T12:20:00Z",
    },
  ],
  artifacts: [
    {
      id: "art-1",
      sessionId: "sess_mode",
      artifactType: "operating_mode_proposal",
      action: "created",
      entityRef: "phased-mode",
      title: "Phased mode",
      createdAt: "2026-05-01T12:21:00Z",
    },
  ],
};

const OPERATIONS_SESSION: AgentSession = {
  ...META_SESSION,
  id: "sess_ops",
  title: "Manage stalled initiatives",
  kind: "swarm_operations",
  status: "waiting_for_user",
  skillId: "swarm-manager-operations-session",
  taskId: "task-ops",
  runId: "run-ops",
  updatedAt: "2026-05-01T12:30:00Z",
};

describe("SessionsTab", () => {
  beforeEach(() => {
    navigateMock.mockReset();
    useAgentSessionStore.setState({
      ...agentSessionStoreInitialState,
      sessions: [META_SESSION, AUTHORING_SESSION, OPERATIONS_SESSION],
      status: "success",
    });
  });

  it("filters by active sessions, kind, proposals, and applied artifacts", () => {
    const sessions = [META_SESSION, AUTHORING_SESSION, OPERATIONS_SESSION];
    expect(applySessionFilters(sessions, { ...BASE_FILTERS, activeOnly: true })).toHaveLength(3);
    expect(applySessionFilters(sessions, { ...BASE_FILTERS, kinds: ["meta_orchestration"] })).toEqual([META_SESSION]);
    expect(applySessionFilters(sessions, { ...BASE_FILTERS, kinds: ["swarm_operations"] })).toEqual([OPERATIONS_SESSION]);
    expect(applySessionFilters(sessions, { ...BASE_FILTERS, hasProposals: true })).toEqual([AUTHORING_SESSION]);
    expect(applySessionFilters(sessions, { ...BASE_FILTERS, hasAppliedArtifacts: true })).toEqual([AUTHORING_SESSION]);
  });

  it("sorts sessions by recency and alphabetically", () => {
    expect(applySessionSort([META_SESSION, AUTHORING_SESSION], RECENCY_SORT).map((session) => session.id)).toEqual([
      "sess_mode",
      "sess_meta",
    ]);
    expect(applySessionSort([META_SESSION, AUTHORING_SESSION], { field: "alphabetical", direction: "asc" }).map((session) => session.id)).toEqual([
      "sess_mode",
      "sess_meta",
    ]);
  });

  it("renders sidebar cards and opens the selected session", async () => {
    const onOpenSession = vi.fn();
    render(<SessionsTab searchQuery="" filters={BASE_FILTERS} sort={RECENCY_SORT} onOpenSession={onOpenSession} />);

    expect(screen.getByText("Manage stalled initiatives")).toBeInTheDocument();
    expect(screen.getByText("Swarm operations")).toBeInTheDocument();
    expect(screen.getByText("Author phased execution mode")).toBeInTheDocument();
    expect(screen.getByText("Plan quality work")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Author phased execution mode"));

    await waitFor(() => {
      expect(navigateMock).toHaveBeenCalledWith("/sessions/sess_mode");
    });
    expect(onOpenSession).toHaveBeenCalledWith("sess_mode");
  });

  it("renders an empty state when filters remove all sessions", () => {
    render(
      <SessionsTab
        searchQuery=""
        filters={{ ...BASE_FILTERS, statuses: ["failed"] }}
        sort={RECENCY_SORT}
      />,
    );

    expect(screen.getByText("No sessions match your filters.")).toBeInTheDocument();
  });
});

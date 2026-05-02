import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IAgentSessionService } from "../services/agent-session-service";
import type { AgentSession, AgentSessionArtifact } from "../types";
import {
  artifactEntityKey,
  resetAgentSessionStoreService,
  selectActiveAgentSessions,
  setAgentSessionStoreService,
  useAgentSessionStore,
} from "./agent-session-store";

const SESSION_A: AgentSession = {
  id: "sess_a",
  title: "Plan A",
  kind: "meta_orchestration",
  status: "running",
  skillId: "swarm-manager-meta-orchestrator",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:00:00Z",
  messages: [],
  proposals: [],
  artifacts: [],
};

const SESSION_B: AgentSession = {
  ...SESSION_A,
  id: "sess_b",
  title: "Plan B",
  status: "complete",
  updatedAt: "2026-05-01T13:00:00Z",
};

const ARTIFACT: AgentSessionArtifact = {
  id: "art-1",
  sessionId: "sess_a",
  artifactType: "initiative",
  action: "created",
  entityRef: "quality-gates",
  createdAt: "2026-05-01T12:02:00Z",
};

describe("agent-session-store", () => {
  let service: IAgentSessionService;

  beforeEach(() => {
    window.localStorage.clear();
    service = {
      list: vi.fn().mockResolvedValue([SESSION_A, SESSION_B]),
      get: vi.fn().mockResolvedValue(SESSION_A),
      create: vi.fn().mockResolvedValue(SESSION_A),
      continue: vi.fn().mockResolvedValue({ ...SESSION_A, messages: [{ id: "msg-1", role: "user", content: "Hi", createdAt: "2026-05-01T12:00:00Z", attachmentIds: [] }] }),
      refresh: vi.fn().mockResolvedValue({ ...SESSION_A, status: "waiting_for_user" }),
      cancel: vi.fn().mockResolvedValue({ ...SESSION_A, status: "canceled" }),
      applyProposal: vi.fn().mockResolvedValue({ session: { ...SESSION_A, status: "waiting_for_user" }, artifacts: [ARTIFACT] }),
      listArtifacts: vi.fn().mockResolvedValue([ARTIFACT]),
      getArtifactsByEntity: vi.fn().mockResolvedValue([ARTIFACT]),
    };
    setAgentSessionStoreService(service);
    useAgentSessionStore.getState().reset();
  });

  afterEach(() => {
    resetAgentSessionStoreService();
    useAgentSessionStore.getState().reset();
  });

  it("fetches and sorts sessions", async () => {
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });

    const state = useAgentSessionStore.getState();
    expect(service.list).toHaveBeenCalledWith(undefined);
    expect(state.status).toBe("success");
    expect(state.sessions.map((session) => session.id)).toEqual(["sess_b", "sess_a"]);
  });

  it("loads, creates, continues, refreshes, cancels, and applies sessions", async () => {
    const loaded = await useAgentSessionStore.getState().loadSession("sess_a");
    expect(loaded.id).toBe("sess_a");
    expect(useAgentSessionStore.getState().sessions.map((s) => s.id)).toContain("sess_a");

    await useAgentSessionStore.getState().createSession({
      kind: "meta_orchestration",
      title: "Plan",
      initialMessage: "Plan this.",
    });
    expect(service.create).toHaveBeenCalled();

    const continued = await useAgentSessionStore.getState().continueSession({
      sessionId: "sess_a",
      message: "Continue.",
    });
    expect(continued.messages).toHaveLength(1);

    const refreshed = await useAgentSessionStore.getState().refreshSession("sess_a");
    expect(refreshed.status).toBe("waiting_for_user");

    const artifacts = await useAgentSessionStore.getState().applyProposal("sess_a", "prop-1");
    expect(artifacts).toEqual([ARTIFACT]);

    const canceled = await useAgentSessionStore.getState().cancelSession("sess_a");
    expect(canceled.status).toBe("canceled");
    expect(useAgentSessionStore.getState().isMutating).toBe(false);
  });

  it("loads artifacts by entity and exposes active selectors", async () => {
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });

    const artifacts = await useAgentSessionStore.getState().loadArtifactsByEntity("initiative", "quality-gates");

    expect(artifacts).toEqual([ARTIFACT]);
    expect(useAgentSessionStore.getState().artifactsByEntity[artifactEntityKey("initiative", "quality-gates")]).toEqual([
      ARTIFACT,
    ]);
    expect(selectActiveAgentSessions(useAgentSessionStore.getState()).map((session) => session.id)).toEqual([
      "sess_a",
    ]);
  });

  it("records errors without clearing existing session data", async () => {
    vi.mocked(service.list).mockResolvedValue([SESSION_A]);
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });

    vi.mocked(service.list).mockRejectedValue(new Error("network"));
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });

    const state = useAgentSessionStore.getState();
    expect(state.sessions).toHaveLength(1);
    expect(state.status).toBe("success");
    expect(state.error?.message).toBe("network");
  });
});

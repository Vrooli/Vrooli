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
  title: "Manage Swarm",
  kind: "swarm_operations",
  skillId: "swarm-manager-operations-session",
  status: "complete",
  updatedAt: "2026-05-01T13:00:00Z",
};

const ARTIFACT: AgentSessionArtifact = {
  id: "art-1",
  sessionId: "sess_a",
  artifactType: "milestone",
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
      previewPrompt: vi.fn().mockResolvedValue({ prompt: "", initial: true }),
      get: vi.fn().mockResolvedValue(SESSION_A),
      getStartupBrief: vi.fn().mockResolvedValue({
        type: "startup_brief",
        ref: "startup_brief/meta_orchestration",
        title: "Startup brief",
        summary: "Use this first.",
        selectedAt: "2026-05-01T12:00:00Z",
      }),
      create: vi.fn().mockResolvedValue(SESSION_A),
		changeKind: vi.fn().mockResolvedValue({ session: { ...SESSION_A, kind: "swarm_operations" }, droppedContextRefs: [], starterJobCleared: false }),
      start: vi.fn().mockResolvedValue({ ...SESSION_A, messages: [{ id: "msg-0", role: "user", content: "Start", createdAt: "2026-05-01T12:00:00Z", attachmentIds: [] }] }),
      continue: vi.fn().mockResolvedValue({ ...SESSION_A, messages: [{ id: "msg-1", role: "user", content: "Hi", createdAt: "2026-05-01T12:00:00Z", attachmentIds: [] }] }),
      uploadAttachments: vi.fn().mockResolvedValue([]),
      listEvents: vi.fn().mockResolvedValue({ events: [], hasMore: false, nextAfterSequence: 0n }),
      refresh: vi.fn().mockResolvedValue({ ...SESSION_A, status: "waiting_for_user" }),
      cancel: vi.fn().mockResolvedValue({ ...SESSION_A, status: "canceled" }),
      delete: vi.fn().mockResolvedValue("sess_a"),
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

  it("does not let filtered refreshes replace the canonical session list", async () => {
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });

    vi.mocked(service.list).mockResolvedValue([SESSION_A]);
    await useAgentSessionStore.getState().fetchSessions({ activeOnly: true }, { force: true });

    const state = useAgentSessionStore.getState();
    expect(service.list).toHaveBeenLastCalledWith({ activeOnly: true });
    expect(state.sessions.map((session) => session.id)).toEqual(["sess_b", "sess_a"]);
    expect(window.localStorage.getItem("swarm-manager.agent-sessions.v1")).toContain("sess_b");
  });

  it("fetches the canonical list even while a detail session is refreshing", async () => {
    useAgentSessionStore.setState({
      sessions: [SESSION_A],
      status: "success",
      isRefreshing: true,
      lastFetchedAt: Date.now(),
    });

    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });

    const state = useAgentSessionStore.getState();
    expect(service.list).toHaveBeenCalledWith(undefined);
    expect(state.sessions.map((session) => session.id)).toEqual(["sess_b", "sess_a"]);
  });

  it("loads, creates, continues, refreshes, cancels, and applies sessions", async () => {
    const loaded = await useAgentSessionStore.getState().loadSession("sess_a");
    expect(loaded.id).toBe("sess_a");
    expect(useAgentSessionStore.getState().sessions.map((s) => s.id)).toContain("sess_a");

    await useAgentSessionStore.getState().createSession({
      kind: "swarm_operations",
      title: "Manage Swarm operations",
    });
    expect(service.create).toHaveBeenCalledWith({
      kind: "swarm_operations",
      title: "Manage Swarm operations",
    });

	const changed = await useAgentSessionStore.getState().changeSessionKind({ sessionId: "sess_a", kind: "swarm_operations", contextRefs: [] });
	expect(changed.session.kind).toBe("swarm_operations");
	expect(service.changeKind).toHaveBeenCalledWith({ sessionId: "sess_a", kind: "swarm_operations", contextRefs: [] });

    const started = await useAgentSessionStore.getState().startSession({
      sessionId: "sess_a",
      message: "Start.",
    });
    expect(started.messages).toHaveLength(1);

    const continued = await useAgentSessionStore.getState().continueSession({
      sessionId: "sess_a",
      message: "Continue.",
    });
    expect(continued.messages).toHaveLength(1);

    const refreshed = await useAgentSessionStore.getState().refreshSession("sess_a");
    expect(refreshed.status).toBe("waiting_for_user");

    const artifacts = await useAgentSessionStore.getState().applyProposal("sess_a", "prop-1");
    expect(artifacts).toEqual([ARTIFACT]);

    await useAgentSessionStore.getState().listSessionEvents({ sessionId: "sess_a", afterSequence: 1n });
    expect(service.listEvents).toHaveBeenCalledWith({ sessionId: "sess_a", afterSequence: 1n });

    const canceled = await useAgentSessionStore.getState().cancelSession("sess_a");
    expect(canceled.status).toBe("canceled");
    expect(useAgentSessionStore.getState().isMutating).toBe(false);
  });

  it("loads artifacts by entity and exposes active selectors", async () => {
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });

    const artifacts = await useAgentSessionStore.getState().loadArtifactsByEntity("milestone", "quality-gates");

    expect(artifacts).toEqual([ARTIFACT]);
    expect(useAgentSessionStore.getState().artifactsByEntity[artifactEntityKey("milestone", "quality-gates")]).toEqual([
      ARTIFACT,
    ]);
    expect(selectActiveAgentSessions(useAgentSessionStore.getState()).map((session) => session.id)).toEqual(["sess_a"]);
  });

  it("deletes sessions from memory, entity artifact cache, and persisted storage", async () => {
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });
    await useAgentSessionStore.getState().loadArtifactsByEntity("milestone", "quality-gates");

    await useAgentSessionStore.getState().deleteSession("sess_a");

    const state = useAgentSessionStore.getState();
    expect(service.delete).toHaveBeenCalledWith("sess_a");
    expect(state.sessions.map((session) => session.id)).toEqual(["sess_b"]);
    expect(state.artifactsByEntity[artifactEntityKey("milestone", "quality-gates")]).toBeUndefined();
    expect(state.isMutating).toBe(false);
    expect(window.localStorage.getItem("swarm-manager.agent-sessions.v1")).toContain("sess_b");
    expect(window.localStorage.getItem("swarm-manager.agent-sessions.v1")).not.toContain("sess_a");
  });

  it("preserves session data when delete fails", async () => {
    await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });
    vi.mocked(service.delete).mockRejectedValue(new Error("delete failed"));

    await expect(useAgentSessionStore.getState().deleteSession("sess_a")).rejects.toThrow("delete failed");

    const state = useAgentSessionStore.getState();
    expect(state.sessions.map((session) => session.id)).toEqual(["sess_b", "sess_a"]);
    expect(state.error?.message).toBe("delete failed");
    expect(state.isMutating).toBe(false);
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

  // The session detail page polls every 3s for the whole life of a running
  // session, and almost every response is identical to the last. Replacing the
  // stored object anyway re-rendered the entire transcript — re-parsing every
  // message's markdown and remounting every mermaid diagram — to paint exactly
  // the same pixels. Identity is the contract that stops it, so it is tested
  // directly.
  describe("refreshSession identity", () => {
    it("keeps the same session object when the poll returns unchanged data", async () => {
      vi.mocked(service.list).mockResolvedValue([SESSION_A]);
      await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });
      const before = useAgentSessionStore.getState().sessions;

      vi.mocked(service.refresh).mockResolvedValue({ ...SESSION_A });
      await useAgentSessionStore.getState().refreshSession(SESSION_A.id);
      await useAgentSessionStore.getState().refreshSession(SESSION_A.id);

      const after = useAgentSessionStore.getState().sessions;
      expect(after).toBe(before);
      expect(after[0]).toBe(before[0]);
    });

    it("replaces the session object when the poll returns a new message", async () => {
      vi.mocked(service.list).mockResolvedValue([SESSION_A]);
      await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });
      const before = useAgentSessionStore.getState().sessions[0];

      vi.mocked(service.refresh).mockResolvedValue({
        ...SESSION_A,
        messages: [{ id: "msg-1", role: "assistant", content: "Done.", createdAt: "2026-05-01T12:05:00Z", attachmentIds: [] }],
      });
      await useAgentSessionStore.getState().refreshSession(SESSION_A.id);

      const after = useAgentSessionStore.getState().sessions[0];
      expect(after).not.toBe(before);
      expect(after?.messages).toHaveLength(1);
    });

    it("replaces the session object when only the status changes", async () => {
      vi.mocked(service.list).mockResolvedValue([SESSION_A]);
      await useAgentSessionStore.getState().fetchSessions(undefined, { force: true });
      const before = useAgentSessionStore.getState().sessions[0];

      vi.mocked(service.refresh).mockResolvedValue({ ...SESSION_A, status: "complete" });
      await useAgentSessionStore.getState().refreshSession(SESSION_A.id);

      const after = useAgentSessionStore.getState().sessions[0];
      expect(after).not.toBe(before);
      expect(after?.status).toBe("complete");
    });
  });
});

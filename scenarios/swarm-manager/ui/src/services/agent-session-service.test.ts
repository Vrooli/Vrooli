import { beforeEach, describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createAgentSessionService, type IAgentSessionService } from "./agent-session-service";

const SESSION_RESPONSE = {
  id: "sess_1",
  title: "Plan work",
  kind: "meta_orchestration",
  status: "running",
  skill_id: "swarm-manager-meta-orchestrator",
  run_id: "run-1",
  task_id: "task-1",
  profile_key: "swarm-manager/default",
  created_at: "2026-05-01T12:00:00Z",
  updated_at: "2026-05-01T12:01:00Z",
  messages: [
    {
      id: "msg-1",
      role: "user",
      content: "Plan this.",
      created_at: "2026-05-01T12:00:00Z",
    },
  ],
  proposals: [
    {
      id: "prop-1",
      kind: "backlog_batch_import",
      status: "ready",
      summary: "Create work.",
      payload_json: "{\"items\":[]}",
      created_at: "2026-05-01T12:01:00Z",
      updated_at: "2026-05-01T12:01:00Z",
    },
  ],
  artifacts: [],
};

const ARTIFACT_RESPONSE = {
  id: "art-1",
  session_id: "sess_1",
  artifact_type: "milestone",
  action: "created",
  entity_ref: "quality-gates",
  title: "Quality Gates",
  created_at: "2026-05-01T12:02:00Z",
};

describe("agent-session-service", () => {
  let mockApiClient: IApiClient;
  let service: IAgentSessionService;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    service = createAgentSessionService(mockApiClient);
  });

  it("lists sessions with filters and maps proto JSON", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({ sessions: [SESSION_RESPONSE] });

    const sessions = await service.list({
      kind: "meta_orchestration",
      activeOnly: true,
      limit: 25,
    });

    expect(mockApiClient.get).toHaveBeenCalledWith(
      "/agent-sessions?kind=meta_orchestration&active_only=true&limit=25"
    );
    expect(sessions[0]).toMatchObject({
      id: "sess_1",
      kind: "meta_orchestration",
      status: "running",
      runId: "run-1",
      messages: [{ id: "msg-1", role: "user", content: "Plan this." }],
    });
  });

  it("keeps known and future proposal kinds from invalidating the session list", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      sessions: [{
        ...SESSION_RESPONSE,
        proposals: [
          { ...SESSION_RESPONSE.proposals[0], kind: "mutation_list", status: "needs_revision" },
          { ...SESSION_RESPONSE.proposals[0], id: "prop-future", kind: "future_server_kind", status: "future_status" },
        ],
      }],
    });

    await expect(service.list()).resolves.toMatchObject([
      {
        proposals: [
          { kind: "mutation_list", status: "needs_revision" },
          { kind: "future_server_kind", status: "future_status" },
        ],
      },
    ]);
  });

	it("changes a draft kind and maps dropped context", async () => {
		vi.mocked(mockApiClient.patch).mockResolvedValue({
			session: { ...SESSION_RESPONSE, status: "draft", kind: "workflow_authoring", skill_id: "swarm-manager-workflow-authoring" },
			dropped_context_refs: [{ type: "execution", ref: "exec-1" }],
			starter_job_cleared: true,
		});
		const result = await service.changeKind({
			sessionId: "sess_1",
			kind: "workflow_authoring",
			contextRefs: [{ type: "execution", ref: "exec-1" }],
		});
		expect(mockApiClient.patch).toHaveBeenCalledWith("/agent-sessions/sess_1/kind", {
			session_id: "sess_1",
			kind: "workflow_authoring",
			context_refs: [{ type: "execution", ref: "exec-1" }],
		});
		expect(result).toMatchObject({
			session: { kind: "workflow_authoring" },
			droppedContextRefs: [{ type: "execution", ref: "exec-1" }],
			starterJobCleared: true,
		});
	});

  it("creates, starts, and continues sessions using backend field names", async () => {
    vi.mocked(mockApiClient.post).mockResolvedValue({ session: SESSION_RESPONSE });

    await service.create({
      kind: "swarm_operations",
      title: "Manage Swarm operations",
    });
    expect(mockApiClient.post).toHaveBeenCalledWith("/agent-sessions", {
      kind: "swarm_operations",
      title: "Manage Swarm operations",
    });

    await service.start({
      sessionId: "sess_1",
      message: "Draft it.",
      attachmentIds: ["att-0"],
    });
    expect(mockApiClient.post).toHaveBeenLastCalledWith("/agent-sessions/sess_1/start", {
      session_id: "sess_1",
      message: "Draft it.",
      attachment_ids: ["att-0"],
      context_refs: [],
    });

    await service.start({
      sessionId: "sess_1",
      message: "No briefing.",
      autoContextPolicy: "none",
    });
    expect(mockApiClient.post).toHaveBeenLastCalledWith("/agent-sessions/sess_1/start", {
      session_id: "sess_1",
      message: "No briefing.",
      attachment_ids: [],
      context_refs: [],
      auto_context_policy: "none",
    });

    await service.continue({
      sessionId: "sess_1",
      message: "Continue.",
      attachmentIds: ["att-1"],
    });
    expect(mockApiClient.post).toHaveBeenLastCalledWith("/agent-sessions/sess_1/continue", {
      session_id: "sess_1",
      message: "Continue.",
      attachment_ids: ["att-1"],
      context_refs: [],
    });
  });

  it("lists session events with cursor params", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      events: [
        {
          id: "evt-1",
          run_id: "run-1",
          sequence: "7",
          created_at: "2026-05-01T12:03:00Z",
          event_type: "tool_call",
          tool_name: "Read",
          input: "{\"file\":\"AGENTS.md\"}",
        },
      ],
      has_more: false,
      next_after_sequence: "7",
    });

    const result = await service.listEvents({ sessionId: "sess_1", afterSequence: 5n, limit: 25 });

    expect(mockApiClient.get).toHaveBeenCalledWith("/agent-sessions/sess_1/events?after_sequence=5&limit=25");
    expect(result.events[0]).toMatchObject({ eventType: "tool_call", toolName: "Read" });
    expect(result.nextAfterSequence).toBe(7n);
  });

  it("refreshes, cancels, applies proposals, and reads artifacts", async () => {
    vi.mocked(mockApiClient.post).mockResolvedValueOnce({ session: SESSION_RESPONSE });
    await expect(service.refresh("sess_1")).resolves.toMatchObject({ id: "sess_1" });
    expect(mockApiClient.post).toHaveBeenCalledWith("/agent-sessions/sess_1/refresh", {});

    vi.mocked(mockApiClient.post).mockResolvedValueOnce({ session: { ...SESSION_RESPONSE, status: "canceled" } });
    await expect(service.cancel("sess_1")).resolves.toMatchObject({ status: "canceled" });
    expect(mockApiClient.post).toHaveBeenCalledWith("/agent-sessions/sess_1/cancel", {});

    vi.mocked(mockApiClient.post).mockResolvedValueOnce({
      session: { ...SESSION_RESPONSE, status: "waiting_for_user" },
      artifacts: [ARTIFACT_RESPONSE],
    });
    const applied = await service.applyProposal("sess_1", "prop-1");
    expect(mockApiClient.post).toHaveBeenCalledWith("/agent-sessions/sess_1/proposals/prop-1/apply", {
      session_id: "sess_1",
      proposal_id: "prop-1",
    });
    expect(applied.artifacts).toHaveLength(1);

    vi.mocked(mockApiClient.get).mockResolvedValueOnce({ artifacts: [ARTIFACT_RESPONSE] });
    await expect(service.listArtifacts("sess_1")).resolves.toMatchObject([{ entityRef: "quality-gates" }]);
    expect(mockApiClient.get).toHaveBeenCalledWith("/agent-sessions/sess_1/artifacts");

    vi.mocked(mockApiClient.get).mockResolvedValueOnce({ artifacts: [ARTIFACT_RESPONSE] });
    await expect(service.getArtifactsByEntity("milestone", "quality-gates")).resolves.toHaveLength(1);
    expect(mockApiClient.get).toHaveBeenCalledWith(
      "/artifacts/by-entity?artifact_type=milestone&entity_ref=quality-gates"
    );
  });

  it("deletes a session through the session resource endpoint", async () => {
    vi.mocked(mockApiClient.delete).mockResolvedValue({ session_id: "sess_1" });

    await expect(service.delete("sess_1")).resolves.toBe("sess_1");

    expect(mockApiClient.delete).toHaveBeenCalledWith("/agent-sessions/sess_1");
  });
});

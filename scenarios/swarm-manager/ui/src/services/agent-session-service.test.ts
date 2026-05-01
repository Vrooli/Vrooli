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
  artifact_type: "initiative",
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

  it("creates and continues sessions using backend field names", async () => {
    vi.mocked(mockApiClient.post).mockResolvedValue({ session: SESSION_RESPONSE });

    await service.create({
      kind: "operating_mode_authoring",
      title: "Author mode",
      initialMessage: "Draft it.",
      initiative: "mode-work",
    });
    expect(mockApiClient.post).toHaveBeenCalledWith("/agent-sessions", {
      kind: "operating_mode_authoring",
      title: "Author mode",
      initial_message: "Draft it.",
      initiative: "mode-work",
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
    });
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
    await expect(service.getArtifactsByEntity("initiative", "quality-gates")).resolves.toHaveLength(1);
    expect(mockApiClient.get).toHaveBeenCalledWith(
      "/artifacts/by-entity?artifact_type=initiative&entity_ref=quality-gates"
    );
  });
});

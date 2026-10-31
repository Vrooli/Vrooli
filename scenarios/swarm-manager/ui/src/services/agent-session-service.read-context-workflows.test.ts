// Actual generated response validation/read adapters. Fixture annotations and
// historical correlations are display data, not current authenticated principals.
import { afterEach, describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createAgentSessionService } from "./agent-session-service";

const clients: IApiClient[] = [];
function fixture(data: unknown) {
  const api: IApiClient = { get: vi.fn().mockResolvedValue(data), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
  clients.push(api); return { api, service: createAgentSessionService(api) };
}
const session = { id: "historical-session", title: "Fixture owner", kind: "swarm_operations", status: "complete", skill_id: "fixture-skill", created_at: "created", updated_at: "updated" };
const artifact = { id: "historical-artifact", session_id: "historical-session", artifact_type: "file", action: "linked", entity_ref: "source/file.ts", created_at: "created" };
const brief = { type: "startup_brief", ref: "startup/latest", title: "Original brief", summary: "Exact retained summary", node_id: "fixture-node", metadata_json: '{"fixture":true}', selected_at: "selected" };
afterEach(() => {
  for (const api of clients.splice(0)) {
    expect(api.get).toHaveBeenCalledTimes(1);
    for (const method of ["post", "put", "patch", "delete"] as const) expect(api[method]).not.toHaveBeenCalled();
  }
  vi.restoreAllMocks();
});
describe("session history, context and artifact read compatibility", () => {
  it.each(["backlog_item", "goal", "capture"])("preserves canonical %s proposal target through the actual read adapter", async type => {
    const f = fixture({ session: { ...session, proposal_target: { type, ref: "fixture-ref", name: "Original target" } } });
    expect((await f.service.get("historical-session")).proposalTarget).toEqual({ type, ref: "fixture-ref", name: "Original target" });
    expect(f.api.get).toHaveBeenCalledWith("/agent-sessions/historical-session");
  });
  it.each(["agent", "operator"])("retains declared %s provenance, scoped context and attachment bytes without acquiring authority", async type => {
    const attribution = type === "agent" ? { type, run_id: "historical-run", task_id: "historical-task", profile_key: "historical-profile", session_id: "historical-session", session_kind: "swarm_operations", source: "fixture/source" } : { type };
    const expectedAttribution = type === "agent" ? { type, runId: "historical-run", taskId: "historical-task", profileKey: "historical-profile", sessionId: "historical-session", sessionKind: "swarm_operations", source: "fixture/source" } : { type };
    const f = fixture({ session: { ...session, task_id: "historical-task", run_id: "historical-run", profile_key: "historical-profile", failure_reason: "Original failure", created_by: attribution, starter_job_id: "fixture-job", staged_context_refs: [{ type: "goal", ref: "fixture-goal" }],
      messages: [{ id: "historical-message", role: "user", content: "Original message", created_at: "created", attachment_ids: ["historical-image"], context: [brief] }],
      proposals: [{ id: "historical-proposal", kind: "future_kind", status: "future_status", summary: "Original proposal", payload_json: '{"literal":"<script>"}', created_at: "created", updated_at: "updated", attribution }],
      artifacts: [{ ...artifact, title: "Original artifact", proposal_id: "historical-proposal", activity_id: "historical-activity", run_id: "historical-run", mutation_source: "fixture/source", attribution }],
      attachments: [{ id: "historical-image", filename: "Original image", content_type: "image/png", size_bytes: "3", created_at: "created" }],
    } });
    const result = await f.service.get("historical-session");
    expect(result).toMatchObject({ taskId: "historical-task", runId: "historical-run", profileKey: "historical-profile", failureReason: "Original failure", createdBy: expectedAttribution, starterJobId: "fixture-job", stagedContextRefs: [{ type: "goal", ref: "fixture-goal" }], messages: [{ id: "historical-message", role: "user", content: "Original message", attachmentIds: ["historical-image"], context: [{ type: "startup_brief", ref: "startup/latest", title: "Original brief", summary: "Exact retained summary", nodeId: "fixture-node", metadataJson: '{"fixture":true}', selectedAt: "selected" }] }], proposals: [{ kind: "future_kind", status: "future_status", payloadJson: '{"literal":"<script>"}', attribution: expectedAttribution }], artifacts: [{ artifactType: "file", action: "linked", title: "Original artifact", proposalId: "historical-proposal", activityId: "historical-activity", runId: "historical-run", mutationSource: "fixture/source", attribution: expectedAttribution }], attachments: [{ id: "historical-image", contentType: "image/png", sizeBytes: 3n, url: "/agent-sessions/historical-session/attachments/historical-image" }] });
    expect(f.api.get).toHaveBeenCalledWith("/agent-sessions/historical-session");
  });
  it("keeps absent ownership/context/proposals empty without synthesizing correlation", async () => {
    const f = fixture({ session }); const result = await f.service.get("historical-session");
    expect(result).toMatchObject({ messages: [], proposals: [], attachments: [], artifacts: [] });
    for (const key of ["taskId", "runId", "profileKey", "failureReason", "createdBy", "proposalTarget", "starterJobId", "stagedContextRefs"]) expect(result).not.toHaveProperty(key);
  });
  it("reads the retained startup brief without refreshing or publishing it", async () => {
    const f = fixture({ brief }); expect(await f.service.getStartupBrief("historical-session")).toEqual({ type: "startup_brief", ref: "startup/latest", title: "Original brief", summary: "Exact retained summary", nodeId: "fixture-node", metadataJson: '{"fixture":true}', selectedAt: "selected" });
    expect(f.api.get).toHaveBeenCalledWith("/agent-sessions/historical-session/startup-brief");
  });
  it("refuses a malformed retained brief without refresh fallback", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {}); const f = fixture({ brief: { ...brief, type: "unsupported" } });
    await expect(f.service.getStartupBrief("historical-session", false)).rejects.toThrow("Invalid agent session startup brief response");
    expect(f.api.get).toHaveBeenCalledWith("/agent-sessions/historical-session/startup-brief");
  });
  it("correlates entity artifact reads with exact encoded type/reference", async () => {
    const f = fixture({ artifacts: [artifact] }); expect(await f.service.getArtifactsByEntity("file", "source/file with space.ts")).toEqual([{ id: "historical-artifact", sessionId: "historical-session", artifactType: "file", action: "linked", entityRef: "source/file.ts", createdAt: "created" }]);
    expect(f.api.get).toHaveBeenCalledWith("/artifacts/by-entity?artifact_type=file&entity_ref=source%2Ffile+with+space.ts");
  });
  it.each([{ artifacts: [] }, { artifacts: [artifact] }])("retains exact selected session artifact catalog (%j)", async ({ artifacts }) => {
    const f = fixture({ artifacts }); const result = await f.service.listArtifacts("historical-session"); expect(result).toHaveLength(artifacts.length);
    if (artifacts.length) expect(result[0]).toMatchObject({ id: "historical-artifact", sessionId: "historical-session", artifactType: "file", action: "linked" });
    expect(f.api.get).toHaveBeenCalledWith("/agent-sessions/historical-session/artifacts");
  });
  it("refuses an unsupported target without converting the current session to another owner", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {}); const f = fixture({ session: { ...session, proposal_target: { type: "execution", ref: "fixture-ref", name: "Original target" } } });
    await expect(f.service.get("historical-session")).rejects.toThrow("Invalid agent session response"); expect(f.api.get).toHaveBeenCalledWith("/agent-sessions/historical-session");
  });
});

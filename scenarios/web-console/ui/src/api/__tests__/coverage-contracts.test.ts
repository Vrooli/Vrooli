import { describe, expect, it, vi } from "vitest";
import { FleetState } from "@vrooli/proto-types/web-console/v1/machines/machines_pb";
import * as continuity from "../continuity";
import * as templates from "../grouptemplates";
import * as machines from "../machines";
import * as terminalScreen from "../terminalScreen";
import * as devices from "../devices";
import * as handoff from "../handoffrules";
import * as snippets from "../snippets";
import * as roles from "../workspaceRoles";
import * as workspace from "../workspace";

describe("small API domain wrappers", () => {
  it("decodes continuity integrity, receipts, and filtered search results", async () => {
    vi.spyOn(continuity.continuityClient, "integrity").mockResolvedValue({ sessions: 2n, conversationSessions: 3n, conversationEvents: 4n, checkpoints: 5n, workspacePanes: 6n, orphanConversations: 7n, orphanCheckpoints: 8n, orphanWorkspacePanes: 9n, uncatalogedConversations: 10n, generation: "g", eventContentHash: "h" } as never);
    vi.spyOn(continuity.continuityClient, "getReceipt").mockResolvedValue({ operationId: "op", sessionId: "s", command: "run", fromState: "queued", toState: "done", status: "succeeded", errorCode: "", createdAt: "a", completedAt: "b" } as never);
    const search = vi.spyOn(continuity.continuityClient, "search").mockResolvedValue({ matches: [{ eventId: "e", sessionId: "s", sequence: 4n, role: "assistant", createdAt: "a", excerpt: "x", lifecycleState: "active" }], totalMatches: 1n, distinctSessions: 1n, truncated: false } as never);

    await expect(continuity.getContinuityIntegrity()).resolves.toMatchObject({ sessions: 2, eventContentHash: "h" });
    await expect(continuity.getContinuityReceipt("op")).resolves.toMatchObject({ operationId: "op", completedAt: "b" });
    await expect(continuity.searchLocalContinuity("x", { after: "a", agentType: "codex", cwd: "/tmp", state: "active", limit: 4, sessionId: "s", agentSessionId: "as", title: "t", topicSummary: "summary" })).resolves.toMatchObject({ matches: [{ sequence: 4 }], totalMatches: 1 });
    expect(search).toHaveBeenCalledWith(expect.objectContaining({ query: "x", createdAfter: "a", lifecycleState: "active", limit: 4, topicSummary: "summary" }));
  });

  it("maps group-template defaults, role modes, upserts, and deletes", async () => {
    const template = { id: "t", name: "T", color: "blue", useCount: 3, roles: [{ label: "A", command: "run", workingDir: "/tmp", incomingPrompt: "{{payload}}", backend: "", targetId: "", startMode: "unexpected" }] };
    vi.spyOn(templates.groupTemplatesClient, "listTemplates").mockResolvedValue({ templates: [template] } as never);
    const upsert = vi.spyOn(templates.groupTemplatesClient, "upsertTemplate").mockResolvedValue({ template } as never);
    const remove = vi.spyOn(templates.groupTemplatesClient, "deleteTemplate").mockResolvedValue({} as never);
    await expect(templates.listGroupTemplates()).resolves.toMatchObject([{ use_count: 3, roles: [{ start_mode: "waiting" }] }]);
    await expect(templates.upsertGroupTemplate({ name: "T", roles: [{ label: "A", command: "run", working_dir: "/tmp", incoming_prompt: "{{payload}}", backend: "", target_id: "", start_mode: "eager" }] })).resolves.toMatchObject({ id: "t" });
    expect(upsert).toHaveBeenCalledWith(expect.objectContaining({ id: "", color: "", hasUseCount: false }));
    await templates.deleteGroupTemplate("t");
    expect(remove).toHaveBeenCalledWith({ id: "t" });
  });

  it("decodes fleet projections and delegates machine operations", async () => {
    const machine = { target: undefined, grant: undefined, drift: [{ kind: "version", name: "v", reason: "old" }], heartbeatAgeSeconds: 2n, manageable: true };
    vi.spyOn(machines.machineClient, "list").mockResolvedValue({ state: FleetState.EMPTY, machines: [machine], joinRequests: [], presets: [], message: "m", recoveryAction: "r", controlPlane: undefined } as never);
    const issue = vi.spyOn(machines.machineClient, "issueCode").mockResolvedValue({ code: "abc", expiresInSeconds: 30n } as never);
    const decide = vi.spyOn(machines.machineClient, "decide").mockResolvedValue({ message: "ok" } as never);
    const forget = vi.spyOn(machines.machineClient, "forget").mockResolvedValue({} as never);
    await expect(machines.listFleet()).resolves.toMatchObject({ status: "empty", machines: [{ heartbeatAgeSeconds: 2, grant: { effects: [] } }] });
    expect(machines.decodeMachine(machine as never)).toMatchObject({ manageable: true, drift: [{ name: "v" }] });
    await expect(machines.issueJoinCode()).resolves.toEqual({ code: "abc", expiresInSeconds: 30 });
    await expect(machines.decideJoinRequest({ requestId: "r", approve: true, confirmationWords: ["one"], preset: "safe" })).resolves.toBe("ok");
    await machines.forgetMachine("m");
    expect(issue).toHaveBeenCalledWith({ label: "" });
    expect(decide).toHaveBeenCalled();
    expect(forget).toHaveBeenCalledWith({ machineId: "m" });
  });

  it("returns screen text and fails closed when the screen endpoint errors", async () => {
    vi.spyOn(terminalScreen.terminalClient, "getScreen").mockResolvedValue({ plainText: "screen" } as never);
    await expect(terminalScreen.getScreenText("s")).resolves.toBe("screen");
    vi.spyOn(terminalScreen.terminalClient, "getScreen").mockRejectedValue(new Error("offline"));
    await expect(terminalScreen.getScreenText("s")).resolves.toBeNull();
  });

  it("maps device roster actions and labels", async () => {
    vi.spyOn(devices.deviceClient, "list").mockResolvedValue({ devices: [{ deviceId: "d", deviceLabel: "phone", deviceClass: "mobile", connectionCount: 1, firstSeenAt: { seconds: 2n }, sessions: [{ sessionId: "s", sessionName: "main", holdsLease: true }], isSelf: true, reconnecting: false }] } as never);
    vi.spyOn(devices.deviceClient, "disconnect").mockResolvedValue({ closedConnections: 2 } as never);
    vi.spyOn(devices.deviceClient, "giveControl").mockResolvedValue({ transferred: true } as never);
    await expect(devices.listDevices()).resolves.toMatchObject([{ deviceId: "d", firstSeenAt: "1970-01-01T00:00:02.000Z", sessions: [{ holdsLease: true }] }]);
    await expect(devices.disconnectDevice("d")).resolves.toBe(2);
    await expect(devices.giveControl("d")).resolves.toBe(true);
    devices.renameOwnDevice("new label");
  });

  it("maps handoff rules, snippets, and role update presence", async () => {
    const rule = { id: "r", name: "files", enabled: true, source: "unknown", pattern: "*.md", surfaces: ["messages"], sortOrder: 1 };
    vi.spyOn(handoff.handoffRulesClient, "listRules").mockResolvedValue({ rules: [rule] } as never);
    vi.spyOn(handoff.handoffRulesClient, "upsertRule").mockResolvedValue({ rule } as never);
    vi.spyOn(handoff.handoffRulesClient, "deleteRule").mockResolvedValue({} as never);
    await expect(handoff.listHandoffRules()).resolves.toMatchObject([{ source: "file_path", sort_order: 1 }]);
    await expect(handoff.upsertHandoffRule({ name: "files", enabled: true, source: "file_path", pattern: "*.md", surfaces: ["messages"] })).resolves.toMatchObject({ id: "r" });
    await handoff.deleteHandoffRule("r");

    const snippet = { id: "s", name: "n", body: "b", color: "", pinned: false, useCount: 1, lastUsedAt: "", sortOrder: 0, createdAt: "", updatedAt: "" };
    vi.spyOn(snippets.snippetsClient, "listSnippets").mockResolvedValue({ snippets: [snippet] } as never);
    vi.spyOn(snippets.snippetsClient, "upsertSnippet").mockResolvedValue({ snippet } as never);
    vi.spyOn(snippets.snippetsClient, "deleteSnippet").mockResolvedValue({ deleted: true } as never);
    vi.spyOn(snippets.snippetsClient, "touchSnippet").mockResolvedValue({ snippet } as never);
    vi.spyOn(snippets.snippetsClient, "promoteSnippet").mockResolvedValue({ identifier: "skill" } as never);
    await expect(snippets.listSnippets()).resolves.toMatchObject([{ use_count: 1 }]);
    await expect(snippets.upsertSnippet({ name: "n", body: "b" })).resolves.toMatchObject({ id: "s" });
    await expect(snippets.deleteSnippet("s")).resolves.toBe(true);
    await expect(snippets.touchSnippet("s")).resolves.toMatchObject({ id: "s" });
    await expect(snippets.promoteSnippet("s")).resolves.toBe("skill");

    const role = { id: "role", groupId: "g", label: "A", command: "run", workingDir: "/tmp", incomingPrompt: "", backend: "", targetId: "", sessionId: "", sortOrder: 0 };
    vi.spyOn(workspace.workspaceClient, "listRoles").mockResolvedValue({ roles: [role] } as never);
    vi.spyOn(workspace.workspaceClient, "createRole").mockResolvedValue({ role } as never);
    vi.spyOn(workspace.workspaceClient, "updateRole").mockResolvedValue({ role: { ...role, sessionId: "s" } } as never);
    vi.spyOn(workspace.workspaceClient, "deleteRole").mockResolvedValue({} as never);
    await expect(roles.listRoles("g")).resolves.toMatchObject([{ session_id: null }]);
    await expect(roles.createRole({ group_id: "g" })).resolves.toMatchObject({ group_id: "g" });
    await expect(roles.updateRole("role", { label: "B", session_id: null, group_id: "g" })).resolves.toMatchObject({ session_id: "s" });
    await roles.deleteRole("role");
  });
});

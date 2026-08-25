import { beforeEach, describe, expect, it, vi } from "vitest";
import * as ai from "../ai";
import * as capabilities from "../capabilities";
import * as conversation from "../conversation";
import * as filePreview from "../filePreview";
import * as sessions from "../sessions";
import * as settings from "../settings";
import * as targets from "../targets";
import * as workspace from "../workspace";
import { ArchiveRestoreState } from "@vrooli/proto-types/web-console/v1/sessions/sessions_pb";
import { CatalogState } from "@vrooli/proto-types/web-console/v1/targets/targets_pb";
import { EntryType, PreviewKind } from "@vrooli/proto-types/web-console/v1/file_preview/file_preview_pb";

describe("Connect client wrappers", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("maps AI generation, suggestions, config and health", async () => {
    vi.spyOn(ai.aiClient, "generate").mockResolvedValue({ command: "ls", provider: "local" } as never);
    vi.spyOn(ai.aiClient, "suggest").mockResolvedValue({ commands: ["pwd"], provider: "local" } as never);
    vi.spyOn(ai.aiClient, "getConfig").mockResolvedValue({ providers: [{ name: "local", enabled: true, priority: 1, timeoutSec: 5, maxRetries: 2 }], health: [{ name: "local", available: true, lastCheck: "", lastLatency: "", errorCount: 1n, successCount: 2n, errorRate: .1 }] } as never);
    vi.spyOn(ai.aiClient, "updateConfig").mockResolvedValue({ providers: [{ name: "local", enabled: false, priority: 2, timeoutSec: 8, maxRetries: 1 }], health: [] } as never);
    vi.spyOn(ai.aiClient, "getHealth").mockResolvedValue({ health: [{ name: "local", available: false, lastCheck: "now", lastLatency: "1ms", errorCount: 2n, successCount: 0n, errorRate: 1 }] } as never);
    await expect(ai.generateAICommand("list", undefined)).resolves.toEqual({ command: "ls", provider: "local" });
    await expect(ai.generateAISuggestions("where", "ctx")).resolves.toEqual({ commands: ["pwd"], provider: "local" });
    await expect(ai.getAIConfig()).resolves.toMatchObject({ providers: [{ timeout_sec: 5, max_retries: 2 }], health: [{ error_count: 1, last_check: undefined }] });
    await ai.updateAIConfig({ name: "local", enabled: false, priority: 2, timeout_sec: 8, max_retries: 1 });
    await expect(ai.getAIHealth()).resolves.toEqual([{ name: "local", available: false, last_check: "now", last_latency: "1ms", error_count: 2, success_count: 0, error_rate: 1 }]);
  });

  it("maps settings and conversation wrappers, including bigint cursors", async () => {
    vi.spyOn(settings.settingsClient, "getSessionDefaults").mockResolvedValue({ defaults: { defaultBackend: "standard", defaultPolicy: { mode: "preset", duration: "1h" } } } as never);
    vi.spyOn(settings.settingsClient, "updateSessionDefaults").mockResolvedValue({ defaults: undefined } as never);
    await expect(settings.getSessionDefaults()).resolves.toEqual({ default_backend: "standard", default_policy: { mode: "preset", duration: "1h" } });
    await expect(settings.updateSessionDefaults({ default_backend: "tmux", default_policy: { mode: "never" } })).resolves.toEqual({ default_backend: "", default_policy: { mode: "never", duration: "" } });

    const event = { id: "e1", sessionId: "s1", source: "agent", role: "assistant", text: "hi", speechParagraphs: ["hi"], originalSpeechParagraphs: [], summarized: false, createdAt: "now", sequence: 2n, deliveryState: "complete", ttsState: "none", consumptionState: "new" };
    vi.spyOn(conversation.conversationClient, "get").mockResolvedValue({ sessionId: "s1", events: [event], cursor: { lastSeenSequence: 1n, lastListenedSequence: 0n }, hasMore: true, oldestSequence: 1n, newestSequence: 2n, totalCount: 2n } as never);
    vi.spyOn(conversation.conversationClient, "updateCursor").mockResolvedValue({ cursor: { lastSeenSequence: 2n, lastListenedSequence: 1n } } as never);
    vi.spyOn(conversation.conversationClient, "search").mockResolvedValue({ matches: [{ eventId: "e1", sequence: 2n, excerpt: "hi" }], truncated: false, totalMatches: 1n } as never);
    vi.spyOn(conversation.conversationClient, "searchArchived").mockResolvedValue({ matches: [], truncated: false, totalMatches: 0n, distinctSessions: 0n } as never);
    vi.spyOn(conversation.conversationClient, "getRange").mockResolvedValue({ sessionId: "s1", events: [], cursor: undefined } as never);
    vi.spyOn(conversation.conversationClient, "summarizeEvent").mockResolvedValue({ summarized: true, speechParagraphs: ["short"], error: "" } as never);
    await expect(conversation.getConversationSession("s1", { sinceSequence: 1, beforeSequence: 3 })).resolves.toMatchObject({ events: [{ sequence: 2, originalSpeechParagraphs: undefined }], cursor: { lastSeenSequence: 1 } });
    await expect(conversation.updateConversationCursor("s1", { lastSeenSequence: 2 })).resolves.toEqual({ lastSeenSequence: 2, lastListenedSequence: 1 });
    await expect(conversation.searchConversation("s1", "hi")).resolves.toEqual({ matches: [{ eventId: "e1", sequence: 2, excerpt: "hi" }], truncated: false, totalMatches: 1 });
    await expect(conversation.searchArchivedConversations("x")).resolves.toMatchObject({ totalMatches: 0 });
    await expect(conversation.getConversationRange("s1", 1, 2)).resolves.toMatchObject({ sessionId: "s1", cursor: { lastSeenSequence: 0 } });
    await expect(conversation.summarizeEvent("s1", "e1")).resolves.toEqual({ summarized: true, speechParagraphs: ["short"], error: undefined });
  });

  it("maps workspace requests and responses", async () => {
    vi.spyOn(workspace.workspaceClient, "getLayout").mockResolvedValue({ activePane: "s1", panes: [{ sessionId: "s1", name: "main", headerColor: "red", themeId: "dark", fontSize: 14, sortOrder: 1, groupId: "", supportsMessagesView: true, manuallyUnread: false }], groups: [{ id: "g1", name: "G", color: "blue", sortOrder: 0, isCollapsed: false }] } as never);
    vi.spyOn(workspace.workspaceClient, "saveLayout").mockResolvedValue({} as never);
    vi.spyOn(workspace.workspaceClient, "updatePane").mockResolvedValue({ pane: { sessionId: "s1", name: "new", headerColor: "red", themeId: "dark", fontSize: 16, sortOrder: 2, groupId: "g1", supportsMessagesView: true, manuallyUnread: true } } as never);
    vi.spyOn(workspace.workspaceClient, "deletePane").mockResolvedValue({} as never);
    vi.spyOn(workspace.workspaceClient, "createGroup").mockResolvedValue({ group: { id: "g1", name: "G", color: "blue", sortOrder: 0, isCollapsed: false } } as never);
    vi.spyOn(workspace.workspaceClient, "updateGroup").mockResolvedValue({ group: { id: "g1", name: "G2", color: "green", sortOrder: 1, isCollapsed: true } } as never);
    vi.spyOn(workspace.workspaceClient, "deleteGroup").mockResolvedValue({} as never);
    await expect(workspace.getWorkspaceLayout()).resolves.toMatchObject({ active_pane: "s1", panes: [{ group_id: null }], groups: [{ id: "g1" }] });
    await workspace.saveWorkspaceLayout({ active_pane: null, pane_order: ["s1"] });
    await expect(workspace.updateWorkspacePane("s1", { name: "new", group_id: null, manually_unread: true })).resolves.toMatchObject({ name: "new", group_id: "g1" });
    await workspace.deleteWorkspacePane("s1");
    await expect(workspace.createTabGroup({ name: "G", color: "blue" })).resolves.toMatchObject({ id: "g1" });
    await expect(workspace.updateTabGroup("g1", { name: "G2", is_collapsed: true })).resolves.toMatchObject({ name: "G2", is_collapsed: true });
    await workspace.deleteTabGroup("g1");
  });

  it("caches capability liveness and decodes actions", async () => {
    capabilities._resetCapabilitiesCache();
    const response = { capabilities: [{ id: "audio", name: "Audio", description: "", dependencyKind: "scenario", dependencySlug: "audio-tools", features: ["stt"], status: "available", message: "ok", checkedAt: "now", reasonCode: "", actionKind: "", actionLabel: "", operatorCommand: "" }], timestamp: "now", sessionBackends: [{ id: "standard", displayName: "Standard", description: "", survivesRestart: false, available: true, reason: "" }], defaultBackend: "standard" };
    vi.spyOn(capabilities.capabilitiesClient, "get").mockResolvedValue(response as never);
    const live = vi.spyOn(capabilities.capabilitiesClient, "liveness").mockResolvedValue(response as never);
    await expect(capabilities.fetchCapabilities()).resolves.toMatchObject({ capabilities: [{ id: "audio" }] });
    await expect(capabilities.fetchCapabilitiesLivenessCached()).resolves.toMatchObject({ timestamp: "now" });
    expect(live).toHaveBeenCalledTimes(1);
    expect(capabilities.getCapabilitiesLivenessSnapshot()).toBeNull();
    await expect(capabilities.refreshCapabilitiesLiveness()).resolves.toMatchObject({ timestamp: "now" });
    expect(capabilities.getCapabilitiesLivenessSnapshot()).toMatchObject({ timestamp: "now" });
    vi.spyOn(capabilities.capabilitiesClient, "runAction").mockResolvedValue({ success: true, status: "ok", message: "done", capabilityId: "audio", actionKind: "start", capabilities: response.capabilities, timestamp: "now" } as never);
    await expect(capabilities.runCapabilityAction("audio", "start")).resolves.toMatchObject({ success: true, capabilityId: "audio" });
  });

  it("decodes session origins and builds websocket URLs", () => {
    expect(sessions.coerceOriginName("remote")).toBe("remote");
    expect(sessions.coerceOriginName("other")).toBe("unspecified");
    expect(sessions.decodeSession(undefined)).toMatchObject({ backend: "standard", origin: "unspecified", policy: { mode: "never" } });
    expect(sessions.buildSessionWsUrl("s1", { id: "d", label: "phone" })).toContain("deviceId=d");
  });

  it("covers session lifecycle wrappers and recovery projections", async () => {
    const session = { id: "s1", shell: "/bin/sh", createdAt: "now", cols: 80, rows: 24, backend: "persistent", survivesRestart: true, policy: { mode: "preset", duration: "1h" }, recovered: true, owner: "owner", displayLabel: "label" };
    vi.spyOn(sessions.sessionsClient, "create").mockResolvedValue({ session } as never);
    vi.spyOn(sessions.sessionsClient, "list").mockResolvedValue({ sessions: [session], recovery: { inProgress: true, total: 2, recovered: 1, awaitingRecovery: 1, adopted: 0 } } as never);
    vi.spyOn(sessions.sessionsClient, "get").mockResolvedValue({ session } as never);
    vi.spyOn(sessions.sessionsClient, "delete").mockResolvedValue({} as never);
    vi.spyOn(sessions.sessionsClient, "archive").mockResolvedValue({} as never);
    vi.spyOn(sessions.sessionsClient, "unarchive").mockResolvedValue({} as never);
    vi.spyOn(sessions.sessionsClient, "listArchived").mockResolvedValue({ sessions: [{ id: "a", archivedAt: "a", createdAt: "c", agentType: "codex", agentSessionId: "as", cwd: "/tmp", paneName: "", headerColor: "", groupName: "", messageCount: 2n, restoreState: ArchiveRestoreState.REOPENABLE, restoreStateReason: "", awaitingRecovery: true }], total: 1 } as never);
    vi.spyOn(sessions.sessionsClient, "getArchiveRetention").mockResolvedValue({ policy: { messageLessAgeDays: 3, agentHomeAgeDays: 4, maxBytes: 5n }, stats: { entryCount: 1n, messageCount: 2n, transcriptBytes: 3n, agentHomeBytes: 4n, totalBytes: 7n } } as never);
    vi.spyOn(sessions.sessionsClient, "listRecoverable").mockResolvedValue({ sessions: [{ id: "r", backend: "persistent", shell: "/bin/sh", cols: 80, rows: 24, createdAt: "c", orphanedAt: "o", lastActivityAt: "l", agentType: "codex", agentSessionId: "a", launchCommand: "run", cwd: "/tmp", lastRolloutPath: "p", recoverable: true, notRecoverableReason: "", paneName: "p", headerColor: "red", groupName: "g" }] } as never);
    vi.spyOn(sessions.sessionsClient, "recover").mockResolvedValue({ oldSessionId: "r", newSessionId: "s2", agentType: "codex", commandSent: "run", codexHomeCopied: true } as never);
    vi.spyOn(sessions.sessionsClient, "reopen").mockResolvedValue({ oldSessionId: "a", newSessionId: "s3", agentType: "codex", commandSent: "run", codexHomeCopied: false } as never);
    vi.spyOn(sessions.sessionsClient, "dismissRecoverable").mockResolvedValue({} as never);
    vi.spyOn(sessions.sessionsClient, "getPolicy").mockResolvedValue({ policy: { sessionId: "s1", policy: { mode: "preset", duration: "1h" }, expiresAt: "later", ttlSeconds: 10, hasExpiry: true } } as never);
    vi.spyOn(sessions.sessionsClient, "updatePolicy").mockResolvedValue({ policy: { sessionId: "s1", policy: { mode: "never", duration: "" }, expiresAt: "", ttlSeconds: 0, hasExpiry: false } } as never);

    await expect(sessions.createSession({ backend: "persistent", policy: { mode: "preset", duration: "1h" }, idempotency_key: "k" })).resolves.toMatchObject({ id: "s1", recovered: true });
    await expect(sessions.listSessionsWithRecovery()).resolves.toMatchObject({ sessions: [{ id: "s1" }], recovery: { in_progress: true, recovered: 1 } });
    await expect(sessions.listSessions()).resolves.toHaveLength(1);
    await expect(sessions.getSession("s1")).resolves.toMatchObject({ shell: "/bin/sh" });
    await sessions.deleteSession("s1");
    await sessions.archiveSession("s1");
    await sessions.archiveSession("remote:r");
    await sessions.unarchiveSession("s1");
    await sessions.unarchiveSession("remote:r");
    await expect(sessions.listArchivedSessions()).resolves.toMatchObject({ sessions: [{ pane_name: "a", restore_state: "reopenable" }] });
    await expect(sessions.getArchiveRetention()).resolves.toMatchObject({ policy: { max_bytes: 5 }, stats: { total_bytes: 7 } });
    await expect(sessions.listRecoverableSessions()).resolves.toMatchObject([{ id: "r", recoverable: true }]);
    await expect(sessions.recoverSession("r", "k")).resolves.toEqual({ old_session_id: "r", new_session_id: "s2", agent_type: "codex", command_sent: "run", codex_home_copied: true });
    await expect(sessions.reopenSession("a", "k")).resolves.toMatchObject({ new_session_id: "s3" });
    await sessions.dismissRecoverableSession("r");
    await expect(sessions.getSessionPolicy("s1")).resolves.toMatchObject({ expires_at: "later", ttl_seconds: 10 });
    await expect(sessions.updateSessionPolicy("s1", { mode: "never" })).resolves.toMatchObject({ policy: { mode: "never" } });
  });

  it("decodes target catalogs and file preview pages", async () => {
    const target = { id: "t", kind: "ssh", label: "Remote", os: "linux", arch: "amd64", nodeId: "n", revision: "r", status: "online", online: true, lastSeenAt: { seconds: 2n, nanos: 0 }, dispatchable: false, readiness: [{ key: "ssh", label: "SSH", passed: true, detail: "ok" }], failureRung: "credential", state: 99, recoveryAction: "fix", survivesRestart: true };
    vi.spyOn(targets.targetCatalogClient, "list").mockResolvedValue({ state: CatalogState.READY, targets: [target], message: "ok", recoveryAction: "" } as never);
    vi.spyOn(targets.targetCatalogClient, "get").mockResolvedValue({ target } as never);
    await expect(targets.listTargetCatalog()).resolves.toMatchObject({ status: "ready", targets: [{ kind: "ssh", state: "unconfigured", last_seen_at: "1970-01-01T00:00:02.000Z" }] });
    await expect(targets.getTarget("t")).resolves.toMatchObject({ id: "t", available: false });
    expect(targets.decodeTarget({ ...target, kind: "unknown", dispatchable: true, state: 0, lastSeenAt: undefined } as never)).toMatchObject({ kind: "bridge-node", state: "dispatchable" });

    vi.spyOn(filePreview.filePreviewClient, "resolve").mockResolvedValue({ previewId: "p", inputPath: "x", resolvedPath: "/x", basename: "x", hasLine: true, line: 4, resolutionBasis: "path", previewKind: PreviewKind.MARKDOWN, mimeType: "text/markdown", sizeBytes: 3n, canPreview: true, canDownload: true, supportsRange: false, textContentAvailable: true, listingAvailable: false, blobUrl: "/blob", expiresUnixNano: 2_000_000_000n, warnings: [] } as never);
    vi.spyOn(filePreview.filePreviewClient, "listDirectory").mockResolvedValue({ resolvedPath: "/x", parentPath: "/", entries: [{ name: "a", entryType: EntryType.FILE, previewKind: PreviewKind.TEXT, sizeBytes: 2n, mtimeUnixNano: 3_000_000n, canPreview: true, symlinkTarget: "", symlinkBroken: false, mode: "644", childCount: -1n }], totalEntries: 1, truncated: false, nextPageToken: "", effectiveSort: 1, warnings: [] } as never);
    vi.spyOn(filePreview.filePreviewClient, "getTextContent").mockResolvedValue({ resolvedPath: "/x", previewKind: PreviewKind.TEXT, mimeType: "text/plain", content: "hi", truncated: false, hasLine: false, line: 0 } as never);
    await expect(filePreview.resolveFilePreview("s", "x", "cli")).resolves.toMatchObject({ kind: "markdown", line: 4, blobHref: expect.stringContaining("/blob") });
    await expect(filePreview.listDirectory("s", "p", { sort: "name", showHidden: false })).resolves.toMatchObject({ entries: [{ entryType: "file", kind: "text", childCount: null }] });
    await expect(filePreview.getFilePreviewText("s", "p")).resolves.toEqual({ resolvedPath: "/x", kind: "text", mimeType: "text/plain", content: "hi", truncated: false, line: undefined });
  });
});

import { afterEach, describe, expect, it, vi } from "vitest";
import { conversationClient, searchNodeArchive, searchSemanticConversations } from "../conversation";

afterEach(() => vi.restoreAllMocks());

describe("optional archive search wrappers", () => {
  it("decodes local and semantic results with coverage and provenance", async () => {
    const search = vi.spyOn(conversationClient, "searchSemantic").mockResolvedValue({
      local: { matches: [{ eventId: "local-e", sessionId: "local-s", sequence: 4n, role: "user", createdAt: "now", excerpt: "local" }], totalMatches: 1n, distinctSessions: 1n, truncated: false },
      semanticHits: [{ stableHitId: "stable-e", eventId: "am-e", role: "assistant", occurredAt: "then", snippet: "meaning", harness: "codex", sourceSessionId: "am-s", weak: false }],
      coverage: { canonicalVisibleMessages: 10n, catalogDocuments: 9n, lexicalDocuments: 8n, semanticDocuments: 7n, pendingDocuments: 1n, deletedDocuments: 0n, lexicalRatio: .8, semanticRatio: .7, freshnessLagMs: 5n },
      semanticTookMs: 3n, semanticAvailable: true, semanticError: "", degradations: [],
    } as never);

    await expect(searchSemanticConversations("meaning", { agentType: "codex", role: "user" }, 20)).resolves.toMatchObject({
      local: { matches: [{ eventId: "local-e", sequence: 4 }], totalMatches: 1 },
      semanticAvailable: true, semanticTookMs: 3,
      semanticHits: [{ stableHitId: "stable-e", sourceSessionId: "am-s", snippet: "meaning", harness: "codex" }],
      coverage: { canonicalVisibleMessages: 10, semanticDocuments: 7, semanticRatio: .7, freshnessLagMs: 5 },
    });
    expect(search).toHaveBeenCalledWith({ query: "meaning", limit: 20, harnesses: ["codex"] });
  });

  it("keeps a node result scoped to the explicitly chosen node and its native identity", async () => {
    const search = vi.spyOn(conversationClient, "searchNodeArchive").mockResolvedValue({
      nodeId: "node-lab", available: true, degradation: "",
      search: { matches: [{ eventId: "native-e", sessionId: "native-s", sequence: 8n, role: "assistant", createdAt: "now", occurredAt: "2026-08-17T12:10:00Z", excerpt: "local to node" }], truncated: false, totalMatches: 1n, distinctSessions: 1n },
    } as never);
    await expect(searchNodeArchive("node-lab", "local to node", { agentType: "grok" }, 12)).resolves.toMatchObject({
      nodeId: "node-lab", available: true,
      search: { matches: [{ eventId: "native-e", sessionId: "native-s", sequence: 8, occurredAt: "2026-08-17T12:10:00Z" }] },
    });
    expect(search).toHaveBeenCalledWith({ nodeId: "node-lab", query: "local to node", harness: "grok", role: "", limit: 12 });
  });
});

import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const srcRoot = resolve(here, "../..");

/**
 * Decoupling assertion (UI side).
 *
 * The approvals inbox reaches human-approval decisions ONLY through
 * notification-hub (via web-console's own REST facade). It must NOT reach
 * agent-manager at all — not by import, and not by a hardcoded agent-manager
 * discovery token / URL — even though agent-manager is one of the scenarios
 * that raise those decisions. Decisions arrive as the hub's domain-neutral
 * `Ask`, keyed by `source`; the inbox never talks to a producer. The Go-side
 * counterpart is
 * api/approvals_facade_test.go:TestApprovalsInboxDoesNotImportAgentManager.
 */
const inboxFiles = [
  "api/approvals.ts",
  "stores/useApprovalsStore.ts",
  "hooks/useApprovals.ts",
  "components/approvals/ApprovalsDrawer.tsx",
  "components/approvals/ApprovalContextPanel.tsx",
  "components/approvals/approvalsFormat.ts",
];

// An import from any agent-manager surface: the proto-types package, a scenario
// client, or a relative hop into an agent-manager tree.
const FORBIDDEN_IMPORT = /from\s+["'][^"']*agent-manager[^"']*["']|import\s*\(\s*["'][^"']*agent-manager[^"']*["']\s*\)/;

// Any mention of the agent-manager scenario slug at all — a hardcoded discovery
// token or URL would reach the producer directly, which the decoupling forbids
// just as much as an import. (Comments legitimately say "agent-manager" to
// explain the decoupling, so strip line/block comments before scanning.)
function stripComments(source: string): string {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/(^|[^:])\/\/[^\n]*/g, "$1");
}
const FORBIDDEN_SLUG = /agent-manager/;

describe("approvals inbox decoupling", () => {
  for (const rel of inboxFiles) {
    it(`${rel} imports no agent-manager client`, () => {
      const source = readFileSync(resolve(srcRoot, rel), "utf8");
      expect(FORBIDDEN_IMPORT.test(source)).toBe(false);
    });

    it(`${rel} contains no agent-manager discovery token / URL (outside comments)`, () => {
      const code = stripComments(readFileSync(resolve(srcRoot, rel), "utf8"));
      // The ONLY legitimate occurrence is the bare "agent-manager" string
      // literal used to CLASSIFY a decision's source (the ApprovalSource union,
      // the KNOWN_SOURCES list, a switch/case label). That never addresses the
      // scenario. Anything else — a URL containing agent-manager, or the slug
      // passed to a discovery/resolve/fetch call — is a coupling and forbidden.
      const offending = code
        .split("\n")
        .filter((line) => FORBIDDEN_SLUG.test(line))
        // Drop lines whose ONLY agent-manager mention is a standalone quoted
        // label with no URL scheme and no discovery/resolve/fetch call on it.
        .filter((line) => {
          const isBareLabel = /["']agent-manager["']/.test(line);
          const looksLikeUrl = /https?:|\/\/|\.local|:\d|\/api\//.test(line);
          const looksLikeDiscovery = /\b(resolve|discover|fetch|url|host|endpoint|client|scenario)\b/i.test(line);
          return !(isBareLabel && !looksLikeUrl && !looksLikeDiscovery);
        });
      expect(offending).toEqual([]);
    });
  }

  it("the approvals api client calls the web-console REST facade, not a hub Connect client", () => {
    const source = readFileSync(resolve(srcRoot, "api/approvals.ts"), "utf8");
    // Locked architecture: the Go facade owns the Connect call to the hub so it
    // can inject the operator Bearer + Cloudflare Access assertion. The UI is
    // REST-only and must NOT create a Connect client/transport to the hub nor
    // import the notification-hub proto.
    expect(source).toContain("/api/v1/approvals");
    expect(source).not.toContain("@connectrpc/connect");
    expect(source).not.toContain("createClient");
    expect(source).not.toContain("createScenarioConnectTransport");
    expect(source).not.toContain("notification-hub/v1/conversations");
  });
});

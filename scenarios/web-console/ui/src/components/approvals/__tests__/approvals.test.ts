import { describe, expect, it } from "vitest";
import type { ApprovalDecision } from "../../../api/approvals";
import { parseApprovalsDeepLink } from "../../../stores/useApprovalsStore";
import {
  approveOptionKey,
  isActionable,
  recommendedOption,
  relativeTime,
  safeContextUrl,
  secondaryOptions,
  sortDecisions,
  urgencyRank,
} from "../approvalsFormat";

function decision(over: Partial<ApprovalDecision>): ApprovalDecision {
  return {
    id: "a",
    question: "q",
    options: [
      { key: "yes", label: "Yes" },
      { key: "no", label: "No" },
    ],
    recommended: "yes",
    recommendationReason: "",
    defaultAnswer: "",
    reversible: true,
    urgency: "normal",
    contextUrl: "",
    deadline: "",
    state: "pending",
    answer: "",
    answerLabel: "",
    note: "",
    answeredBy: "",
    answeredAt: "",
    late: false,
    defaultAppliedAt: "",
    resolvedAt: "",
    createdAt: new Date().toISOString(),
    updatedAt: "",
    source: "agent-manager",
    sourceEventType: "agent-manager.run-review.decision_requested.v1",
    kind: "run-review",
    ...over,
  };
}

describe("approvalsFormat", () => {
  it("ranks urgency highest-first", () => {
    expect(urgencyRank("critical")).toBeGreaterThan(urgencyRank("high"));
    expect(urgencyRank("high")).toBeGreaterThan(urgencyRank("normal"));
    expect(urgencyRank("")).toBe(0);
  });

  it("sorts by urgency then soonest deadline", () => {
    const now = Date.now();
    const low = decision({ id: "low", urgency: "low" });
    const crit = decision({ id: "crit", urgency: "critical" });
    const highSoon = decision({ id: "soon", urgency: "high", deadline: new Date(now + 60_000).toISOString() });
    const highLate = decision({ id: "late", urgency: "high", deadline: new Date(now + 600_000).toISOString() });
    const order = sortDecisions([low, highLate, crit, highSoon]).map((d) => d.id);
    expect(order).toEqual(["crit", "soon", "late", "low"]);
  });

  it("resolves the recommended option and the rest", () => {
    const d = decision({});
    expect(recommendedOption(d)).toEqual({ key: "yes", label: "Yes" });
    expect(secondaryOptions(d).map((o) => o.key)).toEqual(["no"]);
  });

  it("marks only a pending decision actionable", () => {
    expect(isActionable(decision({ state: "pending" }))).toBe(true);
    expect(isActionable(decision({ state: "expired" }))).toBe(false);
    expect(isActionable(decision({ state: "answered" }))).toBe(false);
  });

  it("formats relative time for past and future", () => {
    const now = Date.now();
    expect(relativeTime(new Date(now + 2 * 3600_000).toISOString(), now)).toBe("in 2h");
    expect(relativeTime(new Date(now - 6 * 60_000).toISOString(), now)).toBe("6m ago");
    expect(relativeTime("", now)).toBe("");
  });

  it("approveOptionKey returns the recommended option only when valid", () => {
    expect(approveOptionKey(decision({ recommended: "yes" }))).toBe("yes");
    // An invalid recommendation must NOT fall back to options[0] for approve.
    expect(approveOptionKey(decision({ recommended: "nonexistent" }))).toBeNull();
    expect(approveOptionKey(decision({ recommended: "" }))).toBeNull();
  });
});

describe("safeContextUrl", () => {
  it("allows http(s) and root-relative URLs", () => {
    expect(safeContextUrl("https://vrooli.com/x")).toBe("https://vrooli.com/x");
    expect(safeContextUrl("http://localhost:8080/y")).toBe("http://localhost:8080/y");
    expect(safeContextUrl("/scenarios/deployment-manager/release/1")).toBe("/scenarios/deployment-manager/release/1");
  });

  it("rejects dangerous or protocol-relative schemes (XSS guard)", () => {
    expect(safeContextUrl("javascript:alert(1)")).toBeNull();
    expect(safeContextUrl("JavaScript:alert(1)")).toBeNull();
    expect(safeContextUrl("data:text/html,<script>alert(1)</script>")).toBeNull();
    expect(safeContextUrl("vbscript:msgbox(1)")).toBeNull();
    expect(safeContextUrl("//evil.com/x")).toBeNull();
    expect(safeContextUrl("")).toBeNull();
    expect(safeContextUrl("   ")).toBeNull();
    expect(safeContextUrl("not a url")).toBeNull();
  });
});

describe("parseApprovalsDeepLink", () => {
  it("opens on an explicit approval id", () => {
    expect(parseApprovalsDeepLink("?approval=ask-1")).toEqual({ open: true, decisionId: "ask-1" });
  });
  it("accepts the agent-manager authorization/run aliases", () => {
    expect(parseApprovalsDeepLink("?authorization=ask-9")).toEqual({ open: true, decisionId: "ask-9" });
    expect(parseApprovalsDeepLink("?run=ask-7")).toEqual({ open: true, decisionId: "ask-7" });
  });
  it("opens with no specific decision on the flag", () => {
    expect(parseApprovalsDeepLink("?approvals")).toEqual({ open: true, decisionId: null });
    expect(parseApprovalsDeepLink("?approvals=1")).toEqual({ open: true, decisionId: null });
  });
  it("does not open without an approvals param", () => {
    expect(parseApprovalsDeepLink("?foo=bar")).toEqual({ open: false, decisionId: null });
    expect(parseApprovalsDeepLink("")).toEqual({ open: false, decisionId: null });
  });
});

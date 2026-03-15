import { describe, it, expect } from "vitest";
import {
  buildScenarioEnvelope,
  verificationHint,
  composePrompt,
  testFailureContextItems,
  codeQualityContextItems,
  ruleViolationContextItems,
  rulesSummaryContextItem,
} from "./agentContext";
import type { ScenarioEnvelopeData, AgentContextItem, AuditorViolation } from "./api";
import type { TestPhaseResult, TidinessIssue } from "./api";

// =============================================================================
// Test data factories
// =============================================================================

function makeEnvelopeData(overrides?: Partial<ScenarioEnvelopeData>): ScenarioEnvelopeData {
  return {
    name: "my-scenario",
    displayName: "My Scenario",
    description: "A test scenario for unit tests",
    path: "scenarios/my-scenario",
    tags: ["web", "api"],
    dependencies: {
      scenarios: { "test-genie": "Automated testing", "tidiness-manager": "Code quality" },
      resources: { postgres: "Database storage" },
    },
    lifecycle: {
      testCommand: "test-genie execute my-scenario --preset comprehensive",
      buildCommand: "cd api && go build -o api .",
    },
    ...overrides,
  };
}

function makeTestPhase(overrides?: Partial<TestPhaseResult>): TestPhaseResult {
  return {
    name: "unit-tests",
    status: "failed",
    durationSeconds: 12,
    classification: "assertion-failure",
    remediation: "Fix the failing assertion",
    error: "Expected 200 got 404\n  at test.ts:42",
    logPath: "/tmp/logs/unit-tests.log",
    ...overrides,
  } as TestPhaseResult;
}

function makeTidinessIssue(overrides?: Partial<TidinessIssue>): TidinessIssue {
  return {
    id: "issue-1",
    title: "Unused import",
    file_path: "src/utils.ts",
    line_number: 5,
    category: "lint",
    severity: "low",
    description: "The import 'foo' is never used.",
    remediation_steps: "Remove the unused import.",
    ...overrides,
  } as TidinessIssue;
}

function makeViolation(overrides?: Partial<AuditorViolation>): AuditorViolation {
  return {
    id: "v-1",
    type: "makefile-lifecycle",
    severity: "high",
    title: "Missing test target",
    file_path: "Makefile",
    line_number: 10,
    description: "Makefile is missing a 'test' target.",
    code_snippet: "build:\n\tgo build .",
    recommendation: "Add a test target to the Makefile.",
    source: "scenario-stack-governor",
    ...overrides,
  } as AuditorViolation;
}

function makeContextItem(overrides?: Partial<AgentContextItem>): AgentContextItem {
  return {
    kind: "test-failure",
    id: "test-unit",
    label: "Test failure: unit",
    markdown: "### Test failure\nSome details",
    ...overrides,
  };
}

// =============================================================================
// buildScenarioEnvelope
// =============================================================================

describe("buildScenarioEnvelope", () => {
  it("renders all sections with full data", () => {
    const md = buildScenarioEnvelope(makeEnvelopeData());

    expect(md).toContain("## Scenario Context");
    expect(md).toContain("**My Scenario**");
    expect(md).toContain("`my-scenario`");
    expect(md).toContain("`scenarios/my-scenario`");
    expect(md).toContain("A test scenario for unit tests");
    expect(md).toContain("web, api");

    // Dependencies
    expect(md).toContain("### Dependencies");
    expect(md).toContain("**test-genie** (scenario): Automated testing");
    expect(md).toContain("**tidiness-manager** (scenario): Code quality");
    expect(md).toContain("**postgres** (resource): Database storage");

    // Lifecycle
    expect(md).toContain("### Verification Commands");
    expect(md).toContain("`test-genie execute my-scenario --preset comprehensive`");
    expect(md).toContain("`cd api && go build -o api .`");

    // Guidance
    expect(md).toContain("### Deeper Guidance");
    expect(md).toContain('`prompt-manager search "my-scenario" -limit 5`');

    // Ends with separator
    expect(md).toMatch(/---$/);
  });

  it("omits tags line when tags are empty", () => {
    const md = buildScenarioEnvelope(makeEnvelopeData({ tags: [] }));
    expect(md).not.toContain("**Tags:**");
  });

  it("omits Dependencies section when no dependencies exist", () => {
    const md = buildScenarioEnvelope(
      makeEnvelopeData({ dependencies: { scenarios: {}, resources: {} } }),
    );
    expect(md).not.toContain("### Dependencies");
  });

  it("falls back to vrooli scenario test when testCommand is missing", () => {
    const md = buildScenarioEnvelope(
      makeEnvelopeData({ lifecycle: { testCommand: undefined } }),
    );
    expect(md).toContain("`vrooli scenario test my-scenario`");
  });

  it("omits build line when buildCommand is missing", () => {
    const md = buildScenarioEnvelope(
      makeEnvelopeData({ lifecycle: { testCommand: "make test", buildCommand: undefined } }),
    );
    expect(md).not.toContain("**Build:**");
  });
});

// =============================================================================
// verificationHint
// =============================================================================

describe("verificationHint", () => {
  it("returns test-genie command for test-failure", () => {
    const hint = verificationHint("test-failure", "my-app");
    expect(hint).toContain("`vrooli scenario test my-app`");
    expect(hint).toContain("test-genie");
    expect(hint).toContain("`prompt-manager skill read test`");
  });

  it("returns tidiness-manager command for code-quality-issue", () => {
    const hint = verificationHint("code-quality-issue", "my-app");
    expect(hint).toContain("`tidiness-manager scan my-app`");
    expect(hint).toContain("tidiness-manager");
    expect(hint).toContain("`prompt-manager skill read refactor`");
  });

  it("returns scenario-auditor command for rule-violation", () => {
    const hint = verificationHint("rule-violation", "my-app");
    expect(hint).toContain("`scenario-auditor scan my-app`");
    expect(hint).toContain("scenario-auditor");
  });

  it("returns scenario-auditor command for rules-summary", () => {
    const hint = verificationHint("rules-summary", "my-app");
    expect(hint).toContain("`scenario-auditor scan my-app`");
  });
});

// =============================================================================
// composePrompt (with envelope)
// =============================================================================

describe("composePrompt", () => {
  it("prepends envelope before user message", () => {
    const envelope = "## Scenario Context\nSome envelope text\n---";
    const prompt = composePrompt("fix the bug", [], envelope);

    const envelopeIdx = prompt.indexOf("## Scenario Context");
    const messageIdx = prompt.indexOf("fix the bug");
    expect(envelopeIdx).toBeGreaterThanOrEqual(0);
    expect(messageIdx).toBeGreaterThan(envelopeIdx);
  });

  it("omits envelope when undefined", () => {
    const prompt = composePrompt("fix the bug", []);
    expect(prompt).toBe("fix the bug");
  });

  it("places screenshot paths before envelope", () => {
    const items: AgentContextItem[] = [
      makeContextItem({ kind: "screenshot", screenshotPaths: ["/tmp/shot.png"] }),
    ];
    const envelope = "## Scenario Context\n---";
    const prompt = composePrompt("check this", items, envelope);

    const pathIdx = prompt.indexOf("/tmp/shot.png");
    const envelopeIdx = prompt.indexOf("## Scenario Context");
    expect(pathIdx).toBeGreaterThanOrEqual(0);
    expect(pathIdx).toBeLessThan(envelopeIdx);
  });

  it("combines envelope + message + context items in correct order", () => {
    const items: AgentContextItem[] = [
      makeContextItem({ markdown: "### Issue details" }),
    ];
    const envelope = "## Scenario Context\n---";
    const prompt = composePrompt("fix it", items, envelope);

    const envelopeIdx = prompt.indexOf("## Scenario Context");
    const messageIdx = prompt.indexOf("fix it");
    const contextIdx = prompt.indexOf("## Attached Context");
    const issueIdx = prompt.indexOf("### Issue details");

    expect(envelopeIdx).toBeLessThan(messageIdx);
    expect(messageIdx).toBeLessThan(contextIdx);
    expect(contextIdx).toBeLessThan(issueIdx);
  });

  it("truncates when total exceeds 50K chars", () => {
    const longMessage = "x".repeat(60_000);
    const prompt = composePrompt(longMessage, []);
    expect(prompt).toContain("... (context truncated to fit limit)");
    // The truncation message itself adds chars, so just check it's bounded.
    expect(prompt.length).toBeLessThan(60_000);
  });

  it("handles envelope-only prompt (no message, no items)", () => {
    const prompt = composePrompt("", [], "## Envelope\n---");
    expect(prompt).toBe("## Envelope\n---");
  });
});

// =============================================================================
// Context builders with verification hints
// =============================================================================

describe("testFailureContextItems", () => {
  it("includes verification hint in markdown", () => {
    const items = testFailureContextItems([makeTestPhase()], "my-scenario");
    expect(items).toHaveLength(1);
    expect(items[0]!.markdown).toContain("`vrooli scenario test my-scenario`");
    expect(items[0]!.markdown).toContain("test-genie");
  });

  it("filters out non-failed phases", () => {
    const items = testFailureContextItems(
      [makeTestPhase({ status: "passed" as TestPhaseResult["status"] }), makeTestPhase({ name: "integration", status: "failed" as TestPhaseResult["status"] })],
      "s",
    );
    expect(items).toHaveLength(1);
    expect(items[0]!.id).toBe("test-integration");
  });

  it("includes phase metadata in markdown", () => {
    const items = testFailureContextItems([makeTestPhase()], "s");
    const md = items[0]!.markdown;
    expect(md).toContain("### Test Phase: unit-tests");
    expect(md).toContain("**Status:** failed");
    expect(md).toContain("**Duration:** 12s");
    expect(md).toContain("**Classification:** assertion-failure");
    expect(md).toContain("Expected 200 got 404");
  });
});

describe("codeQualityContextItems", () => {
  it("includes verification hint in markdown", () => {
    const items = codeQualityContextItems([makeTidinessIssue()], "my-scenario");
    expect(items).toHaveLength(1);
    expect(items[0]!.markdown).toContain("`tidiness-manager scan my-scenario`");
  });

  it("includes issue details in markdown", () => {
    const items = codeQualityContextItems([makeTidinessIssue()], "s");
    const md = items[0]!.markdown;
    expect(md).toContain("### Unused import");
    expect(md).toContain("`src/utils.ts`:5");
    expect(md).toContain("**Category:** lint");
    expect(md).toContain("**Remediation:** Remove the unused import.");
  });
});

describe("ruleViolationContextItems", () => {
  it("includes verification hint in markdown", () => {
    const items = ruleViolationContextItems([makeViolation()], "my-scenario");
    expect(items).toHaveLength(1);
    expect(items[0]!.markdown).toContain("`scenario-auditor scan my-scenario`");
  });

  it("includes violation details in markdown", () => {
    const items = ruleViolationContextItems([makeViolation()], "s");
    const md = items[0]!.markdown;
    expect(md).toContain("### Rule Violation: makefile-lifecycle");
    expect(md).toContain("**Severity:** high");
    expect(md).toContain("**Recommendation:** Add a test target");
  });
});

describe("rulesSummaryContextItem", () => {
  it("includes verification hint in markdown", () => {
    const item = rulesSummaryContextItem(
      [makeViolation()],
      undefined,
      "my-scenario",
    );
    expect(item.markdown).toContain("`scenario-auditor scan my-scenario`");
  });

  it("computes severity breakdown from violations when no summary provided", () => {
    const violations: AuditorViolation[] = [
      makeViolation({ severity: "high" }),
      makeViolation({ severity: "medium", id: "v-2" }),
      makeViolation({ severity: "low", id: "v-3" }),
    ];
    const item = rulesSummaryContextItem(violations, undefined, "s");
    expect(item.markdown).toContain("**Total violations:** 3");
    expect(item.markdown).toContain("**High:** 1");
    expect(item.markdown).toContain("**Medium:** 1");
    expect(item.markdown).toContain("**Low:** 1");
  });

  it("uses summary data when provided", () => {
    const item = rulesSummaryContextItem(
      [],
      {
        total: 5,
        by_severity: { high: 2, medium: 2, low: 1 },
        highest_severity: "high",
        recommended_steps: ["Fix the Makefile", "Add health checks"],
        generated_at: "2026-03-15T00:00:00Z",
      },
      "s",
    );
    expect(item.markdown).toContain("**Total violations:** 5");
    expect(item.markdown).toContain("Fix the Makefile");
    expect(item.markdown).toContain("Add health checks");
  });
});

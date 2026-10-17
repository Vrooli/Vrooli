import { describe, expect, it } from "vitest";
import type { ArtifactRef, PhaseInfo, RunInfo, RunPhaseDescriptor } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import type { DiffResult } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import { baselineComparisonContextItem, buildScenarioEnvelope, changeSummaryContextItem, composePrompt, runArtifactContextItem, runPhaseContextItem, verificationHint } from "./agentContext";
import type { AgentContextItem, ScenarioEnvelopeData } from "./api";

describe("canonical Agent context", () => {
  it("builds a scenario envelope", () => {
    const data = {
      name: "demo",
      displayName: "Demo",
      description: "A demo scenario",
      path: "scenarios/demo",
      tags: [],
      dependencies: { scenarios: {}, resources: {} },
      lifecycle: { testCommand: "vrooli scenario test demo" },
    } as ScenarioEnvelopeData;
    expect(buildScenarioEnvelope(data)).toContain("autonomous code improvement agent");
    expect(buildScenarioEnvelope(data)).toContain("scenarios/demo");
  });

  it("preserves run, phase, provider, and findings identity", () => {
    const run = { runId: "run-1" } as RunInfo;
    const phase = { name: "future-health", status: "failed", findingsSummary: { blockers: 1, errors: 2, warnings: 3, infos: 0, total: 6 } } as PhaseInfo;
    const descriptor = { phase: "future-health", displayName: "Future Health", provider: "future-provider" } as RunPhaseDescriptor;
    const item = runPhaseContextItem(run, phase, descriptor, "demo");
    expect(item.id).toBe("test-phase:run-1:future-health");
    expect(item.markdown).toContain("future-provider");
    expect(item.markdown).toContain("6 total");
    expect(item.markdown).toContain("vrooli scenario test demo");
  });

  it("keeps artifact and comparison context opaque, bounded, and identity-rich", () => {
    const artifact = runArtifactContextItem(
      { runId: "run-1" } as RunInfo,
      { id: "opaque-1", kind: "future.report", producingPhase: "future-health", label: "Future report" } as ArtifactRef,
    );
    expect(artifact.id).toBe("artifact:run-1:opaque-1");
    expect(artifact.markdown).toContain("future.report");
    expect(artifact.markdown).not.toContain("/tmp/");

    const comparison = baselineComparisonContextItem({
      verdict: "regression",
      baseline: { name: "main", run: { runId: "base-1" } },
      currentGit: { sha: "abc123" },
      evidence: { currentRunId: "current-1" },
      phases: [{ phase: "future-health", verdict: "not-comparable", reasons: [{ detail: "Provider unavailable" }] }],
    } as DiffResult);
    expect(comparison.id).toBe("baseline-comparison:base-1:current-1");
    expect(comparison.markdown).toContain("Provider unavailable");
  });

  it("builds change summary context", () => {
    const item = changeSummaryContextItem({ staged: { "a.ts": { additions: 3, deletions: 1, files: 1 } } });
    expect(item.label).toContain("1 files");
    expect(item.markdown).toContain("Additions:** +3");
  });

  it("generates an actionable prompt for phase findings", () => {
    const items: AgentContextItem[] = [{ kind: "test-failure", id: "phase", label: "Future Health", markdown: "## Phase" }];
    const prompt = composePrompt("", items);
    expect(prompt).toContain("Fix all of the test failures");
    expect(prompt).toContain("## Acceptance Criteria");
  });

  it("keeps screenshots informational", () => {
    const items: AgentContextItem[] = [{ kind: "screenshot", id: "artifact", label: "Image", markdown: "## Screenshot" }];
    expect(composePrompt("", items)).not.toContain("Fix all of the");
    expect(verificationHint("test-failure", "demo")).toContain("vrooli scenario test demo");
  });
});

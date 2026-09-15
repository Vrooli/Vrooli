import type { AgentContextItem, ScenarioEnvelopeData } from "./api";
import type { SnapshotSetMeta, SnapshotFile, RepoFileStats } from "./api";
import { fetchScreenshotPath } from "./api";
import { aggregateFileStats, formatNetLines } from "./metrics";
import type { ArtifactRef, PhaseInfo, RunInfo, RunPhaseDescriptor } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import type { DiffResult } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

const _MAX_PROMPT_CHARS = 50_000;

// =============================================================================
// Verification hints
// =============================================================================

/** Context kinds that have a meaningful re-verification command. */
export type VerifiableContextKind = "test-failure";

/**
 * Returns a markdown blockquote with the command that detected this issue
 * and optionally a skill reference for deeper guidance.
 *
 * Appended to actionable context items so the agent knows how to verify its fix.
 */
export function verificationHint(kind: VerifiableContextKind, scenarioName: string): string {
  switch (kind) {
    case "test-failure":
      return (
        `\n> **Verify fix:** \`vrooli scenario test ${scenarioName}\` (detected by test-genie)\n` +
        `> **For guidance:** \`prompt-manager skill read test\`\n`
      );
  }
}

/** Build stable Agent context from the canonical run snapshot, not a tab-only projection. */
export function runPhaseContextItem(
  run: RunInfo,
  phase: PhaseInfo,
  descriptor: RunPhaseDescriptor | undefined,
  scenarioName: string,
): AgentContextItem {
  const findings = phase.findingsSummary;
  const markdown = [
    `## Test phase: ${descriptor?.displayName || phase.name}`,
    "",
    `- Run: \`${run.runId}\``,
    `- Phase: \`${phase.name}\``,
    `- Status: **${phase.status}**`,
    descriptor?.provider ? `- Provider: \`${descriptor.provider}\`` : "",
    findings ? `- Findings: ${findings.total} total (${findings.blockers} blockers, ${findings.errors} errors, ${findings.warnings} warnings)` : "",
    verificationHint("test-failure", scenarioName),
  ].filter(Boolean).join("\n");
  return {
    kind: "test-failure",
    id: `test-phase:${run.runId}:${phase.name}`,
    label: `${descriptor?.displayName || phase.name} (${phase.status})`,
    markdown,
  };
}

/** Build path-free Agent context for one opaque artifact reference. */
export function runArtifactContextItem(run: RunInfo, artifact: ArtifactRef): AgentContextItem {
  const relationships = (artifact.relationships ?? []).slice(0, 20).map((relationship) =>
    `- Relationship: ${relationship.type || "related"} → \`${relationship.targetArtifactId}\``,
  );
  return {
    kind: artifact.kind === "screenshot" ? "screenshot" : "test-failure",
    id: `artifact:${run.runId}:${artifact.id}`,
    label: artifact.label || artifact.kind,
    markdown: [
      "## Test evidence",
      "",
      `- Run: \`${run.runId}\``,
      `- Artifact: \`${artifact.id}\``,
      `- Kind: \`${artifact.kind}\``,
      `- Producing phase: \`${artifact.producingPhase || "unknown"}\``,
      `- Media type: \`${artifact.mediaType || "unknown"}\``,
      `- Provenance: \`${artifact.provenance}\``,
      ...relationships,
    ].join("\n"),
  };
}

/** Build bounded, identity-rich Agent context for a baseline comparison. */
export function baselineComparisonContextItem(diff: DiffResult): AgentContextItem {
  const baseRunId = diff.baseline?.run?.runId || "unavailable";
  const currentRunId = diff.evidence?.currentRunId || "unavailable";
  const attention = diff.phases.filter((phase) => phase.verdict !== "clean");
  return {
    kind: "test-failure",
    id: `baseline-comparison:${baseRunId}:${currentRunId}`,
    label: `Baseline comparison (${diff.verdict})`,
    markdown: [
      "## Baseline comparison",
      "",
      `- Baseline: \`${diff.baseline?.name || "unknown"}\``,
      `- Base run: \`${baseRunId}\``,
      `- Current run: \`${currentRunId}\``,
      `- Verdict: **${diff.verdict}**`,
      `- Current SHA: \`${diff.currentGit?.sha || "unavailable"}\``,
      `- Needs attention: ${attention.length} phase(s)`,
      "",
      ...attention.slice(0, 20).map((phase) => `- \`${phase.phase}\`: ${phase.verdict}${phase.reasons[0]?.detail ? ` — ${phase.reasons[0].detail}` : ""}`),
      attention.length > 20 ? `- …and ${attention.length - 20} more phase(s)` : "",
    ].filter(Boolean).join("\n"),
  };
}

// =============================================================================
// Scenario envelope
// =============================================================================

/**
 * Build a markdown "scenario envelope" that orients an AI agent within a scenario.
 *
 * This is silently prepended to the first message of a conversation — it is not
 * a user-visible context chip. The output is agent-agnostic plain markdown.
 *
 * The envelope is framed as a **directive** (not a description) so the agent
 * treats incoming context as actionable work rather than information to discuss.
 *
 * @param data - Enriched scenario metadata from the /scenarios/{slug}/envelope endpoint.
 * @returns Formatted markdown string ending with a horizontal rule separator.
 */
export function buildScenarioEnvelope(data: ScenarioEnvelopeData): string {
  const lines: string[] = [];

  lines.push("## Your Role");
  lines.push("");
  lines.push(
    `You are an autonomous code improvement agent for the **${data.displayName}** scenario (\`${data.name}\`). ` +
    "When issues are attached below, **immediately begin fixing them** — read the relevant source files, " +
    "implement the necessary changes, and verify your fixes with the provided commands. " +
    "Do not ask clarifying questions unless a critical ambiguity would lead to wasted work; " +
    "instead state your assumption and proceed.",
  );
  lines.push("");
  lines.push(`- **Path:** \`${data.path}\``);
  lines.push(`- **Description:** ${data.description}`);

  if (data.tags.length > 0) {
    lines.push(`- **Tags:** ${data.tags.join(", ")}`);
  }

  // Dependencies — only render if at least one exists.
  const scenarioDeps = Object.entries(data.dependencies.scenarios);
  const resourceDeps = Object.entries(data.dependencies.resources);
  if (scenarioDeps.length > 0 || resourceDeps.length > 0) {
    lines.push("");
    lines.push("### Dependencies");
    for (const [name, desc] of scenarioDeps) {
      lines.push(`- **${name}** (scenario): ${desc}`);
    }
    for (const [name, desc] of resourceDeps) {
      lines.push(`- **${name}** (resource): ${desc}`);
    }
  }

  // Verification commands.
  lines.push("");
  lines.push("### Verification Commands");
  lines.push(`- **Run tests:** \`${data.lifecycle.testCommand || `vrooli scenario test ${data.name}`}\``);
  if (data.lifecycle.buildCommand) {
    lines.push(`- **Build:** \`${data.lifecycle.buildCommand}\``);
  }

  // Skill discovery hint.
  lines.push("");
  lines.push("### Deeper Guidance");
  lines.push(`For detailed guidance on working with this scenario, run: \`prompt-manager search "${data.name}" -limit 5\``);

  lines.push("");
  lines.push("---");

  return lines.join("\n");
}

// =============================================================================
// Context item builders
// =============================================================================

/** Build a context item for a screenshot. */
export function screenshotContextItem(snapshot: SnapshotSetMeta, file: SnapshotFile): AgentContextItem {
  const pageName = file.pageLabel || file.pagePath || file.filename;
  return {
    kind: "screenshot" as const,
    id: `screenshot-${snapshot.id}-${file.filename}`,
    label: `Screenshot: ${pageName}`,
    markdown: `### Screenshot: ${pageName}\n\n- **Captured:** ${new Date(snapshot.createdAt).toLocaleString()}\n- **Viewport:** ${file.viewportWidth ?? "?"}x${file.viewportHeight ?? "?"}\n- **Theme:** ${file.theme || "default"}\n\n_Image file attached below._\n`,
  };
}

/** Resolve filesystem paths for screenshot context items so the agent can read them. */
export async function resolveScreenshotPaths(
  items: AgentContextItem[],
  scenarioSlug: string,
  repoId?: string,
): Promise<AgentContextItem[]> {
  const results = await Promise.all(
    items.map(async (item) => {
      if (item.kind !== "screenshot") return item;
      // Parse captureId and filename from id: "screenshot-{captureId}-{filename}"
      const match = item.id.match(/^screenshot-(.+?)-([^-]+\.\w+)$/);
      if (!match?.[1] || !match[2]) return item;
      const captureId = match[1];
      const filename = match[2];
      try {
        const path = await fetchScreenshotPath(captureId, scenarioSlug, filename, repoId);
        return { ...item, screenshotPaths: [path] };
      } catch {
        // Graceful degradation — screenshot may have been deleted
        return item;
      }
    }),
  );
  return results;
}

/** Build a context item from file change statistics. */
export function changeSummaryContextItem(fileStats: RepoFileStats): AgentContextItem {
  const agg = aggregateFileStats(fileStats);
  if (!agg || agg.totalFiles === 0) {
    return {
      kind: "change-summary" as const,
      id: "change-summary",
      label: "Change Summary",
      markdown: "### Change Summary\n\nNo file changes detected.\n",
    };
  }

  let md = `### Change Summary\n\n`;
  md += `- **Files changed:** ${agg.totalFiles}\n`;
  md += `- **Additions:** +${agg.totalAdditions}\n`;
  md += `- **Deletions:** -${agg.totalDeletions}\n`;
  md += `- **Net:** ${formatNetLines(agg.totalNetLines)}\n`;

  return {
    kind: "change-summary" as const,
    id: "change-summary",
    label: `Change Summary (${agg.totalFiles} files)`,
    markdown: md,
  };
}

/** Build a context item from scenario-wide quality data. */
export function scenarioQualityContextItem(scoreData: {
  score: number;
  violations: number;
  breakdown?: {
    lint_issues: number;
    type_issues: number;
    long_files: number;
    complex_functions: number;
    tech_debt_markers: number;
    duplication_issues: number;
  };
}): AgentContextItem {
  let md = `### Improve: Code quality score is ${Math.round(scoreData.score)}/100\n\n`;
  md += `- **Violations:** ${scoreData.violations}\n`;
  if (scoreData.breakdown) {
    md += `\n**Breakdown:**\n`;
    const entries: [string, number][] = [
      ["Lint issues", scoreData.breakdown.lint_issues],
      ["Type issues", scoreData.breakdown.type_issues],
      ["Long files", scoreData.breakdown.long_files],
      ["Complex functions", scoreData.breakdown.complex_functions],
      ["Tech debt markers", scoreData.breakdown.tech_debt_markers],
      ["Duplication", scoreData.breakdown.duplication_issues],
    ];
    for (const [label, value] of entries) {
      if (value > 0) md += `- ${label}: ${value}\n`;
    }
  }

  return {
    kind: "scenario-quality" as const,
    id: "scenario-quality",
    label: `Code Quality: ${Math.round(scoreData.score)}/100 (${scoreData.violations} violations)`,
    markdown: md,
  };
}

// =============================================================================
// Prompt composition (re-exported from agentPrompt.ts)
// =============================================================================

export { composePrompt, buildTaskDirective, buildAcceptanceCriteria } from "./agentPrompt";

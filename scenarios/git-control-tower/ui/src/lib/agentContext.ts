import type { AgentContextItem, AuditorViolation, AuditorViolationSummary, ScenarioEnvelopeData } from "./api";
import type { TestPhaseResult, TidinessIssue, SnapshotSetMeta, SnapshotFile, RepoFileStats } from "./api";
import { fetchScreenshotPath } from "./api";
import { aggregateFileStats, formatNetLines } from "./metrics";

const _MAX_PROMPT_CHARS = 50_000;
const MAX_ERROR_LINES = 200;

// =============================================================================
// Verification hints
// =============================================================================

/** Context kinds that have a meaningful re-verification command. */
export type VerifiableContextKind = "test-failure" | "code-quality-issue" | "rule-violation" | "rules-summary";

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
    case "code-quality-issue":
      return (
        `\n> **Verify fix:** \`tidiness-manager scan ${scenarioName}\` (detected by tidiness-manager)\n` +
        `> **For guidance:** \`prompt-manager skill read refactor\`\n`
      );
    case "rule-violation":
    case "rules-summary":
      return `\n> **Verify fix:** \`scenario-auditor scan ${scenarioName}\` (detected by scenario-auditor)\n`;
  }
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

/**
 * Build context items from failed test phases.
 *
 * @param phases - Test phase results (only failed phases produce items).
 * @param scenarioName - Scenario slug, used to generate the verification hint.
 */
export function testFailureContextItems(phases: TestPhaseResult[], scenarioName: string): AgentContextItem[] {
  return phases
    .filter((p) => p.status === "failed")
    .map((phase) => {
      let md = `### Fix: Test failure in "${phase.name}"\n`;
      md += `- **Duration:** ${phase.durationSeconds}s\n`;
      if (phase.classification) md += `- **Classification:** ${phase.classification}\n`;
      if (phase.remediation) md += `- **Remediation:** ${phase.remediation}\n`;
      if (phase.error) {
        const lines = phase.error.split("\n");
        const truncated = lines.length > MAX_ERROR_LINES
          ? [...lines.slice(0, MAX_ERROR_LINES), `\n... (${lines.length - MAX_ERROR_LINES} more lines truncated)`]
          : lines;
        md += `\n\`\`\`\n${truncated.join("\n")}\n\`\`\`\n`;
      }
      if (phase.logPath) md += `\n_Log file:_ \`${phase.logPath}\`\n`;

      md += verificationHint("test-failure", scenarioName);

      return {
        kind: "test-failure" as const,
        id: `test-${phase.name}`,
        label: `Test failure: ${phase.name}`,
        markdown: md,
      };
    });
}

/**
 * Build context items from code quality issues.
 *
 * @param issues - Tidiness issues from the tidiness-manager.
 * @param scenarioName - Scenario slug, used to generate the verification hint.
 */
export function codeQualityContextItems(issues: TidinessIssue[], scenarioName: string): AgentContextItem[] {
  return issues.map((issue) => {
    let md = `### Fix: ${issue.title}\n`;
    md += `- **File:** \`${issue.file_path}\``;
    if (issue.line_number) md += `:${issue.line_number}`;
    md += "\n";
    md += `- **Category:** ${issue.category}\n`;
    md += `- **Severity:** ${issue.severity}\n`;
    if (issue.description) md += `\n${issue.description}\n`;
    if (issue.remediation_steps) md += `\n**Remediation:** ${issue.remediation_steps}\n`;

    md += verificationHint("code-quality-issue", scenarioName);

    return {
      kind: "code-quality-issue" as const,
      id: `quality-${issue.id}`,
      label: `${issue.category}: ${issue.file_path}${issue.line_number ? `:${issue.line_number}` : ""}`,
      markdown: md,
    };
  });
}

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

/**
 * Build context items from auditor rule violations.
 *
 * @param violations - Individual violations from the scenario-auditor.
 * @param scenarioName - Scenario slug, used to generate the verification hint.
 */
export function ruleViolationContextItems(violations: AuditorViolation[], scenarioName: string): AgentContextItem[] {
  return violations.map((v, i) => {
    let md = `### Fix: ${v.title}\n`;
    md += `- **Rule:** ${v.type}\n`;
    md += `- **Severity:** ${v.severity}\n`;
    if (v.file_path) {
      md += `- **File:** \`${v.file_path}\``;
      if (v.line_number) md += `:${v.line_number}`;
      md += "\n";
    }
    if (v.description) md += `\n${v.description}\n`;
    if (v.code_snippet) md += `\n\`\`\`\n${v.code_snippet}\n\`\`\`\n`;
    if (v.recommendation) md += `\n**Recommendation:** ${v.recommendation}\n`;
    if (v.source) md += `\n_Source: ${v.source}_\n`;

    md += verificationHint("rule-violation", scenarioName);

    return {
      kind: "rule-violation" as const,
      id: `rule-${v.id || `${v.type}-${i}`}`,
      label: `${v.severity}: ${v.type} — ${v.title.slice(0, 60)}`,
      markdown: md,
    };
  });
}

/**
 * Build a summary context item from auditor violations.
 *
 * @param violations - All violations (used for counting if summary is absent).
 * @param summary - Pre-computed summary with severity breakdown and recommended steps.
 * @param scenarioName - Scenario slug, used to generate the verification hint.
 */
export function rulesSummaryContextItem(
  violations: AuditorViolation[],
  summary: AuditorViolationSummary | undefined,
  scenarioName: string,
): AgentContextItem {
  const total = summary?.total ?? violations.length;
  const bySev = summary?.by_severity ?? {};
  const high = bySev["high"] ?? violations.filter(v => v.severity === "high").length;
  const medium = bySev["medium"] ?? violations.filter(v => v.severity === "medium").length;
  const low = bySev["low"] ?? violations.filter(v => v.severity === "low").length;

  let md = `### Fix: ${total} standards compliance violations\n\n`;
  md += `- **Total:** ${total}\n`;
  md += `- **High:** ${high}\n`;
  md += `- **Medium:** ${medium}\n`;
  md += `- **Low:** ${low}\n`;
  if (summary?.recommended_steps?.length) {
    md += `\n**Recommended steps:**\n`;
    for (const step of summary.recommended_steps) {
      md += `- ${step}\n`;
    }
  }

  md += verificationHint("rules-summary", scenarioName);

  return {
    kind: "rules-summary" as const,
    id: "rules-summary",
    label: `Rules: ${total} violations (${high}H/${medium}M/${low}L)`,
    markdown: md,
  };
}

// =============================================================================
// Prompt composition (re-exported from agentPrompt.ts)
// =============================================================================

export { composePrompt, buildTaskDirective, buildAcceptanceCriteria } from "./agentPrompt";

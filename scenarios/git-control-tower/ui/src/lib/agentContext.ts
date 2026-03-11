import type { AgentContextItem } from "./api";
import type { TestPhaseResult, TidinessIssue, SnapshotSetMeta, RepoFileStats } from "./api";
import { aggregateFileStats, formatNetLines } from "./metrics";

const MAX_PROMPT_CHARS = 50_000;
const MAX_ERROR_LINES = 200;

/** Build context items from failed test phases. */
export function testFailureContextItems(phases: TestPhaseResult[]): AgentContextItem[] {
  return phases
    .filter((p) => p.status === "failed")
    .map((phase) => {
      let md = `### Test Phase: ${phase.name}\n`;
      md += `- **Status:** ${phase.status}\n`;
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

      return {
        kind: "test-failure" as const,
        id: `test-${phase.name}`,
        label: `Test failure: ${phase.name}`,
        markdown: md,
      };
    });
}

/** Build context items from code quality issues. */
export function codeQualityContextItems(issues: TidinessIssue[]): AgentContextItem[] {
  return issues.map((issue) => {
    let md = `### ${issue.title}\n`;
    md += `- **File:** \`${issue.file_path}\``;
    if (issue.line_number) md += `:${issue.line_number}`;
    md += "\n";
    md += `- **Category:** ${issue.category}\n`;
    md += `- **Severity:** ${issue.severity}\n`;
    if (issue.description) md += `\n${issue.description}\n`;
    if (issue.remediation_steps) md += `\n**Remediation:** ${issue.remediation_steps}\n`;

    return {
      kind: "code-quality-issue" as const,
      id: `quality-${issue.id}`,
      label: `${issue.category}: ${issue.file_path}${issue.line_number ? `:${issue.line_number}` : ""}`,
      markdown: md,
    };
  });
}

/** Build a context item for a screenshot. */
export function screenshotContextItem(snapshot: SnapshotSetMeta, pageName: string): AgentContextItem {
  return {
    kind: "screenshot" as const,
    id: `screenshot-${snapshot.id}-${pageName}`,
    label: `Screenshot: ${pageName}`,
    markdown: `### Visual Reference: ${pageName}\n\nScreenshot captured at ${new Date(snapshot.createdAt).toLocaleString()} (${snapshot.screenshotCount} total screenshots).\n\n_Note: Agents cannot view images directly. This context indicates visual state was captured for the page._\n`,
  };
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
  let md = `### Scenario Code Quality\n\n`;
  md += `- **Score:** ${Math.round(scoreData.score)}/100\n`;
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

/** Compose a full prompt from user message and attached context items. */
export function composePrompt(message: string, contextItems: AgentContextItem[]): string {
  let prompt = message.trim();
  if (contextItems.length > 0) {
    const contextSection = contextItems.map((item) => item.markdown).join("\n---\n\n");
    if (prompt) {
      prompt += "\n\n---\n\n## Attached Context\n\n" + contextSection;
    } else {
      prompt = "## Attached Context\n\n" + contextSection;
    }
  }

  if (prompt.length > MAX_PROMPT_CHARS) {
    prompt = prompt.slice(0, MAX_PROMPT_CHARS) + "\n\n... (context truncated to fit limit)";
  }

  return prompt;
}

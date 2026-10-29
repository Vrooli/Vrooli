// How a run is named and attributed wherever it appears (lists, dashboard,
// detail header). Titles are for people; machine identifiers stay available as
// a short secondary reference rather than as the title itself.
import type { Run } from "../types";
import { runnerTypeFromSlug, runnerTypeLabel } from "./utils";

export interface RunHeading {
  title: string;
  /** Short machine reference shown secondary to the title (id prefix). */
  reference: string;
}

const UUID = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi;
const WORKFLOW_LABEL = /^Workflow ([0-9a-f-]{36}) node (\S+)$/i;
const SESSION_LABEL = /^([a-z][\w-]*) session (\S+)$/i;
const TITLE_LIMIT = 90;

/** "codex" → "Codex", "claude-code" → "Claude Code"; unknown kinds are title-cased. */
export function harnessLabel(kind: string): string {
  const runnerType = runnerTypeFromSlug(kind);
  if (runnerType !== undefined) return runnerTypeLabel(runnerType);
  return kind
    .split(/[-_]/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/**
 * Turns free text (a prompt's opening, a derived label) into a one-line title:
 * markup and boilerplate tags removed, whitespace collapsed, first sentence or
 * line kept, long ids shortened, and the result capped at a readable length.
 */
export function readableTitle(text: string, limit = TITLE_LIMIT): string {
  const cleaned = text
    .replace(/<([a-z][\w-]*)[^>]*>[\s\S]*?<\/\1>/gi, " ")
    .replace(/<\/?[a-z][^>]*>/gi, " ")
    .split(/\r?\n/)
    .map((line) => line.replace(/^[#>*\-\s]+/, "").trim())
    .find((line) => line.length > 0) ?? "";
  const shortened = cleaned.replace(/\s+/g, " ").replace(UUID, (id) => id.slice(0, 8));
  if (shortened.length <= limit) return shortened;
  const cut = shortened.slice(0, limit);
  const lastSpace = cut.lastIndexOf(" ");
  return `${(lastSpace > limit * 0.6 ? cut.slice(0, lastSpace) : cut).replace(/[\s,;:.–—-]+$/, "")}…`;
}

function shortSession(sessionId: string, kind: string): string {
  const bare = kind && sessionId.startsWith(`${kind}-`) ? sessionId.slice(kind.length + 1) : sessionId;
  return bare.slice(0, 8);
}

export function runHeading(run: Run, taskTitle?: string): RunHeading {
  const reference = run.id.slice(0, 8);
  const label = run.label.trim();
  const workflow = WORKFLOW_LABEL.exec(label);
  if (workflow?.[1] && workflow[2]) {
    return { title: `Workflow step · ${workflow[2]}`, reference: workflow[1].slice(0, 8) };
  }
  const session = SESSION_LABEL.exec(label);
  const kind = run.harnessKind || run.importSourceHarness || session?.[1] || "";
  if (session && run.labelSource === "harness") {
    const sessionId = run.harnessSessionId || session[2] || "";
    return { title: `${harnessLabel(kind)} session`, reference: shortSession(sessionId, kind) || reference };
  }
  const text = label || taskTitle?.trim() || run.promptPreview.trim();
  const title = text ? readableTitle(text) : "";
  return { title: title || `Run ${reference}`, reference };
}

/**
 * Who ran it: the agent profile when there is one; otherwise where the run came
 * from (an attached or imported coding-agent session). A profile id that does
 * not resolve says so rather than inventing a name.
 */
export function runOrigin(run: Run, profileName?: string): string {
  if (profileName) return profileName;
  if (run.importedAt) {
    return run.importSourceHarness ? `Imported · ${harnessLabel(run.importSourceHarness)}` : "Imported";
  }
  if (run.harnessKind) return `${harnessLabel(run.harnessKind)} · attached`;
  const runnerType = run.resolvedConfig?.runnerType;
  if (runnerType) return runnerTypeLabel(runnerType);
  return run.agentProfileId ? "Profile unavailable" : "No profile";
}

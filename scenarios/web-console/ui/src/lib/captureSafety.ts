/**
 * Will this launch command produce a conversation transcript?
 *
 * Web Console records an agent's messages by reading the transcript that agent
 * writes to disk. For Codex and Grok, finding that transcript depends on the
 * process being started with a session-scoped home directory, and only the
 * governed launcher sets one. A bare `codex --yolo` gets a home only if the
 * PATH shim happens to resolve first — which is a property of the shell the
 * console does not control, and which has silently stopped holding before.
 *
 * The settings editor is the one screen that can say so before the operator
 * finds out from an empty Messages pane, so this classifies a command the way
 * the launcher would run it. It is a static read of the command text: it can
 * be sure a command IS governed, and it can say when one merely *might* be, but
 * it never claims a command is broken — a wrapper script this cannot see into
 * may well do the right thing.
 *
 * [REQ:P0-006b] Configurable Shortcut Entries
 */

export type CaptureVerdict =
  /** Routed through a launcher that always sets the session home. */
  | "governed"
  /** Depends on PATH resolution; capture may silently not happen. */
  | "path-dependent"
  /** Not an agent launch, so there is nothing to capture. */
  | "not-an-agent"
  /** An agent whose capture does not depend on the launch command at all. */
  | "independent"
  /** No command yet. */
  | "empty";

export interface CaptureAssessment {
  verdict: CaptureVerdict;
  /** The launcher form this command resolves through, when it names one. */
  via?: string;
}

/** Launchers that materialize a session-scoped agent home before exec. */
const GOVERNED_FORMS: { pattern: RegExp; via: string }[] = [
  { pattern: /\bvrooli-agent-launcher\b/, via: "vrooli-agent-launcher" },
  { pattern: /\bvrooli\s+agent\s+launch\b/, via: "vrooli agent launch" },
];

/**
 * Agents whose transcript is found without a session-scoped home.
 *
 * Claude Code is identified out of band (a Stop hook plus process
 * attribution) and its transcript path is derivable from the working
 * directory, so how it is started does not decide whether it is captured.
 * OpenCode is read through its own local server API for the same reason.
 */
const HOME_INDEPENDENT_AGENTS = new Set(["claude", "opencode"]);

export function assessCapture(command: string, agentID: string | undefined): CaptureAssessment {
  const text = command.trim();
  if (!text) return { verdict: "empty" };
  if (!agentID) return { verdict: "not-an-agent" };

  for (const form of GOVERNED_FORMS) {
    if (form.pattern.test(text)) return { verdict: "governed", via: form.via };
  }
  if (HOME_INDEPENDENT_AGENTS.has(agentID)) return { verdict: "independent" };
  return { verdict: "path-dependent" };
}

/**
 * The governed rewrite of a command, so the editor can offer a fix rather than
 * only a warning. Returns null when there is nothing to rewrite.
 *
 * Arguments are carried across as `--arg=` so the launcher passes them to the
 * agent verbatim instead of parsing them as its own flags.
 */
export function governedRewrite(command: string, agentID: string | undefined): string | null {
  const text = command.trim();
  if (!text || !agentID) return null;
  if (assessCapture(text, agentID).verdict !== "path-dependent") return null;

  const parts = text.split(/\s+/);
  // Drop the binary itself; whatever follows is the agent's own arguments.
  const args = parts.slice(1).filter((part) => part.length > 0);
  const rewritten = [`vrooli agent launch --runner ${agentID}`, ...args.map((arg) => `--arg=${arg}`)];
  return rewritten.join(" ");
}

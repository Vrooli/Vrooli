// DOC: docs/reference/configuration.md#launcher-shortcuts
// DOC: docs/internal/SEAMS.md#axis-1-shortcut-profiles-p0-006-p1-010
/**
 * Launch shortcut definitions.
 *
 * ── VOLATILE: This file is an extension point. ──
 * When adding new shortcuts, modifying defaults, or wiring in
 * configuration-driven profiles (P1-010), changes should land HERE
 * and not in the launcher component itself.
 *
 * ── This list is a FALLBACK, not a source of truth. ──
 * The server owns the defaults (api/shortcut_profiles.go) and serves them
 * through GetEffective. This copy exists for one case: the RPC failed and the
 * dialog still has to render something. It drifted once — it carried eight
 * entries, four of them "(attributed)" duplicates, long after the server had
 * dropped them — and the launcher grew a whole module to fold duplicates that
 * only this file still produced. Keep it identical to the Go defaults, and
 * prefer deleting an entry here over adding one.
 *
 * [REQ:P0-006b] Configurable Shortcut Entries
 */

export interface ShortcutEntry {
  label: string;
  command: string;
  description?: string;
  /**
   * The coding agent this entry launches, or absent for a plain operator
   * command. Resolved by the server; the UI reads it and never derives it
   * from the command text.
   */
  agentId?: string;
}

/** Default shortcuts: the agent runtimes web-console captures conversations for. */
export const DEFAULT_SHORTCUTS: ShortcutEntry[] = [
  {
    label: "Claude Code",
    command: "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions",
    description: "AI coding assistant with full permissions",
    agentId: "claude",
  },
  {
    label: "Codex",
    command: "codex --yolo",
    description: "OpenAI Codex CLI in auto-approve mode",
    agentId: "codex",
  },
  {
    label: "OpenCode",
    command: "opencode",
    description: "OpenCode TUI — conversation captured via its local server API",
    agentId: "opencode",
  },
  {
    label: "Grok",
    command: "grok",
    description: "xAI Grok CLI — conversation captured from its session transcript",
    agentId: "grok",
  },
  {
    // Formerly a hardcoded card in the launcher's actions block, rendered
    // unconditionally because nothing in this codebase can tell whether the
    // operator is signed in. As a shortcut it is ordinary data: editable,
    // reorderable, and deletable like every other entry — and the agent
    // itself tells the operator when a sign-in is needed.
    //
    // It carries no agentId on purpose: it does not start a Codex session, so
    // it belongs in the commands list, not on the Codex card.
    label: "Codex sign-in",
    command: "codex login --device-auth",
    description: "Device-code sign-in for the Codex CLI",
  },
];

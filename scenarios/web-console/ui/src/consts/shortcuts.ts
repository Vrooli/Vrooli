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
 * [REQ:P0-006b] Configurable Shortcut Entries
 */

export interface ShortcutEntry {
  label: string;
  command: string;
  description?: string;
}

/** Default shortcuts: the agent runtimes web-console captures conversations for. */
export const DEFAULT_SHORTCUTS: ShortcutEntry[] = [
  {
    label: "Claude Code",
    command: "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions",
    description: "AI coding assistant with full permissions",
  },
  {
    label: "Codex",
    command: "codex --yolo",
    description: "OpenAI Codex CLI in auto-approve mode",
  },
  {
    label: "OpenCode",
    command: "opencode",
    description: "OpenCode TUI — conversation captured via its local server API",
  },
  {
    label: "Grok",
    command: "grok",
    description: "xAI Grok CLI — conversation captured from its session transcript",
  },
  {
    label: "Claude Code (attributed)",
    command: "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent claude -- --dangerously-skip-permissions; fi; exec vrooli agent launch --runner claude --arg=--dangerously-skip-permissions",
    description: "Claude Code with best-effort Agent Manager attribution; direct fallback stays available",
  },
  {
    label: "Codex (attributed)",
    command: "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent codex -- --yolo; fi; exec codex --yolo",
    description: "Codex with best-effort Agent Manager attribution; direct fallback stays available",
  },
  {
    label: "OpenCode (attributed)",
    command: "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent opencode --; fi; exec opencode",
    description: "OpenCode with best-effort Agent Manager attribution; direct fallback stays available",
  },
  {
    label: "Grok (attributed)",
    command: "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent grok --; fi; exec grok",
    description: "Grok with best-effort Agent Manager attribution; direct fallback stays available",
  },
];

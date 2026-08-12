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
/** Default shortcuts: the agent runtimes web-console captures conversations for. */
export const DEFAULT_SHORTCUTS = [
    {
        label: "Claude Code",
        command: "claude --dangerously-skip-permissions",
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
];

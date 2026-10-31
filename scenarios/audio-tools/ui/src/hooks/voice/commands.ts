// DOC: docs/internal/SEAMS.md#voice-command-seam
//
// Voice Command Vocabulary — defines the fixed set of terminal commands
// that can be triggered by voice in persistent mode.
//
// Each command has a set of trigger patterns and an execute function that
// receives a CommandContext with action handles. Command logic stays here;
// UI components never execute commands directly.

/** Action handles provided to commands for executing terminal operations. */
export interface CommandContext {
  /** Create a new terminal session tab. */
  createTab: () => void;
  /** Switch to a terminal tab by 1-based index. */
  switchToTab: (index: number) => void;
  /** Close the currently active terminal tab. */
  closeTab: () => void;
  /** Send raw data to the active terminal. */
  sendToTerminal: (data: string) => void;
  /** Exit persistent voice mode. */
  exitVoiceMode: () => void;
}

export interface VoiceCommand {
  id: string;
  /** Human-readable description shown in the suggestion UI. */
  description: string;
  /** Trigger phrases (lowercase). Matched against segment-final text after prefix removal. */
  patterns: string[];
  /** Execute the command. The `args` object contains parsed parameters (e.g., tab number). */
  execute: (context: CommandContext, args: Record<string, unknown>) => void;
}

// ── Fixed command vocabulary ─────────────────────────────────────────

export const VOICE_COMMANDS: VoiceCommand[] = [
  {
    id: "new-tab",
    description: "New Tab",
    patterns: ["new tab", "add tab", "open tab"],
    execute: (ctx) => ctx.createTab(),
  },
  {
    id: "switch-tab",
    description: "Switch Tab",
    patterns: ["tab", "switch tab", "go to tab"],
    execute: (ctx, args) => {
      const n = typeof args.number === "number" ? args.number : 1;
      ctx.switchToTab(n);
    },
  },
  {
    id: "close-tab",
    description: "Close Tab",
    patterns: ["close tab", "close this tab"],
    execute: (ctx) => ctx.closeTab(),
  },
  {
    id: "send-enter",
    description: "Press Enter",
    patterns: ["send", "enter", "submit"],
    execute: (ctx) => ctx.sendToTerminal("\r"),
  },
  {
    id: "cancel",
    description: "Cancel (Ctrl+C)",
    patterns: ["cancel", "interrupt", "stop"],
    execute: (ctx) => ctx.sendToTerminal("\x03"),
  },
  {
    id: "copy",
    description: "Copy",
    patterns: ["copy"],
    execute: (ctx) => ctx.sendToTerminal("\x1b[67;5u"), // CSI for Ctrl+Shift+C
  },
  {
    id: "paste",
    description: "Paste",
    patterns: ["paste"],
    execute: (ctx) => ctx.sendToTerminal("\x1b[86;5u"), // CSI for Ctrl+Shift+V
  },
  {
    id: "clear",
    description: "Clear Screen",
    patterns: ["clear", "clear screen"],
    execute: (ctx) => ctx.sendToTerminal("\x0c"), // Ctrl+L
  },
  {
    id: "tab-key",
    description: "Tab Key (Autocomplete)",
    patterns: ["tab key", "autocomplete"],
    execute: (ctx) => ctx.sendToTerminal("\t"),
  },
  {
    id: "scroll-up",
    description: "Scroll Up",
    patterns: ["scroll up"],
    execute: (ctx) => ctx.sendToTerminal("\x1b[5~"), // Shift+PageUp
  },
  {
    id: "scroll-down",
    description: "Scroll Down",
    patterns: ["scroll down"],
    execute: (ctx) => ctx.sendToTerminal("\x1b[6~"), // Shift+PageDown
  },
  {
    id: "stop-listening",
    description: "Stop Listening",
    patterns: ["stop listening", "mic off"],
    execute: (ctx) => ctx.exitVoiceMode(),
  },
];


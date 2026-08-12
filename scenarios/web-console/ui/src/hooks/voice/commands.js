// DOC: docs/internal/SEAMS.md#voice-command-seam
//
// Voice Command Vocabulary — defines the fixed set of terminal commands
// that can be triggered by voice in persistent mode.
//
// Each command has a set of trigger patterns and an execute function that
// receives a CommandContext with action handles. Command logic stays here;
// UI components never execute commands directly.
// ── Fixed command vocabulary ─────────────────────────────────────────
export const VOICE_COMMANDS = [
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

/**
 * Returns true when xterm is in any mouse-tracking mode that would
 * interpret pasted bytes as mouse events rather than text. Only
 * pasted multi-byte input is blocked — individual keystrokes from
 * xterm.onData already pass through the terminal's own event handling
 * and are not affected.
 */
export function terminalIsInMouseTrackingMode(t) {
    if (!t)
        return false;
    const modes = t.modes;
    if (!modes)
        return false;
    return modes.mouseTrackingMode !== "none";
}
/**
 * wireKindFor maps the UI-level input source to the wire-level kind
 * discriminator. Only "paste" goes on the paste path; everything else
 * (xterm typing, toolbar keys/submit, voice, upload-triggered stdin)
 * is delivered as keystrokes.
 */
export function wireKindFor(source) {
    return source === "paste" ? "paste" : "keystroke";
}
export function createInputGate(opts) {
    let disposed = false;
    const canAcceptPaste = () => !terminalIsInMouseTrackingMode(opts.getTerminal());
    const submit = (data, source) => {
        if (disposed)
            return { status: "rejected", reason: "disposed" };
        if (!data)
            return { status: "rejected", reason: "empty" };
        const kind = wireKindFor(source);
        // External pause (voice mode etc.) blocks every source uniformly.
        if (opts.isPaused?.() === true) {
            opts.transport.enqueue(data, kind);
            return { status: "queued", reason: "paused" };
        }
        // Paste-specific client-side mode gating: mouse-tracking TUIs
        // running INSIDE xterm consume bytes as mouse events at the
        // browser layer (before the WS frame is sent). Hold paste
        // payloads until the TUI exits that mode. Other sources
        // (keystrokes, toolbar keys, voice) are one byte at a time and
        // don't trigger the same misinterpretation. Tmux-side modes
        // (copy-mode, command-prompt, menu) are handled server-side via
        // paste-buffer and need no gating here.
        if (source === "paste" && terminalIsInMouseTrackingMode(opts.getTerminal())) {
            opts.transport.enqueue(data, kind);
            return { status: "queued", reason: "paused" };
        }
        const res = opts.transport.send(data, kind);
        if (res.sent && typeof res.seq === "number") {
            return { status: "sent", seq: res.seq };
        }
        opts.transport.enqueue(data, kind);
        return { status: "queued", reason: res.reason ?? "not-ready" };
    };
    const dispose = () => {
        disposed = true;
    };
    return { submit, dispose, canAcceptPaste };
}

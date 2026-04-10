import { useEffect } from "react";
import type { Terminal } from "@xterm/xterm";

/**
 * DEL character (0x7f) — the escape sequence xterm.js sends for backspace.
 * This is what the terminal server expects when the user presses backspace.
 */
const DEL = "\x7f";

/**
 * Sentinel string kept inside xterm's hidden textarea so the mobile virtual
 * keyboard always has content to "delete". Without this, holding backspace
 * on a mobile keyboard stops firing events once the textarea is empty —
 * the browser thinks there's nothing left to delete.
 *
 * We use a short repeating sequence of zero-width spaces. These are:
 * - Invisible, so they don't flash if the textarea briefly becomes visible
 * - Long enough (32 chars) that rapid key-repeat doesn't exhaust the buffer
 *   before our replenish handler fires
 *
 * The textarea content is purely a browser-level trick — xterm.js reads
 * keyboard input through its own event pipeline (onData), not from the
 * textarea's value. So this padding never leaks into terminal output.
 */
const PADDING = "\u200B".repeat(32);

/**
 * useMobileBackspaceRepeat — Enables hold-to-delete on mobile virtual keyboards.
 *
 * ## Problem
 * xterm.js captures keyboard input via a hidden <textarea>. On mobile,
 * when the user holds the backspace key, the browser fires `beforeinput`
 * events with `inputType: "deleteContentBackward"`. But it only fires
 * these while the textarea has content to delete. Since xterm's textarea
 * is typically empty, the browser fires one event and stops — so holding
 * backspace deletes only one character.
 *
 * ## Solution
 * 1. Keep the textarea filled with invisible padding (zero-width spaces).
 * 2. Listen for `beforeinput` events with `inputType: "deleteContentBackward"`.
 * 3. On each event, prevent the default deletion (so xterm doesn't see a
 *    DOM change it doesn't expect) and manually inject DEL (0x7f) into
 *    xterm's input stream via `terminal.input()` (the internal input method).
 * 4. Replenish the padding so the next repeat event still has content to delete.
 *
 * This way, the browser keeps firing repeat events as long as the key is
 * held, and each event produces a backspace in the terminal.
 *
 * ## Why `beforeinput` instead of `input`?
 * - `beforeinput` fires before the DOM is modified, letting us preventDefault
 *   to avoid confusing xterm's own input tracking.
 * - `input` fires after the DOM change, which would require us to undo it.
 * - `beforeinput` is supported on all modern mobile browsers (Chrome 60+,
 *   Safari 13+, Firefox 87+).
 *
 * ## Why this is safe
 * - The padding is zero-width spaces, which are not printable terminal chars.
 * - We only intercept "deleteContentBackward" — all other input types
 *   (insertText, insertCompositionText, etc.) pass through to xterm normally.
 * - `terminal.input()` feeds directly into xterm's input pipeline, which is
 *   the same path physical keyboard events take.
 * - The hook is only active on touch devices, so desktop behavior is unchanged.
 */
export function useMobileBackspaceRepeat(terminal: Terminal | null): void {
  useEffect(() => {
    if (!terminal) return;

    const isTouchDevice = "ontouchstart" in window || navigator.maxTouchPoints > 0;
    if (!isTouchDevice) return;

    const textarea = terminal.textarea;
    if (!textarea) return;

    // Seed the textarea with padding so the first backspace has content to delete.
    textarea.value = PADDING;

    function handleBeforeInput(e: InputEvent) {
      if (e.inputType !== "deleteContentBackward") return;

      // Prevent the browser from modifying the textarea — we handle it ourselves.
      e.preventDefault();

      // Inject a DEL character into xterm's input stream.
      // terminal.input() is xterm's internal method that feeds data into the
      // same pipeline as physical keyboard events, triggering onData listeners.
      (terminal as Terminal & { input: (data: string, wasUserInput?: boolean) => void })
        .input(DEL, true);

      // Replenish the textarea so the browser keeps firing repeat events.
      // Safety: textarea is guaranteed non-null here since we checked at the
      // top of the effect and this closure only runs while the listener is attached.
      if (textarea) textarea.value = PADDING;
    }

    // xterm.js clears/resets the textarea value during normal operation
    // (after processing typed characters, on focus, during composition, etc.).
    // When that happens our padding is lost, so the next time the user holds
    // backspace the browser sees an empty textarea and stops firing repeat
    // events after a single deletion.
    //
    // We replenish the padding asynchronously via requestAnimationFrame so
    // xterm finishes reading/clearing the textarea before we refill it.
    function ensurePadding() {
      requestAnimationFrame(() => {
        if (textarea && !textarea.value) {
          textarea.value = PADDING;
        }
      });
    }

    textarea.addEventListener("beforeinput", handleBeforeInput);
    textarea.addEventListener("input", ensurePadding);
    textarea.addEventListener("focus", ensurePadding);

    return () => {
      textarea.removeEventListener("beforeinput", handleBeforeInput);
      textarea.removeEventListener("input", ensurePadding);
      textarea.removeEventListener("focus", ensurePadding);
    };
  }, [terminal]);
}

import * as React from "react";

/**
 * useKeyboardShortcut — register a single, global keyboard shortcut.
 *
 * Per `react-coherence` §0.5 there is one shortcut manager for the whole
 * shell. This hook is the public API for that manager: components register
 * their interest, the hook attaches one `keydown` listener per-mount, and
 * the matcher fires the handler when the chord matches.
 *
 * The chord format is `"mod+k"`, `"shift+/"`, `"escape"`, etc. Modifier
 * tokens are case-insensitive. `mod` means `metaKey` on macOS-style
 * platforms (Cmd) and `ctrlKey` everywhere else.
 */
export interface KeyboardShortcutSpec {
  /** Chord like `"mod+k"` or `"escape"`. */
  chord: string;
  /** Handler. The KeyboardEvent is forwarded for preventDefault, etc. */
  handler: (event: KeyboardEvent) => void;
  /** When false, the hook does not attach. Defaults to true. */
  enabled?: boolean;
  /** Optional target. Defaults to `window`. */
  target?: EventTarget;
}

interface ParsedChord {
  key: string;
  mod: boolean;
  shift: boolean;
  alt: boolean;
}

const parseChord = (chord: string): ParsedChord => {
  const parts = chord.toLowerCase().split("+").map((s) => s.trim()).filter(Boolean);
  let mod = false;
  let shift = false;
  let alt = false;
  let key = "";
  for (const p of parts) {
    if (p === "mod" || p === "ctrl" || p === "cmd" || p === "meta") mod = true;
    else if (p === "shift") shift = true;
    else if (p === "alt" || p === "option") alt = true;
    else key = p;
  }
  return { key, mod, shift, alt };
};

const isMac = (): boolean => {
  if (typeof navigator === "undefined") return false;
  // navigator.userAgentData is the modern API; fall back to userAgent string.
  const ua = navigator.userAgent;
  return /Mac|iPhone|iPad|iPod/i.test(ua);
};

const matchesChord = (event: KeyboardEvent, spec: ParsedChord): boolean => {
  if (event.key.toLowerCase() !== spec.key) return false;
  const modPressed = isMac() ? event.metaKey : event.ctrlKey;
  if (spec.mod !== modPressed) return false;
  if (spec.shift !== event.shiftKey) return false;
  if (spec.alt !== event.altKey) return false;
  return true;
};

export function useKeyboardShortcut({
  chord,
  handler,
  enabled = true,
  target,
}: KeyboardShortcutSpec): void {
  const handlerRef = React.useRef(handler);
  handlerRef.current = handler;

  React.useEffect(() => {
    if (!enabled) return;
    const t: EventTarget = target ?? (typeof window === "undefined" ? new EventTarget() : window);
    const parsed = parseChord(chord);
    if (!parsed.key) return;
    const listener: EventListener = (event) => {
      const ke = event as KeyboardEvent;
      if (matchesChord(ke, parsed)) handlerRef.current(ke);
    };
    t.addEventListener("keydown", listener);
    return () => t.removeEventListener("keydown", listener);
  }, [chord, enabled, target]);
}

import * as React from "react";
import { cn } from "../lib/utils";
import { selectors } from "../consts/selectors";
import { Kbd } from "./ui/kbd";

export interface KeyboardShortcutHintProps {
  /** Chord like `"mod+k"` — same syntax as useKeyboardShortcut. */
  chord: string;
  className?: string;
}

const MOD_LABEL = (): string => {
  if (typeof navigator === "undefined") return "Ctrl";
  return /Mac|iPhone|iPad|iPod/i.test(navigator.userAgent) ? "⌘" : "Ctrl";
};

const TOKEN_LABEL: Record<string, string> = {
  mod: "",
  ctrl: "Ctrl",
  cmd: "⌘",
  meta: "⌘",
  shift: "Shift",
  alt: "Alt",
  option: "Alt",
  escape: "Esc",
  enter: "↵",
  arrowleft: "←",
  arrowright: "→",
  arrowup: "↑",
  arrowdown: "↓",
};

const labelForToken = (token: string): string => {
  if (token === "mod") return MOD_LABEL();
  return TOKEN_LABEL[token] ?? token.toUpperCase();
};

export function KeyboardShortcutHint({ chord, className }: KeyboardShortcutHintProps) {
  const tokens = chord
    .toLowerCase()
    .split("+")
    .map((t) => t.trim())
    .filter(Boolean);

  return (
    <span
      data-testid={selectors.shared.keyboardShortcut.root}
      className={cn("inline-flex items-center gap-1", className)}
    >
      {tokens.map((tok, i) => (
        <React.Fragment key={`${tok}-${i}`}>
          {i > 0 ? <span className="text-app-muted-foreground">+</span> : null}
          <Kbd>{labelForToken(tok)}</Kbd>
        </React.Fragment>
      ))}
    </span>
  );
}

import { useState } from "react";
import { useIconMorph } from "./useIconMorph";

function BubbleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      <path d="M13 8H7" />
      <path d="M17 12H7" />
    </svg>
  );
}

function TerminalIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
      <path d="m7 11 2-2-2-2" />
      <path d="M11 13h4" />
      <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
    </svg>
  );
}

/**
 * The hook is behaviour-only: it decides a technique and drives a clock, and
 * leaves rendering to its caller. This specimen is the smallest thing that
 * exercises that contract — a swap, and the decision it produces.
 */
export function Default() {
  const [terminal, setTerminal] = useState(false);
  const key = terminal ? "terminal" : "bubble";
  const { technique, active, currentRef } = useIconMorph({ iconKey: key });
  return (
    <div data-rcl-hook-root data-testid="hooks.use-icon-morph">
      <button
        type="button"
        data-rcl-hook-action="start"
        onClick={() => {
          setTerminal((value) => !value);
        }}
      >
        swap
      </button>
      <span ref={currentRef} style={{ display: "inline-block", inlineSize: 24, blockSize: 24 }}>
        {terminal ? <TerminalIcon /> : <BubbleIcon />}
      </span>
      <output data-testid="hooks.use-icon-morph-technique">{active ? technique : "idle"}</output>
    </div>
  );
}

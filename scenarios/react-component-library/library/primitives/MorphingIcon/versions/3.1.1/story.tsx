import { useState } from "react";
import { MorphingIcon } from "./MorphingIcon";

type StoryProps = {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
};

function BubbleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      <path d="M13 8H7" />
      <path d="M17 12H7" />
    </svg>
  );
}

function TerminalIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m7 11 2-2-2-2" />
      <path d="M11 13h4" />
      <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
    </svg>
  );
}

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2" /><path d="M12 20v2" /><path d="M2 12h2" /><path d="M20 12h2" />
      <path d="m4.93 4.93 1.41 1.41" /><path d="m17.66 17.66 1.41 1.41" />
      <path d="m6.34 17.66-1.41 1.41" /><path d="m19.07 4.93-1.41 1.41" />
    </svg>
  );
}

/** Any icon component at all — no registry entry, no geometry table. */
export function Default({ args, log }: StoryProps) {
  void log;
  return (
    <MorphingIcon {...args} size="lg">
      <BubbleIcon />
    </MorphingIcon>
  );
}

export function Labelled({ args, log }: StoryProps) {
  void log;
  return (
    <MorphingIcon {...args} size="lg" label="Messages">
      <BubbleIcon />
    </MorphingIcon>
  );
}

export function RegistryGlyph({ args, log }: StoryProps) {
  void log;
  return <MorphingIcon {...args} size="lg" icon="close" />;
}

/**
 * The regression that motivated the rewrite. This glyph's circle is two arc
 * commands; the 2.x parser dropped every curve command, so it rendered as a
 * bare diagonal line.
 */
export function SearchGlyph({ args, log }: StoryProps) {
  void log;
  return <MorphingIcon {...args} size="lg" icon="search" />;
}

export function SizeSmall({ args, log }: StoryProps) {
  void log;
  return (
    <MorphingIcon {...args} size="sm">
      <BubbleIcon />
    </MorphingIcon>
  );
}

/** A compatible pair: measured at 0.862, comfortably above the morph threshold. */
export function CompatibleSwap({ args, log }: StoryProps) {
  const [terminal, setTerminal] = useState(false);
  return (
    <button
      type="button"
      data-testid="morph-swap-trigger"
      onClick={() => {
        setTerminal((value) => !value);
        log("swap");
      }}
    >
      <MorphingIcon {...args} size="lg">
        {terminal ? <TerminalIcon /> : <BubbleIcon />}
      </MorphingIcon>
    </button>
  );
}

/** An incompatible pair: nine strokes against three, so it crossfades instead. */
export function IncompatibleSwap({ args, log }: StoryProps) {
  const [bubble, setBubble] = useState(false);
  return (
    <button
      type="button"
      data-testid="morph-swap-trigger"
      onClick={() => {
        setBubble((value) => !value);
        log("swap");
      }}
    >
      <MorphingIcon {...args} size="lg">
        {bubble ? <BubbleIcon /> : <SunIcon />}
      </MorphingIcon>
    </button>
  );
}

/** Motion pinned off, for callers that own their own transition. */
export function NoTransition({ args, log }: StoryProps) {
  const [terminal, setTerminal] = useState(false);
  return (
    <button
      type="button"
      data-testid="morph-swap-trigger"
      onClick={() => {
        setTerminal((value) => !value);
        log("swap");
      }}
    >
      <MorphingIcon {...args} size="lg" morph="none">
        {terminal ? <TerminalIcon /> : <BubbleIcon />}
      </MorphingIcon>
    </button>
  );
}

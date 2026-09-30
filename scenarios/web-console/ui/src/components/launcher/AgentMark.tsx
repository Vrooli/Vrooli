import type { AgentAppearance } from "./agentAppearance";
import { agentAppearance } from "./agentAppearance";

/**
 * The mark that identifies a coding agent in the launcher grid.
 *
 * Every agent card used to carry the same lightning bolt. Seven identical
 * glyphs carry no information: they cost a column of space and return nothing
 * an operator can aim at. These are one geometric mark per agent on a tinted
 * plate, so the hue alone locates the right card before any label is read.
 *
 * They are deliberately NOT vendor logos. A logo would need a licensed asset
 * per agent, would leave any new or self-hosted agent with a blank square, and
 * would have to be re-cut for both themes. A stroke mark drawn in currentColor
 * works at 13px, in both themes, for an agent nobody has drawn art for yet.
 *
 * [REQ:P0-014a] Launcher Destination And Appearance Disclosure
 */

/**
 * The mark geometry, one entry per agent plus the two non-agent cards.
 *
 * Paths are stroked, never filled, so a single definition renders correctly at
 * any size and in any theme. `vector-effect` is deliberately absent: these are
 * drawn at one size and scaling them would thin the stroke below legibility.
 */
const MARKS: Record<string, React.JSX.Element> = {
  claude: <path d="M8 1.6v12.8M2.5 4.8l11 6.4M13.5 4.8l-11 6.4" />,
  codex: (
    <>
      <circle cx="8" cy="8" r="2.1" />
      <path d="M8 1.9a6.1 6.1 0 0 1 5.28 9.15M8 14.1a6.1 6.1 0 0 1-5.28-9.15" />
    </>
  ),
  opencode: <path d="M5.6 3.4 1.8 8l3.8 4.6M10.4 3.4 14.2 8l-3.8 4.6" />,
  grok: <path d="M2.4 13.6 13.6 2.4M9.4 2.4h4.2v4.2M2.4 6.2 6.1 2.5" />,
  agy: <path d="M8 2.2 13 9.4H3zM4.4 12.9h7.2" />,
  /** The plain terminal: a prompt caret and an input rule. */
  shell: <path d="M2.6 3.6 6.8 8l-4.2 4.4M8.4 12.6h5" />,
  /** The editor card: two sliders, the settings idiom already used elsewhere. */
  edit: (
    <>
      <path d="M2.4 4.6h11M2.4 11.4h11" />
      <circle cx="6" cy="4.6" r="1.7" />
      <circle cx="10.2" cy="11.4" r="1.7" />
    </>
  ),
  /** A command with no agent: a generic command chevron. */
  command: <path d="M3.2 4.2 7.2 8l-4 3.8M8.6 12h4.2" />,
};

export interface AgentMarkProps {
  /** Capability id, or one of the reserved marks: "shell", "edit", "command". */
  mark: string;
  /** Dim the plate for a card that cannot be launched right now. */
  muted?: boolean;
  /** Override the plate and ink, used by the missing/installing card states. */
  appearance?: AgentAppearance;
  className?: string;
}

/**
 * A 24px plate carrying one agent mark.
 *
 * Rendered inline rather than through a sprite: the launcher mounts and
 * unmounts inside a dialog, and a document-level <defs> sprite that outlives
 * the dialog would leak referenced geometry the dialog no longer owns.
 */
export default function AgentMark({ mark, muted = false, appearance, className }: AgentMarkProps) {
  const look = appearance ?? agentAppearance(mark);
  const glyph = MARKS[mark] ?? MARKS.command;
  return (
    <span
      aria-hidden
      data-testid={`agent-mark-${mark}`}
      className={`grid h-6 w-6 shrink-0 place-items-center rounded-lg border border-white/[0.07] ${className ?? ""}`}
      style={{ background: look.plate, opacity: muted ? 0.6 : 1 }}
    >
      <svg
        viewBox="0 0 16 16"
        className="h-[13px] w-[13px]"
        fill="none"
        stroke={look.ink}
        strokeWidth={mark === "opencode" || mark === "grok" || mark === "shell" ? 1.7 : 1.6}
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        {glyph}
      </svg>
    </span>
  );
}

/** The six-dot grip that appears on a card in reorder mode. */
export function ReorderGrip({ active = false }: { active?: boolean }) {
  return (
    <svg
      aria-hidden
      viewBox="0 0 9 15"
      className={`h-[15px] w-[9px] shrink-0 ${active ? "text-wc-accent" : "text-wc-text-faint"}`}
      fill="currentColor"
    >
      <circle cx="2" cy="2" r="1.15" />
      <circle cx="7" cy="2" r="1.15" />
      <circle cx="2" cy="7.5" r="1.15" />
      <circle cx="7" cy="7.5" r="1.15" />
      <circle cx="2" cy="13" r="1.15" />
      <circle cx="7" cy="13" r="1.15" />
    </svg>
  );
}

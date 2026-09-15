import { RUNG_ORDER, rungToken, signalToken, type Rung, type SignalState } from "../../theme/instrument";

/**
 * Maps a signal state onto the stroke used for one ring segment.
 *
 * These read from the same custom properties the lamp does, so the ring and
 * the grid can never drift apart. SVG `stroke` accepts `var()`, so no colour
 * literal is introduced here.
 */
const SEGMENT_STROKE: Record<SignalState, string> = {
  COVERED: "var(--signal-covered)",
  PARTIAL: "var(--signal-partial)",
  EXCURSION: "var(--signal-excursion)",
  // Everything below is unlit, and the differences between them are carried by
  // stroke weight and hue rather than by brightness: the ring's job is to make
  // the LIT arcs read at thumbnail size, and four competing dark treatments
  // would turn the hole back into noise. The grid and the drill-down carry the
  // distinction in full.
  UNMEASURABLE: "var(--color-border-lit)",
  UNAVAILABLE: "var(--color-border)",
  NOT_APPLICABLE: "var(--signal-blind)",
  UNAUTHORED: "var(--signal-blind)",
  BLIND: "var(--signal-blind)",
  SOURCE_DOWN: "var(--color-border)",
  HOST_NOT_SAMPLED: "var(--signal-blind-edge)",
};

/** States whose segments participate in the power-on lamp test. */
const LIVE_STATES: ReadonlySet<SignalState> = new Set<SignalState>([
  "COVERED",
  "PARTIAL",
  "EXCURSION",
]);

export interface RungRingProps {
  cx: number;
  cy: number;
  radius: number;
  /** One state per ladder rung. */
  states: Readonly<Record<Rung, SignalState>>;
  /** Stroke width of each arc. */
  weight?: number;
  /**
   * Staggers the power-on lamp test so a constellation lights up in sequence
   * rather than all at once. Ignored under `prefers-reduced-motion`.
   */
  animationDelay?: number;
}

/**
 * The segmented rung ring: five arcs around one device node, one per ladder
 * rung, drawn clockwise from twelve o'clock in dependency order.
 *
 * The ring exists so the SHAPE of a device's coverage reads before any label
 * does. A device with two lit arcs and three dark ones is legible at thumbnail
 * size, which is what lets the constellation communicate "these parts of the
 * machine are unwatched" to somebody who never reads a single word on it.
 *
 * This renders SVG elements only, so it must be used inside an `<svg>`.
 */
export function RungRing({ cx, cy, radius, states, weight = 6, animationDelay = 0 }: RungRingProps) {
  const circumference = 2 * Math.PI * radius;
  const segment = circumference / RUNG_ORDER.length;
  // A hairline gap between arcs so five segments read as five, not as a ring.
  const gap = Math.min(7, segment * 0.16);
  const dash = segment - gap;

  return (
    <g transform={`rotate(-90 ${cx} ${cy})`} aria-hidden="true">
      {RUNG_ORDER.map((rung, index) => {
        const state = states[rung];
        const live = LIVE_STATES.has(state);
        return (
          <circle
            key={rung}
            className={live ? "ring__segment ring__segment--live" : "ring__segment"}
            cx={cx}
            cy={cy}
            r={radius}
            stroke={SEGMENT_STROKE[state]}
            strokeWidth={weight}
            strokeDasharray={`${dash} ${circumference - dash}`}
            strokeDashoffset={-segment * index}
            style={live ? { animationDelay: `${animationDelay + index * 0.05}s` } : undefined}
          />
        );
      })}
    </g>
  );
}

/**
 * Builds the sentence a screen-reader user hears in place of one ring.
 *
 * The constellation is decorative to assistive technology; this text is not.
 * It is the finding, stated in words, and it must say the same thing the
 * picture says — including which rungs are blind and why.
 */
export function describeRungStates(
  deviceLabel: string,
  states: Readonly<Record<Rung, SignalState>>,
): string {
  const covered = RUNG_ORDER.filter((rung) => states[rung] === "COVERED").length;
  const detail = RUNG_ORDER.map(
    (rung) => `${rungToken(rung).label} ${signalToken(states[rung]).label.toLowerCase()}`,
  ).join(", ");
  return `${deviceLabel}: ${covered} of ${RUNG_ORDER.length} rungs covered. ${detail}.`;
}

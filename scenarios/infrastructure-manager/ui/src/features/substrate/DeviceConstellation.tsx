import { useId } from "react";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";

import { RungRing, describeRungStates } from "../../components/instrument/RungRing";
import { RUNG_ORDER, type SignalState } from "../../theme/instrument";
import { type DeviceClassGroup } from "./model";

/**
 * The device constellation.
 *
 * The host sits at the centre and every device class orbits it, connected by
 * real parent edges. Each node carries a five-segment ring showing which rungs
 * of the observability ladder are covered.
 *
 * The load-bearing property is that HOLES READ BEFORE LABELS. A class with no
 * covered rung renders as a dashed outline with no lit segment, so a reader
 * seeing this at thumbnail size — a screenshot in a feed — sees which parts of
 * the machine are unwatched before reading a single word. That is the one idea
 * the whole surface exists to communicate, and it is why the geometry is fixed
 * rather than force-directed: a layout that reflows on every read cannot be
 * recognised twice.
 */

const VIEWBOX = { width: 920, height: 620 } as const;
const CENTRE = { x: VIEWBOX.width / 2, y: VIEWBOX.height / 2 } as const;
const ORBIT = 215;
const NODE_RADIUS = 36;
const HUB_RADIUS = 46;

export interface DeviceConstellationProps {
  hostName: string;
  groups: readonly DeviceClassGroup[];
  /** Invoked when a class node is activated. */
  onSelectClass?: (deviceClass: string) => void;
  selectedClass?: string | null;
  /** Accessible summary sentence for the whole shape. */
  summary: string;
}

interface PlacedGroup {
  group: DeviceClassGroup;
  x: number;
  y: number;
  /** True when no rung on this class is covered — the shape's holes. */
  unseen: boolean;
}

/**
 * Places class nodes evenly around the hub, starting at twelve o'clock and
 * going clockwise. Even spacing is deliberate: an angle that encoded severity
 * would make the ring's shape depend on the reading, and the reader would lose
 * the ability to find the same device in the same place twice.
 */
function place(groups: readonly DeviceClassGroup[]): readonly PlacedGroup[] {
  const count = Math.max(groups.length, 1);
  return groups.map((group, index) => {
    const angle = (index / count) * Math.PI * 2 - Math.PI / 2;
    return {
      group,
      x: CENTRE.x + Math.cos(angle) * ORBIT,
      y: CENTRE.y + Math.sin(angle) * ORBIT,
      unseen: RUNG_ORDER.every((rung) => group.rungs[rung] !== "COVERED"),
    };
  });
}

/** Wraps a class name onto two lines so long names do not overrun their node. */
function splitLabel(value: string): readonly string[] {
  const words = value.split(/[\s-]+/);
  if (words.length < 2) return [value];
  const midpoint = Math.ceil(words.length / 2);
  return [words.slice(0, midpoint).join(" "), words.slice(midpoint).join(" ")];
}

/** A compact tag for the node centre, e.g. "graphics-device" -> "GFX". */
function nodeTag(value: string): string {
  const head = value.split(/[\s-]+/)[0] ?? value;
  return head.slice(0, 3).toUpperCase();
}

export function DeviceConstellation({
  hostName,
  groups,
  onSelectClass,
  selectedClass,
  summary,
}: DeviceConstellationProps) {
  const { t } = useTranslation();
  const titleId = useId();
  const descId = useId();
  const placed = place(groups);

  return (
    <div className="panel p-space-sm">
      <svg
        viewBox={`0 0 ${VIEWBOX.width} ${VIEWBOX.height}`}
        role="img"
        aria-labelledby={`${titleId} ${descId}`}
        preserveAspectRatio="xMidYMid meet"
        className="block w-full h-auto"
      >
        <title id={titleId}>
          {t(strings.pages.substrate.constellationTitle, { host: hostName })}
        </title>
        <desc id={descId}>{summary}</desc>

        {/*
          Edges are drawn first so nodes cover their ends. An edge to an unseen
          class is dashed: the connection is declared by the device tree, but
          nothing flows along it.
        */}
        <g fill="none" strokeWidth={1}>
          {placed.map((entry) => (
            <path
              key={`edge-${entry.group.deviceClass}`}
              d={`M${CENTRE.x} ${CENTRE.y} L${entry.x} ${entry.y}`}
              stroke={entry.unseen ? "var(--color-border-lit)" : "var(--color-border)"}
              strokeDasharray={entry.unseen ? "3 5" : undefined}
            />
          ))}
        </g>

        {/* The hub: the machine itself. */}
        <circle
          cx={CENTRE.x}
          cy={CENTRE.y}
          r={HUB_RADIUS}
          fill="var(--color-surface-raised)"
          stroke="var(--color-border-lit)"
        />
        <text
          x={CENTRE.x}
          y={CENTRE.y - 2}
          textAnchor="middle"
          fill="var(--color-foreground)"
          fontFamily="var(--font-display)"
          fontSize={19}
          fontWeight={600}
          letterSpacing={1.6}
        >
          {hostName.toUpperCase()}
        </text>
        <text
          x={CENTRE.x}
          y={CENTRE.y + 16}
          textAnchor="middle"
          fill="var(--color-subtle-foreground)"
          fontFamily="var(--font-mono)"
          fontSize={11}
          letterSpacing={0.6}
        >
          {t(strings.pages.substrate.hubClasses, { count: groups.length })}
        </text>

        {placed.map((entry, index) => (
          <ConstellationNode
            key={entry.group.deviceClass}
            entry={entry}
            index={index}
            selected={selectedClass === entry.group.deviceClass}
            onSelect={onSelectClass}
            summaryLabel={t(strings.pages.substrate.nodeSummary, {
              members: entry.group.members.length,
              covered: RUNG_ORDER.filter((rung) => entry.group.rungs[rung] === "COVERED").length,
              rungs: RUNG_ORDER.length,
            })}
          />
        ))}
      </svg>
    </div>
  );
}

interface ConstellationNodeProps {
  entry: PlacedGroup;
  index: number;
  selected: boolean;
  onSelect?: (deviceClass: string) => void;
  /** Pre-translated "N members, C of R covered" line. */
  summaryLabel: string;
}

function ConstellationNode({ entry, index, selected, onSelect, summaryLabel }: ConstellationNodeProps) {
  const { group, x, y, unseen } = entry;
  const label = describeRungStates(group.deviceClass, group.rungs);
  const lines = splitLabel(group.deviceClass);
  const interactive = Boolean(onSelect);

  return (
    <g
      role={interactive ? "button" : undefined}
      tabIndex={interactive ? 0 : undefined}
      aria-label={interactive ? label : undefined}
      aria-pressed={interactive ? selected : undefined}
      onClick={interactive ? () => onSelect?.(group.deviceClass) : undefined}
      onKeyDown={
        interactive
          ? (event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onSelect?.(group.deviceClass);
              }
            }
          : undefined
      }
      className={interactive ? "cursor-pointer focus:outline-none" : undefined}
      style={interactive ? { outlineOffset: 4 } : undefined}
    >
      <RungRing
        cx={x}
        cy={y}
        radius={NODE_RADIUS}
        states={group.rungs}
        animationDelay={index * 0.06}
      />

      {/*
        An unseen class gets a dashed outline instead of a solid disc. This is
        the visual grammar of the whole board: a device the platform cannot see
        is drawn as an outline of a device, not as a device.
      */}
      <circle
        cx={x}
        cy={y}
        r={NODE_RADIUS - 11}
        fill={unseen ? "transparent" : "var(--color-surface-raised)"}
        stroke={selected ? "var(--signal-covered)" : "var(--color-border-lit)"}
        strokeWidth={selected ? 2 : 1}
        strokeDasharray={unseen ? "4 4" : undefined}
      />
      <text
        x={x}
        y={y + 5}
        textAnchor="middle"
        fill={unseen ? "var(--signal-blind-text)" : "var(--color-foreground)"}
        fontFamily="var(--font-mono)"
        fontSize={13}
        fontWeight={600}
        letterSpacing={1}
        pointerEvents="none"
      >
        {nodeTag(group.deviceClass)}
      </text>

      {/* Labels sit outside the ring, pushed away from the hub. */}
      {lines.map((line, lineIndex) => (
        <text
          key={line}
          x={x}
          y={y + NODE_RADIUS + 20 + lineIndex * 15}
          textAnchor="middle"
          fill="var(--color-foreground)"
          fontFamily="var(--font-sans)"
          fontSize={13}
          fontWeight={600}
          pointerEvents="none"
        >
          {line}
        </text>
      ))}
      <text
        x={x}
        y={y + NODE_RADIUS + 20 + lines.length * 15}
        textAnchor="middle"
        fill={unseen ? "var(--signal-excursion)" : "var(--color-subtle-foreground)"}
        fontFamily="var(--font-mono)"
        fontSize={11}
        pointerEvents="none"
      >
        {summaryLabel}
      </text>
    </g>
  );
}

/**
 * Builds the text alternative for the constellation.
 *
 * This is not a caption — it is the finding, stated in words, and a
 * screen-reader user must get the same conclusion a sighted reader gets from
 * the shape: which parts of the machine are unwatched.
 */
export function describeConstellation(
  hostName: string,
  groups: readonly DeviceClassGroup[],
): string {
  if (groups.length === 0) {
    return `No device classes were enumerated for ${hostName}. The board has read nothing, which is not the same as the machine having nothing attached.`;
  }
  const unseen = groups.filter((group) =>
    RUNG_ORDER.every((rung) => group.rungs[rung] !== "COVERED"),
  );
  const head =
    `${groups.length} device classes are arranged around host ${hostName}. ` +
    `Each carries a five-segment ring showing which rungs of the observability ladder are covered.`;
  const holes =
    unseen.length === 0
      ? " Every class has at least one covered rung."
      : ` ${unseen.length} of them have no covered rung at all and appear as dashed outlines: ` +
        `${unseen.map((group) => group.deviceClass).join(", ")}.`;
  const detail = groups
    .map((group) => describeRungStates(group.deviceClass, group.rungs))
    .join(" ");
  return `${head}${holes} ${detail}`;
}

/** Re-exported so callers can reason about node state without re-deriving it. */
export function classIsUnseen(rungs: Readonly<Record<string, SignalState>>): boolean {
  return RUNG_ORDER.every((rung) => rungs[rung] !== "COVERED");
}

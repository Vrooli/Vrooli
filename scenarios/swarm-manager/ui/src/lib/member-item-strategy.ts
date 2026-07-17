/**
 * Member-item strategy mapping module — THE single place the UI translates the
 * `item-level` wire vocabulary into the declarative member-item-strategy
 * vocabulary.
 *
 * Domain truth (packages/proto/schemas/swarm-manager/v1/domain/
 * agent_operations.proto, `AgentOpsMemberItemStrategy`): "item-level" is NOT a
 * methodology loop. It has no phase graph, no mode.json, and no server
 * catalog entry (the item-level pseudo-mode was deleted in Phase 9). It is
 * workflow strategy configuration on the initiative — each member item runs
 * its own operation; the initiative only coordinates scheduling.
 *
 * ## Wire-value stance
 * Initiatives persist `mode: "item-level"` (or blank, which has always meant
 * the same thing) as the strategy sentinel. This module owns both the
 * presentation mapping and the client-side catalog synthesis:
 * - Services keep READING and WRITING the literal wire value
 *   {@link MEMBER_ITEM_STRATEGY_WIRE_VALUE} (`"item-level"`). Switching an
 *   initiative to the strategy still goes through the existing switch-mode
 *   mutation with this wire value — the server treats it as a plain field
 *   write with no mode definition behind it.
 * - The server operating-mode catalog contains only genuine modes; the UI
 *   synthesizes the strategy's catalog entry via
 *   {@link memberItemStrategyCatalogEntry} (initiative-mode-service appends
 *   it) so the picker keeps offering the strategy option.
 * - ALL display goes through {@link resolveModePresentation} /
 *   {@link presentModeLabel}; no component may hand-roll an
 *   `mode === "item-level"` ternary.
 *
 * ## Deep links
 * The only item-level deep-link surface is the route param
 * `/operating-modes/item-level` (`operatingModeDetailPath`); there is no
 * `?mode=` query param. The URL stays addressable: initiative-mode-service
 * resolves the sentinel detail client-side (synthesized entry + linked
 * initiatives from the initiative list), and `OperatingModeDetailsPage`
 * renders it as "Workflow Strategy / Member-item workflow" with a notice
 * banner, not as an operating mode.
 *
 * ## Statistics
 * Stats surfaces (usage-by-mode, phase-runs, backlog-sync buckets) keep
 * counting initiatives stored under the wire value — the bucket is
 * relabeled {@link MEMBER_ITEM_STRATEGY_LABEL}, never dropped or silently
 * merged into another mode.
 */

import type { OperatingModeCatalogEntry } from "../types/operating-mode";

/**
 * Wire value the strategy is persisted under on initiatives. Read/write
 * paths keep using this literal value; it is a sentinel, not a mode id.
 */
export const MEMBER_ITEM_STRATEGY_WIRE_VALUE = "item-level";

/** Operator-facing label for the member-item workflow strategy. */
export const MEMBER_ITEM_STRATEGY_LABEL = "Member-item workflow";

/** How a stored initiative `mode` value should be presented. */
export type ModePresentation =
  | {
      kind: "member-item-strategy";
      /** Wire value the strategy is persisted under (permanent policy). */
      wireValue: typeof MEMBER_ITEM_STRATEGY_WIRE_VALUE;
      label: typeof MEMBER_ITEM_STRATEGY_LABEL;
    }
  | { kind: "mode"; mode: string };

/**
 * Map a persisted mode value onto its presentation. `"item-level"` and blank
 * (legacy records predate the mode field) are the member-item strategy;
 * genuine operating modes pass through untouched.
 */
export function resolveModePresentation(mode: string | null | undefined): ModePresentation {
  const normalized = (mode ?? "").trim();
  if (normalized === "" || normalized === MEMBER_ITEM_STRATEGY_WIRE_VALUE) {
    return {
      kind: "member-item-strategy",
      wireValue: MEMBER_ITEM_STRATEGY_WIRE_VALUE,
      label: MEMBER_ITEM_STRATEGY_LABEL,
    };
  }
  return { kind: "mode", mode: normalized };
}

/** True when the stored mode value presents as the member-item strategy. */
export function isMemberItemStrategy(mode: string | null | undefined): boolean {
  return resolveModePresentation(mode).kind === "member-item-strategy";
}

/**
 * Normalize a stored mode value to its wire vocabulary: blank collapses to
 * the legacy strategy wire value, everything else passes through. This is the
 * DATA-side default (what services send/compare on the wire) — display goes
 * through {@link presentModeLabel}.
 */
export function normalizeModeWireValue(mode: string | null | undefined): string {
  const presentation = resolveModePresentation(mode);
  return presentation.kind === "member-item-strategy" ? presentation.wireValue : presentation.mode;
}

/**
 * Display label for a stored mode value. The member-item strategy always
 * renders as {@link MEMBER_ITEM_STRATEGY_LABEL}; genuine modes prefer the
 * server/catalog label and fall back to a humanized form of the mode id.
 */
export function presentModeLabel(mode: string | null | undefined, serverLabel?: string): string {
  const presentation = resolveModePresentation(mode);
  if (presentation.kind === "member-item-strategy") return presentation.label;
  const trimmedLabel = serverLabel?.trim();
  if (trimmedLabel) return trimmedLabel;
  return humanizeModeId(presentation.mode);
}

/** Title-case a kebab/underscore mode id ("holistic-loop" → "Holistic Loop"). */
function humanizeModeId(mode: string): string {
  return mode
    .split(/[-_]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

/** Operator-facing description of the member-item workflow strategy. */
export const MEMBER_ITEM_STRATEGY_DESCRIPTION =
  "Each member item runs through its own operation; the initiative provides scheduling strategy, not a methodology loop.";

/**
 * Synthesized catalog entry for the member-item workflow strategy. The server
 * catalog contains only genuine operating modes (the item-level pseudo-mode
 * was deleted in Phase 9); initiative-mode-service appends this entry so the
 * ModePicker keeps offering the strategy as a switch target. Selecting it
 * submits the wire value through the existing switch-mode mutation.
 */
export function memberItemStrategyCatalogEntry(usageCount = 0): OperatingModeCatalogEntry {
  return {
    mode: MEMBER_ITEM_STRATEGY_WIRE_VALUE,
    label: MEMBER_ITEM_STRATEGY_LABEL,
    description: MEMBER_ITEM_STRATEGY_DESCRIPTION,
    bestFor: [
      "Items are right-sized for one agent run each",
      "Items are loosely coupled and reviewable in isolation",
      "Items are stable — scope won't shift mid-execution",
      "Parallelism across many items is valuable",
    ],
    notFor: [
      "Items are coupled by a shared substrate they all change",
      "The natural unit of validation is the system as a whole",
      "Item shape will shift mid-execution as new ground truth emerges",
      "Intermediate states between items leave the system inconsistent",
    ],
    tradeoffs: [
      "Highest parallelism — many items in flight at once",
      "Bounded blast radius per item",
      "Per-item provenance and review surface",
      "Slowest when items aren't already well-shaped",
    ],
    whenInDoubtPickInstead: undefined,
    usageCount,
    targetKind: "initiative",
    runStrategy: "",
    workspaceTabId: "info",
    capabilities: {
      supportsPhases: false,
      canStartPhases: false,
      canCompleteItems: false,
      canApplyBacklogSyncProposals: false,
      requiresAcceptanceCriteria: false,
      supportsArtifacts: false,
      supportsHandoffs: false,
    },
    // The strategy is what a new initiative starts on (blank mode normalizes
    // to the wire value), so it is the default switch target.
    default: true,
    switchable: true,
    supportsPhases: false,
    phases: [],
    phaseGraph: undefined,
    inputContract: undefined,
  };
}

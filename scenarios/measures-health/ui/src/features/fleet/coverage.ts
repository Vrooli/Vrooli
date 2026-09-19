import { DomainStatus, Tier } from "../../api/fleet";
import { strings } from "../../consts/strings";

/**
 * Presentation metadata for the two load-bearing measures enums. The fleet UI
 * colors and orders strictly off {@link DomainStatus} and {@link Tier} — never
 * off a free-text string — so the visual semantics match the gating semantics:
 * UNCOVERED (the only status that fails a scenario) is destructive red, WAIVED
 * is amber, COVERED is emerald, NOT_EXPECTED is muted. Tier degrades
 * emerald → amber → red as extraction maturity drops.
 */

/** Union of the four domain-status label key paths (preserves `t()` typing). */
type StatusLabelKey = (typeof strings.fleet.status)[keyof typeof strings.fleet.status];
/** Union of the four tier label key paths. */
type TierLabelKey = (typeof strings.fleet.tier)[keyof typeof strings.fleet.tier];

export interface ChipMeta<K extends string> {
  /** Translation key path for the human label. */
  labelKey: K;
  /** Tailwind classes for the chip (border + text + bg). */
  chipClass: string;
  /** Lower sorts first (worst/most-urgent first). */
  order: number;
}

const RED = "border-red-500/40 bg-red-500/10 text-red-300";
const AMBER = "border-amber-500/40 bg-amber-500/10 text-amber-300";
const EMERALD = "border-emerald-500/40 bg-emerald-500/10 text-emerald-300";
const SLATE = "border-slate-500/40 bg-slate-500/10 text-slate-300";

// The neutral fallback for an unknown wire value. Connect decodes an
// unrecognized enum as its raw number (not in the map), so the lookups below
// are `Partial` and fall back here — this is genuinely reachable, not dead.
const NEUTRAL_STATUS: ChipMeta<StatusLabelKey> = {
  labelKey: strings.fleet.status.notExpected,
  chipClass: SLATE,
  order: 4,
};
const NEUTRAL_TIER: ChipMeta<TierLabelKey> = {
  labelKey: strings.fleet.tier.none,
  chipClass: SLATE,
  order: 3,
};

const STATUS_META: Partial<Record<DomainStatus, ChipMeta<StatusLabelKey>>> = {
  [DomainStatus.UNCOVERED]: { labelKey: strings.fleet.status.uncovered, chipClass: RED, order: 0 },
  [DomainStatus.WAIVED]: { labelKey: strings.fleet.status.waived, chipClass: AMBER, order: 1 },
  [DomainStatus.COVERED]: { labelKey: strings.fleet.status.covered, chipClass: EMERALD, order: 2 },
  [DomainStatus.NOT_EXPECTED]: { labelKey: strings.fleet.status.notExpected, chipClass: SLATE, order: 3 },
  [DomainStatus.UNSPECIFIED]: NEUTRAL_STATUS,
};

const TIER_META: Partial<Record<Tier, ChipMeta<TierLabelKey>>> = {
  [Tier.FULL]: { labelKey: strings.fleet.tier.full, chipClass: EMERALD, order: 0 },
  [Tier.PARTIAL]: { labelKey: strings.fleet.tier.partial, chipClass: AMBER, order: 1 },
  [Tier.FALLBACK]: { labelKey: strings.fleet.tier.fallback, chipClass: RED, order: 2 },
  [Tier.UNSPECIFIED]: NEUTRAL_TIER,
};

export function domainStatusMeta(status: DomainStatus): ChipMeta<StatusLabelKey> {
  return STATUS_META[status] ?? NEUTRAL_STATUS;
}

export function tierMeta(tier: Tier): ChipMeta<TierLabelKey> {
  return TIER_META[tier] ?? NEUTRAL_TIER;
}

/** Sort domains by descending urgency (UNCOVERED first), then by name. */
export function compareDomainStatus(a: DomainStatus, b: DomainStatus): number {
  return domainStatusMeta(a).order - domainStatusMeta(b).order;
}

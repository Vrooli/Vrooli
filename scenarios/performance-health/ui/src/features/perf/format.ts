import { CaptureTier } from "../../api/perf";
import { strings } from "../../consts/strings";

/**
 * Shared, presentation-only formatting helpers for performance figures. These
 * never throw on unexpected input — the UI must degrade gracefully when the
 * backend reports a value the frontend hasn't enumerated yet.
 */

type TierLabelKey = "none" | "tier0" | "tier1" | "unknown";

/**
 * Static map of tier key → typed translation key. Written as explicit
 * `strings.tier.*` accessors (not a computed `strings.tier[key]`) so the
 * no-unused-keys lint rule can see every key is referenced.
 */
export const TIER_LABEL_KEY: Record<TierLabelKey, (typeof strings.tier)[keyof typeof strings.tier]> = {
  none: strings.tier.none,
  tier0: strings.tier.tier0,
  tier1: strings.tier.tier1,
  unknown: strings.tier.unknown,
};

/** Map the typed CaptureTier enum to a short, stable label key segment. */
export function tierKey(tier: CaptureTier): TierLabelKey {
  switch (tier) {
    case CaptureTier.CAPTURE_TIER_NONE:
      return "none";
    case CaptureTier.CAPTURE_TIER_0:
      return "tier0";
    case CaptureTier.CAPTURE_TIER_1:
      return "tier1";
    default:
      return "unknown";
  }
}

/**
 * The fleet RPC reports tier as a free-text string ("1", "0", "none"). Normalize
 * to the same key space the typed enum uses so chips render consistently.
 */
export function fleetTierKey(tier: string): TierLabelKey {
  switch (tier.trim().toLowerCase()) {
    case "none":
      return "none";
    case "0":
      return "tier0";
    case "1":
      return "tier1";
    default:
      return "unknown";
  }
}

/** Tailwind chip class for a tier key. Token-driven so it tracks the theme. */
export function tierChipClass(key: TierLabelKey): string {
  switch (key) {
    case "tier1":
      return "border border-app-success/40 bg-app-success/10 text-app-success";
    case "tier0":
      return "border border-app-info/40 bg-app-info/10 text-app-info";
    case "none":
      return "border border-app-border bg-app-surface-muted text-app-muted-foreground";
    default:
      return "border border-app-border bg-app-surface-muted text-app-muted-foreground";
  }
}

/** Format a millisecond duration (bigint from int64 fields) as a human string. */
export function formatMs(ms: bigint | number | undefined): string {
  if (ms === undefined) return "—";
  const n = typeof ms === "bigint" ? Number(ms) : ms;
  if (!Number.isFinite(n) || n <= 0) return "—";
  if (n >= 1000) return `${(n / 1000).toFixed(2)} s`;
  return `${Math.round(n)} ms`;
}

/** Format a millisecond duration that may be a fractional double. */
export function formatMsFloat(ms: number | undefined): string {
  if (ms === undefined || !Number.isFinite(ms) || ms <= 0) return "—";
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`;
  return `${ms.toFixed(1)} ms`;
}

/** Format a byte count (bigint from int64) as KB/MB. */
export function formatBytes(bytes: bigint | number | undefined): string {
  if (bytes === undefined) return "—";
  const n = typeof bytes === "bigint" ? Number(bytes) : bytes;
  if (!Number.isFinite(n) || n <= 0) return "—";
  if (n >= 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(2)} MB`;
  if (n >= 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${n} B`;
}

/** Render an ISO timestamp as a short local datetime; pass-through on failure. */
export function formatTimestamp(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

import { Severity } from "../../api/validation";
import { strings } from "../../consts/strings";

/**
 * Presentation metadata for the normalized {@link Severity} contract. The UI
 * colors and orders strictly off this enum — never off a scanner's native
 * string — so the visual semantics match the gating semantics: ERROR is the
 * only level that fails a scenario (destructive red), WARNING is amber, INFO
 * is muted.
 */
/** Union of the three severity-label key paths (preserves `t()` key typing). */
type SeverityLabelKey = (typeof strings.posture.severity)[keyof typeof strings.posture.severity];

export interface SeverityMeta {
  /** Translation key path for the human label. */
  labelKey: SeverityLabelKey;
  /** Tailwind classes for the severity chip (border + text + bg). */
  chipClass: string;
  /** Lower sorts first — ERROR before WARNING before INFO. */
  order: number;
}

const META: Record<Severity, SeverityMeta> = {
  [Severity.ERROR]: {
    labelKey: strings.posture.severity.error,
    chipClass: "border-red-500/40 bg-red-500/10 text-red-300",
    order: 0,
  },
  [Severity.WARNING]: {
    labelKey: strings.posture.severity.warning,
    chipClass: "border-amber-500/40 bg-amber-500/10 text-amber-300",
    order: 1,
  },
  [Severity.INFO]: {
    labelKey: strings.posture.severity.info,
    chipClass: "border-slate-500/40 bg-slate-500/10 text-slate-300",
    order: 2,
  },
  [Severity.UNSPECIFIED]: {
    labelKey: strings.posture.severity.info,
    chipClass: "border-slate-500/40 bg-slate-500/10 text-slate-300",
    order: 3,
  },
};

export function severityMeta(severity: Severity): SeverityMeta {
  return META[severity] ?? META[Severity.UNSPECIFIED];
}

/** Sort findings by descending severity (ERROR first), then by rule id. */
export function compareSeverity(a: Severity, b: Severity): number {
  return severityMeta(a).order - severityMeta(b).order;
}

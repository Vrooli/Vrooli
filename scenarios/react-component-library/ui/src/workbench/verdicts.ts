import { Check, CircleAlert, Minus, X, type LucideIcon } from "lucide-react";

/**
 * The one verdict vocabulary for evidence surfaces.
 *
 * The studio, the in-context stage and the component drawer render the same
 * four states; keeping the type, icon and catalog key here means a new verdict
 * (e.g. "stale") is added once, not in four renderers. Layout and density stay
 * with each surface — only the meaning is shared.
 */
export type Verdict = "pass" | "warn" | "fail" | "unmeasured";

export const VERDICT_ICON: Record<Verdict, LucideIcon> = {
  pass: Check,
  warn: CircleAlert,
  fail: X,
  unmeasured: Minus,
};

/** Catalog keys for the long label. Compact panels may use their own short form. */
export const VERDICT_KEY: Record<Verdict, string> = {
  pass: "workbench.verdict.pass",
  warn: "workbench.verdict.warn",
  fail: "workbench.verdict.fail",
  unmeasured: "workbench.verdict.unmeasured",
};

/** Never let an unknown string read as a pass. */
export function asVerdict(value: string): Verdict {
  return value === "pass" || value === "warn" || value === "fail" || value === "unmeasured"
    ? value
    : "unmeasured";
}

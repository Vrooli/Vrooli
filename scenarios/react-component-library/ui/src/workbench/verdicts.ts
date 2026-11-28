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

/**
 * Never let an unknown string read as a pass.
 *
 * Evidence arrives from three owners with different past-tense recordings:
 * Experience Manager writes passed/failed/skipped, component test reports write
 * passed/failed, and the live instruments write pass/warn/fail/unmeasured. They
 * are one concept, so this boundary is the only place that translates them.
 */
export function asVerdict(value: string | null | undefined): Verdict {
  switch ((value ?? "").trim().toLowerCase()) {
    case "pass":
    case "passed":
    case "ok":
    case "success":
      return "pass";
    case "warn":
    case "warned":
    case "warning":
      return "warn";
    case "fail":
    case "failed":
    case "error":
      return "fail";
    default:
      return "unmeasured";
  }
}

/** The one status tone for a verdict, so chip colours cannot drift per surface. */
export function verdictTone(verdict: Verdict): "green" | "amber" | "red" | "ice" {
  switch (verdict) {
    case "pass":
      return "green";
    case "fail":
      return "red";
    case "warn":
      return "amber";
    default:
      return "ice";
  }
}

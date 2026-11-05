// One reading of the health probe for every surface that shows it (header
// chip, status dialog, dashboard): an API that answers "degraded" is reachable
// and says why; only a request that fails to reach it is "unreachable".

import { HealthStatus, type HealthResponse } from "../types";
import { presentError } from "./errorPresentation";

export type HealthKind = "healthy" | "degraded" | "unreachable" | "checking";

export interface HealthView {
  kind: HealthKind;
  label: string;
  /** Why the API is degraded or unreachable; empty when healthy. */
  reason: string;
}

export function healthView(health: HealthResponse | null | undefined, error: string | null | undefined, reason = ""): HealthView {
  if (error) {
    const presented = presentError(error);
    if (presented.kind === "unreachable" || presented.kind === "timeout" || presented.kind === "misrouted") {
      return { kind: "unreachable", label: "Unreachable", reason: `${presented.title}. ${presented.message}` };
    }
    return { kind: "degraded", label: "Degraded", reason: presented.message };
  }
  if (!health) return { kind: "checking", label: "Checking", reason: "" };
  if (health.status === HealthStatus.HEALTHY) return { kind: "healthy", label: "Healthy", reason: "" };
  return { kind: "degraded", label: "Degraded", reason: reason || "Agent Manager answers but reports reduced capability." };
}

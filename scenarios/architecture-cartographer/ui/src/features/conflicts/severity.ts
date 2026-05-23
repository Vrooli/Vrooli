import { Severity } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

import type { SeverityLevel } from "../../components/SeverityBadge";

/**
 * Map proto Severity → UI SeverityLevel. UNSPECIFIED collapses to "info" so
 * a malformed server response never leaves the UI without a renderable level.
 */
export function severityToLevel(severity: Severity): SeverityLevel {
  switch (severity) {
    case Severity.BLOCKER:
      return "critical";
    case Severity.ERROR:
      return "high";
    case Severity.WARN:
      return "medium";
    case Severity.INFO:
      return "info";
    case Severity.UNSPECIFIED:
      return "info";
    default:
      return "info";
  }
}

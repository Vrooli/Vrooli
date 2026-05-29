import type { SeverityLevel } from "../../components/SeverityBadge";

/**
 * Map a migration finding's severity TOKEN → UI SeverityLevel. Tracked
 * findings carry the cartographer's lower-case severity vocabulary
 * (blocker/error/warn/info), not the proto enum, so this keys on strings.
 * Anything unrecognized collapses to "info".
 */
export function severityTokenToLevel(token: string): SeverityLevel {
  switch (token) {
    case "blocker":
      return "critical";
    case "error":
      return "high";
    case "warn":
      return "medium";
    case "info":
      return "info";
    default:
      return "info";
  }
}

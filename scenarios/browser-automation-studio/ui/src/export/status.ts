/**
 * Status utilities for the replay export page.
 *
 * Utilities for normalizing and generating status messages.
 */

/**
 * Normalizes a status string to lowercase.
 */
export const normalizeStatus = (status?: string | null): string => {
  if (!status) {
    return "";
  }
  return status.trim().toLowerCase();
};

/**
 * Generates a default status message based on the status.
 */
export const defaultStatusMessage = (status: string): string => {
  switch (status) {
    case "ready":
      return "Replay export is ready";
    case "pending":
      return "Replay export pending – timeline frames not captured yet";
    case "unavailable":
      return "Replay export unavailable – execution did not capture any timeline frames";
    case "error":
      return "Replay export failed";
    default:
      return "Replay export unavailable";
  }
};

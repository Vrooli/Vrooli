import { ConnectError, Code } from "@connectrpc/connect";

/** Plain words for a failed usage mode call; a refused caller is told why. */
export function startErrorText(error: unknown, action = "start a usage mode"): string {
  if (error instanceof ConnectError) {
    if (error.code === Code.PermissionDenied) return `Only a verified operator can ${action}.`;
    return error.rawMessage || error.message;
  }
  return error instanceof Error ? error.message : "The request could not be completed.";
}

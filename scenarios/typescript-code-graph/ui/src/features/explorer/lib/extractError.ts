import { Code, ConnectError } from "@connectrpc/connect";

/**
 * True when an Extract failure is the typed `workspace_unsupported` outcome —
 * the API emits Connect `CodeUnimplemented` for project roots that contain a
 * `pnpm-workspace.yaml` (pnpm/yarn workspaces are out of scope for v1, see
 * typescript-code-graph OT-P2-005). This is *designed* behavior, not a bug, so
 * the UI presents it as information rather than a generic error.
 *
 * Unimplemented is unique to this case in TypeScriptCodeGraphService, so the
 * Connect code alone classifies it; the message guard is belt-and-suspenders.
 */
export function isWorkspaceUnsupported(err: unknown): boolean {
  if (!(err instanceof ConnectError)) return false;
  if (err.code !== Code.Unimplemented) return false;
  return true;
}

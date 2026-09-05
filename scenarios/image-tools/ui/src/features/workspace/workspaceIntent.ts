import type { WorkspaceMode } from "./useWorkspace";

/**
 * One-shot handoff from Home / Library into the Workspace.
 *
 * Home routes the user to `/workspace` pre-set to a task, and Library "reopen"
 * loads a stored output back into the canvas — both need to carry a desired
 * mode + operation (and optionally a starting image) across a route change
 * without a global document store (the document stays Workspace-local per the
 * design's session-only persistence decision).
 *
 * Why token-keyed instead of clear-on-read: React StrictMode double-invokes
 * mount effects in dev (mount → unmount → mount), so a plain consume-and-clear
 * would lose the intent on the second mount. Each `setWorkspaceIntent` stamps a
 * monotonic token; `takeWorkspaceIntent` returns the pending intent only if its
 * token hasn't been applied yet. So a StrictMode re-mount (or a later manual
 * `/workspace` visit) is a no-op, while every fresh Home/Library navigation
 * applies exactly once.
 */
export interface WorkspaceIntent {
  /** A starting image to load as the base document. */
  file?: File;
  /** The mode to open in (Edit / Enhance / Create / Analyze). */
  mode?: WorkspaceMode;
  /** A deterministic op to pre-select (Edit mode only). */
  operation?: string;
}

let pending: (WorkspaceIntent & { token: number }) | null = null;
let nextToken = 0;
let appliedToken = 0;

/** Stage an intent and navigate to `/workspace`; the Workspace applies it once. */
export function setWorkspaceIntent(intent: WorkspaceIntent): void {
  nextToken += 1;
  pending = { ...intent, token: nextToken };
}

/**
 * Return the pending intent exactly once. Returns `null` if nothing is pending
 * or the current intent was already applied (StrictMode re-mount / manual
 * revisit), so the Workspace never re-applies a stale handoff.
 */
export function takeWorkspaceIntent(): WorkspaceIntent | null {
  if (!pending || pending.token === appliedToken) {
    return null;
  }
  appliedToken = pending.token;
  return { file: pending.file, mode: pending.mode, operation: pending.operation };
}

/** Test helper — reset module state between cases. */
export function resetWorkspaceIntent(): void {
  pending = null;
  nextToken = 0;
  appliedToken = 0;
}

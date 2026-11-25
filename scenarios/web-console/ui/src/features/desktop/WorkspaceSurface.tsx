import { useCallback, useEffect, useRef, type FocusEvent, type KeyboardEvent, type ReactNode } from "react";
import { WorkspaceHeader } from "@vrooli/react-component-library/WorkspaceHeader/1";
import type { DesktopReadiness, DesktopSession, ReadinessState } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";

import type { RemoteSurfaceState } from "./remoteSurface";
import { workspaceSurfacePresentationAttributes, type WorkspaceSurfaceOwner, type WorkspaceSurfacePaneState } from "./workspaceSurfaceContract";

/** One operator-visible state vocabulary for file, browser and device panes. */
export type WorkspacePaneState = WorkspaceSurfacePaneState;

const stateLabels: Record<WorkspacePaneState, string> = {
  loading: "Loading",
  ready: "Ready",
  observe: "Observe",
  controlling: "Controlling",
  "permission-blocked": "Blocked",
  offline: "Offline",
  revoked: "Revoked",
  choosing: "Choose target",
  unsupported: "Unsupported",
  reconnecting: "Reconnecting",
  failed: "Failed",
  closed: "Closed",
};

const stateAnnouncements: Record<WorkspacePaneState, string> = {
  loading: "Surface is loading.",
  ready: "Surface is ready.",
  observe: "Observe mode. Changes are not being sent to the remote surface.",
  controlling: "Control is active for this surface.",
  "permission-blocked": "Access is blocked by this surface's permission.",
  offline: "The surface is offline.",
  revoked: "Control was revoked for this surface. Close it before reconnecting.",
  choosing: "Choose an item to continue. The preview is waiting for a selection.",
  unsupported: "This preview format is not supported.",
  reconnecting: "Surface is reconnecting.",
  failed: "Surface is unavailable.",
  closed: "Surface is closed.",
};

export function workspacePaneStateLabel(state: WorkspacePaneState): string {
  return stateLabels[state];
}

export function workspacePaneAnnouncement(state: WorkspacePaneState): string {
  return stateAnnouncements[state];
}

export function projectRemoteSurfaceState(state: RemoteSurfaceState, authorized = true): WorkspacePaneState {
  if (!authorized || state === "blocked") return "permission-blocked";
  if (state === "offline") return "offline";
  if (state === "revoked") return "revoked";
  switch (state) {
    case "ready": return "ready";
    case "connecting": return "loading";
    case "disconnected": return "reconnecting";
    case "viewer": return "observe";
    case "controller": return "controlling";
    case "failed": return "failed";
  }
}

export type FilePreviewStateName = "idle" | "resolving" | "loadingText" | "loadingListing" | "ready" | "choosing" | "unsupported" | "error";
export function projectFilePreviewState(open: boolean, status: FilePreviewStateName): WorkspacePaneState {
  if (!open) return "closed";
  if (status === "idle" || status === "resolving" || status === "loadingText" || status === "loadingListing") return "loading";
  if (status === "ready") return "ready";
  if (status === "choosing") return "choosing";
  if (status === "unsupported") return "unsupported";
  return "failed";
}

export type DesktopSourceState = "offline" | "blocked" | "ready" | "connecting" | "disconnected" | "viewer" | "controller" | "revoked" | "failed";
const OFFLINE = 1 as ReadinessState;
const READY = 6 as ReadinessState;
type ReadinessLike = Pick<DesktopReadiness, "state"> & Partial<Pick<DesktopReadiness, "reasonCode" | "recovery">>;
type SessionLike = Pick<DesktopSession, "connectionState" | "controller" | "leaseId" | "leaseEpoch"> & Partial<Pick<DesktopSession, "selectedDisplayId" | "codec">>;

export function desktopOwnerState(readiness?: ReadinessLike, session?: SessionLike): DesktopSourceState {
  if (session?.connectionState === "revoked") return "revoked";
  if (session?.connectionState === "failed") return "failed";
  if (session?.connectionState === "disconnected") return "disconnected";
  if (session?.connectionState === "connecting") return "connecting";
  if (session?.connectionState === "connected") return session.controller ? "controller" : "viewer";
  if (!readiness || readiness.state === OFFLINE) return "offline";
  if (readiness.state !== READY) return "blocked";
  return "ready";
}

export function projectDesktopState(state: DesktopSourceState, hasSession: boolean): WorkspacePaneState {
  if (state === "offline") return "offline";
  if (state === "blocked") return "permission-blocked";
  if (state === "revoked") return "revoked";
  if (state === "failed") return "failed";
  if (state === "disconnected") return "reconnecting";
  if (state === "connecting") return hasSession ? "reconnecting" : "loading";
  if (state === "viewer") return "observe";
  if (state === "controller") return "controlling";
  return "ready";
}

export function desktopPaneState(readiness?: ReadinessLike, session?: SessionLike): WorkspacePaneState {
  return projectDesktopState(desktopOwnerState(readiness, session), Boolean(session));
}

export function canControlDesktop(readiness?: ReadinessLike, session?: SessionLike): boolean {
  return desktopPaneState(readiness, session) === "controlling" && Boolean(session?.leaseId) && (session?.leaseEpoch ?? 0n) > 0n;
}

export function isWorkspaceNavigationKey(event: Pick<KeyboardEvent, "key">): boolean {
  return event.key === "Tab";
}

export function workspaceSurfaceTouchAction(controller: boolean): "none" | "auto" {
  return controller ? "none" : "auto";
}

export function WorkspacePaneStateStatus({ state, label }: { state: WorkspacePaneState; label: string }) {
  return <span data-testid="workspace-pane-state" data-pane-state={state} className="inline-flex min-h-9 shrink-0 items-center rounded-full border border-wc-default px-2 text-[11px] font-medium md:px-2.5" aria-label={`${label}: ${workspacePaneStateLabel(state)}`}>
    <span aria-hidden="true" className="h-2 w-2 rounded-full bg-current md:hidden" />
    <span aria-hidden="true" className="hidden md:inline">{workspacePaneStateLabel(state)}</span>
    <span className="sr-only" role="status" aria-live="polite">{label}: {workspacePaneAnnouncement(state)}</span>
  </span>;
}

/** Shared toolbar semantics: native tab order, Tab exits, arrows move between actions. */
export function WorkspacePaneToolbar({ label, state, children, nowrap = false }: { label: string; state: WorkspacePaneState; children: ReactNode; nowrap?: boolean }) {
  const toolbarRef = useRef<HTMLDivElement>(null);
  const controls = useCallback((toolbar: HTMLDivElement) => Array.from(toolbar.querySelectorAll<HTMLElement>("button:not(:disabled), [role=button]:not([aria-disabled=true])")), []);
  const setRovingTabStop = useCallback((toolbar: HTMLDivElement, active?: HTMLElement) => {
    const available = controls(toolbar);
    const tabStop = active && available.includes(active) ? active : available[0];
    available.forEach((control) => { control.tabIndex = control === tabStop ? 0 : -1; });
  }, [controls]);
  useEffect(() => {
    const toolbar = toolbarRef.current;
    if (toolbar) setRovingTabStop(toolbar, toolbar.contains(document.activeElement) ? document.activeElement as HTMLElement : undefined);
  }, [children, setRovingTabStop]);
  const handleKeyDown = useCallback((event: KeyboardEvent<HTMLDivElement>) => {
    if (isWorkspaceNavigationKey(event)) return;
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
    const buttons = controls(event.currentTarget);
    if (!buttons.length) return;
    const current = buttons.indexOf(document.activeElement as HTMLElement);
    const next = event.key === "Home" ? 0 : event.key === "End" ? buttons.length - 1 :
      (current + (event.key === "ArrowRight" ? 1 : -1) + buttons.length) % buttons.length;
    event.preventDefault();
    buttons[next]?.focus();
  }, [controls]);
  const handleFocusCapture = useCallback((event: FocusEvent<HTMLDivElement>) => {
    if ((event.target as HTMLElement).matches("button:not(:disabled), [role=button]:not([aria-disabled=true])")) {
      setRovingTabStop(event.currentTarget, event.target as HTMLElement);
    }
  }, [setRovingTabStop]);
  return <div ref={toolbarRef} role="toolbar" aria-label={label} aria-orientation="horizontal" data-pane-state={state} onKeyDown={handleKeyDown} onFocusCapture={handleFocusCapture} className={nowrap ? "flex min-h-11 flex-nowrap items-center gap-2 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden" : "flex min-h-11 flex-wrap items-center gap-2"}>
    {children}
  </div>;
}

/** Shared RCL header, state announcement and bounded workspace layout for remote owners. */
export function WorkspaceSurfacePane({ label, title, state, summary, stats, statsLabel, owner, canControl = false, onFocusChange, children }: {
  label: string;
  title: string;
  state: WorkspacePaneState;
  summary?: string;
  stats?: string;
  statsLabel?: string;
  owner: Exclude<WorkspaceSurfaceOwner, "file-preview">;
  canControl?: boolean;
  onFocusChange?: (focused: boolean) => void;
  children: ReactNode;
}) {
  const contractAttributes = workspaceSurfacePresentationAttributes(owner, state, false, canControl);
  return <section {...contractAttributes} aria-label={label} data-testid="workspace-surface-pane" data-state={state} data-surface-focused={state === "failed" || state === "revoked" || state === "closed" ? "false" : undefined} tabIndex={-1} onFocusCapture={(event) => { event.currentTarget.dataset.surfaceFocused = "true"; onFocusChange?.(true); }} onBlurCapture={(event) => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) { event.currentTarget.dataset.surfaceFocused = "false"; onFocusChange?.(false); } }} className="flex min-h-64 min-w-0 w-full flex-col gap-3 rounded-lg border border-app-border bg-app-surface p-3">
    <WorkspaceHeader as="div" title={title} primaryAction={<WorkspacePaneStateStatus state={state} label={title} />} />
    {summary && <p className="min-w-0 break-words text-sm leading-relaxed text-app-muted">{summary}</p>}
    {stats && <p aria-label={statsLabel ?? `${title} transport statistics`} className="min-w-0 break-words border-y border-app-border py-1 text-xs text-app-muted">{stats}</p>}
    {children}
  </section>;
}

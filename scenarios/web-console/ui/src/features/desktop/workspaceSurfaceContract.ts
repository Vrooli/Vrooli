/**
 * Presentation handoff contract for Web Console surfaces. It records only
 * workspace facts; the named domain owner remains authoritative for sessions,
 * data, permissions, and controller leases.
 */
export type WorkspaceSurfaceOwner = "bas" | "device-control" | "file-preview";
export type WorkspaceSurfaceAccess = "read-only" | "observe" | "control";
export type WorkspaceSurfaceLifecycle = "closed" | "loading" | "ready" | "reconnecting" | "failed" | "revoked";
export type WorkspaceSurfaceOperation = "open" | "takeover" | "handback" | "reconnect" | "close";
export type WorkspaceSurfacePresentationLifecycle = WorkspaceSurfaceLifecycle | "blocked" | "offline" | "choosing" | "unsupported";
export type WorkspaceSurfacePaneState =
  | "loading" | "ready" | "observe" | "controlling" | "permission-blocked" | "offline"
  | "revoked" | "choosing" | "unsupported" | "reconnecting" | "failed" | "closed";

export interface WorkspaceSurfaceContractState {
  readonly id: string;
  readonly owner: WorkspaceSurfaceOwner;
  readonly focused: boolean;
  readonly lifecycle: WorkspaceSurfaceLifecycle;
  readonly access: WorkspaceSurfaceAccess;
  /** A presence bit only. Lease identity/receipt stays in the domain owner. */
  readonly leaseHeld: boolean;
  readonly canControl: boolean;
  readonly pending?: WorkspaceSurfaceOperation;
  readonly error?: string;
}

export function workspaceSurfacePresentationAttributes(
  owner: WorkspaceSurfaceOwner,
  paneState: WorkspaceSurfacePaneState,
  focused: boolean,
  canControl = false,
): Record<string, string> {
  const isFile = owner === "file-preview";
  const access = !isFile && paneState === "controlling" ? "control" : !isFile && paneState === "observe" ? "observe" : "read-only";
  const lifecycle: WorkspaceSurfacePresentationLifecycle = paneState === "loading" ? "loading"
    : paneState === "reconnecting" ? "reconnecting"
      : paneState === "failed" ? "failed"
        : paneState === "revoked" ? "revoked"
          : paneState === "closed" ? "closed"
            : paneState === "permission-blocked" ? "blocked"
              : paneState === "offline" ? "offline"
                : paneState === "choosing" ? "choosing"
                  : paneState === "unsupported" ? "unsupported"
                    : "ready";
  return {
    "data-surface-owner": owner,
    "data-surface-lifecycle": lifecycle,
    "data-surface-access": access,
    "data-surface-lease-held": String(access === "control"),
    "data-surface-can-control": String(!isFile && canControl),
    "data-surface-focused": String(focused),
  };
}

export type WorkspaceSurfaceContractEvent =
  | { type: "focus"; focused: boolean }
  | { type: "request"; operation: WorkspaceSurfaceOperation }
  | { type: "owner-opened"; owner: WorkspaceSurfaceOwner; canControl: boolean; leaseHeld?: boolean }
  | { type: "owner-control-granted"; owner: WorkspaceSurfaceOwner; leaseHeld: boolean }
  | { type: "owner-handback-complete"; owner: WorkspaceSurfaceOwner; leaseHeld: boolean }
  | { type: "owner-reconnected"; owner: WorkspaceSurfaceOwner; access: WorkspaceSurfaceAccess; leaseHeld: boolean }
  | { type: "owner-closed"; owner: WorkspaceSurfaceOwner }
  | { type: "owner-revoked"; owner: WorkspaceSurfaceOwner }
  | { type: "owner-operation-failed"; owner: WorkspaceSurfaceOwner; operation: WorkspaceSurfaceOperation; message: string }
  | { type: "owner-failed"; owner: WorkspaceSurfaceOwner; message: string };

export function createWorkspaceSurfaceContract(
  owner: WorkspaceSurfaceOwner,
  id: string,
  canControl = false,
): WorkspaceSurfaceContractState {
  return { id, owner, focused: false, lifecycle: "closed", access: "read-only", leaseHeld: false, canControl };
}

/**
 * Reduce explicit UI intent and owner acknowledgements into one shared
 * handoff vocabulary. A request changes only presentation state; access and
 * lease state change only after an event from the fixed domain owner.
 */
export function transitionWorkspaceSurface(
  state: WorkspaceSurfaceContractState,
  event: WorkspaceSurfaceContractEvent,
): WorkspaceSurfaceContractState {
  if (event.type === "focus") return { ...state, focused: event.focused };
  if (event.type === "request") {
    // Cleanup has priority over any in-flight open/takeover/reconnect request;
    // duplicate cleanup stays idempotent while close is pending.
    if (event.operation === "close") {
      return state.lifecycle === "closed" || state.pending === "close" ? state : { ...state, pending: "close" };
    }
    // A handback can cancel a takeover before its owner acknowledgement arrives.
    if (event.operation === "handback" && state.pending === "takeover") return { ...state, pending: "handback" };
    if (state.pending) return state;
    switch (event.operation) {
      case "open":
        return state.lifecycle === "closed" || state.lifecycle === "failed"
          ? { ...state, lifecycle: "loading", pending: "open", error: undefined }
          : state;
      case "takeover":
        return state.lifecycle === "ready" && state.access === "observe" && state.canControl
          ? { ...state, pending: "takeover", error: undefined }
          : state;
      case "handback":
        return state.lifecycle === "ready" && state.access === "control" && state.leaseHeld
          ? { ...state, pending: "handback" }
          : state;
      case "reconnect":
        return state.lifecycle !== "closed" && state.lifecycle !== "revoked"
          ? { ...state, lifecycle: "reconnecting", pending: "reconnect", error: undefined }
          : state;
    }
  }
  // A workspace adapter cannot acknowledge another owner's operation.
  if (event.owner !== state.owner) return state;
  switch (event.type) {
    case "owner-opened":
      if (state.owner === "file-preview" && event.leaseHeld) return state;
      return { ...state, lifecycle: "ready", access: event.leaseHeld ? "control" : state.owner === "file-preview" ? "read-only" : "observe", leaseHeld: Boolean(event.leaseHeld), canControl: state.owner !== "file-preview" && event.canControl, pending: state.pending === "close" ? "close" : undefined, error: undefined };
    case "owner-control-granted":
      if (state.owner === "file-preview" || !event.leaseHeld || !state.canControl || state.pending !== "takeover") return state;
      return { ...state, access: "control", leaseHeld: true, pending: undefined };
    case "owner-handback-complete":
      if (event.leaseHeld || (state.pending !== "handback" && state.pending !== "close" && state.lifecycle !== "failed")) return state;
      return { ...state, lifecycle: state.lifecycle === "failed" ? "ready" : state.lifecycle, access: state.owner === "file-preview" ? "read-only" : "observe", leaseHeld: false, pending: state.pending === "close" ? "close" : undefined, error: undefined };
    case "owner-reconnected":
      if (state.pending !== "reconnect") return state;
      if (state.owner === "file-preview" && (event.access === "control" || event.leaseHeld)) return state;
      return { ...state, lifecycle: "ready", access: event.access, leaseHeld: event.leaseHeld, pending: undefined, error: undefined };
    case "owner-closed":
      return { ...state, focused: false, lifecycle: "closed", access: "read-only", leaseHeld: false, pending: undefined, error: undefined };
    case "owner-revoked":
      return { ...state, focused: false, lifecycle: "revoked", access: "observe", leaseHeld: false, pending: undefined };
    case "owner-operation-failed":
      if (state.pending !== event.operation) return state;
      if (event.operation === "handback") return { ...state, focused: false, lifecycle: "failed", access: "read-only", leaseHeld: false, pending: state.pending === "close" ? "close" : undefined, error: event.message };
      return { ...state, pending: event.operation === "close" ? "close" : undefined, error: event.message };
    case "owner-failed":
      return { ...state, focused: false, lifecycle: "failed", access: "read-only", leaseHeld: false, pending: state.pending === "close" || state.pending === "handback" ? state.pending : undefined, error: event.message };
    default:
      return state;
  }
}

/** Small owner-local adapter for reporting the shared contract from a pane. */
export class WorkspaceSurfaceHandoff {
  private current: WorkspaceSurfaceContractState;

  constructor(owner: WorkspaceSurfaceOwner, id: string, canControl = false) {
    this.current = createWorkspaceSurfaceContract(owner, id, canControl);
  }

  get state(): WorkspaceSurfaceContractState { return this.current; }

  dispatch(event: WorkspaceSurfaceContractEvent): WorkspaceSurfaceContractState {
    this.current = transitionWorkspaceSurface(this.current, event);
    return this.current;
  }
}

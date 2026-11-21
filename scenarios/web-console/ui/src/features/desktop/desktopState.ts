import type { DesktopReadiness, DesktopSession, ReadinessState } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";

// Generated protobuf enums are type-only in the browser build. Keep the wire
// values local so this projection has no runtime enum dependency.
const OFFLINE = 1 as ReadinessState;
const READY = 6 as ReadinessState;

export type DesktopViewState = "offline" | "blocked" | "ready" | "connecting" | "disconnected" | "viewer" | "controller" | "revoked" | "failed";
type ReadinessLike = Pick<DesktopReadiness, "state"> & Partial<Pick<DesktopReadiness, "reasonCode" | "recovery">>;
type SessionLike = Pick<DesktopSession, "connectionState" | "controller" | "leaseId" | "leaseEpoch"> & Partial<Pick<DesktopSession, "selectedDisplayId" | "codec">>;

export function desktopViewState(readiness?: ReadinessLike, session?: SessionLike): DesktopViewState {
  if (session?.connectionState === "revoked") return "revoked";
  if (session?.connectionState === "failed") return "failed";
  if (session?.connectionState === "disconnected") return "disconnected";
  if (session?.connectionState === "connecting") return "connecting";
  if (session?.connectionState === "connected") return session.controller ? "controller" : "viewer";
  if (!readiness || readiness.state === OFFLINE) return "offline";
  if (readiness.state !== READY) return "blocked";
  return "ready";
}

export function canMutateDesktop(readiness?: ReadinessLike, session?: SessionLike): boolean {
  return desktopViewState(readiness, session) === "controller" && Boolean(session?.leaseId) && (session?.leaseEpoch ?? 0n) > 0n;
}

import { OwnerOpenRequestSchema, type OwnerOpenRequest } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { fromJson, toJson, type JsonValue } from "@bufbuild/protobuf";
import { SessionRefSchema, type SessionRef } from "@vrooli/proto-types/common/v1/surface_pb";

export const desktopRecoveryKey = "portal.desktop-recovery.v1";
export type RecoveryActor = { id: string; realm: string };
export type RecoveryMarker = { actor: RecoveryActor; unknownOpen: boolean; sessions: SessionRef[]; pendingOpen?: OwnerOpenRequest };
const validText = (value: unknown): value is string => typeof value === "string" && value.length > 0 && value.length <= 256;
export function readDesktopRecovery(): RecoveryMarker | undefined {
  const raw = localStorage.getItem(desktopRecoveryKey);
  if (raw === null) return undefined;
  if (raw.length > 4 * 1024 * 1024) throw new Error("Desktop recovery marker exceeds bound");
  const value: unknown = JSON.parse(raw);
  const record = (item: unknown): item is Record<string, unknown> => typeof item === "object" && item !== null && !Array.isArray(item);
  if (!record(value) || (value.version !== 1 && value.version !== 2) || !record(value.actor) || !validText(value.actor.id) || !validText(value.actor.realm) || typeof value.unknownOpen !== "boolean" || !Array.isArray(value.sessions) || value.sessions.length > 2500) throw new Error("Invalid desktop recovery marker");
  const sessions = value.sessions.map((item: unknown) => {
    const ref = fromJson(SessionRefSchema, item as JsonValue);
    const surface = ref.surface;
    const target = surface?.target;
    if (!surface || !target || !validText(ref.sessionId) || !validText(ref.desktopSessionId) || !validText(surface.surfaceId) || !validText(surface.ownerScenario) || !validText(target.resourceId) || !validText(target.ownerScenario)) throw new Error("Invalid recovery reference");
    return ref;
  });
  let pendingOpen: OwnerOpenRequest | undefined;
  if (value.version === 2 && value.pendingOpen !== undefined) {
    pendingOpen = fromJson(OwnerOpenRequestSchema, value.pendingOpen as JsonValue);
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(pendingOpen.requestId) || pendingOpen.requestId === "00000000-0000-0000-0000-000000000000" || !pendingOpen.surface?.target || !validText(pendingOpen.surface.surfaceId) || !validText(pendingOpen.surface.ownerScenario) || !validText(pendingOpen.surface.target.resourceId) || !validText(pendingOpen.surface.target.ownerScenario) || pendingOpen.ttlSeconds < 1 || pendingOpen.ttlSeconds > 600) throw new Error("Invalid pending desktop request");
  }
  return { pendingOpen, actor: { id: value.actor.id, realm: value.actor.realm }, unknownOpen: value.unknownOpen, sessions };
}
// Persist only identity hints. These never contain tokens or restore authority.
export function writeDesktopRecovery(marker: RecoveryMarker | undefined) {
  if (!marker) { localStorage.removeItem(desktopRecoveryKey); return; }
  const raw = JSON.stringify({ version: 2, pendingOpen: marker.pendingOpen ? toJson(OwnerOpenRequestSchema, marker.pendingOpen) : undefined, actor: { id: marker.actor.id, realm: marker.actor.realm }, unknownOpen: marker.unknownOpen, sessions: marker.sessions.map(ref => toJson(SessionRefSchema, ref)) });
  if (raw.length > 4 * 1024 * 1024 || marker.sessions.length > 2500) throw new Error("Desktop recovery marker exceeds bound");
  localStorage.setItem(desktopRecoveryKey, raw);
}

import { fromJson, toJson, type JsonValue } from "@bufbuild/protobuf";
import {
  DeviceSchema,
  type Device,
} from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";

/**
 * Persisted session shape. The hub has two credentials (see `api/transport.ts`):
 * a device token (this browser's trust-group membership) and an owner JWT (for
 * owner-gated device management). The paired `device` is kept so the UI can show
 * "this device" without a round-trip. We persist the device as proto-JSON so the
 * stored shape can't drift from the wire contract.
 */
export interface SessionState {
  deviceToken: string | null;
  device: Device | null;
  ownerToken: string | null;
}

export interface SessionCredentials {
  deviceToken: string | null;
  ownerToken: string | null;
}

const STORAGE_KEY = "device-sync-hub.session";

interface StoredSession {
  deviceToken?: string | null;
  device?: JsonValue | null;
  ownerToken?: string | null;
}

export const emptySession: SessionState = {
  deviceToken: null,
  device: null,
  ownerToken: null,
};

const safeStorage = (): Storage | null => {
  try {
    return typeof window !== "undefined" ? window.localStorage : null;
  } catch {
    // Access can throw in privacy modes / sandboxed iframes.
    return null;
  }
};

/**
 * Read the full persisted session. Tolerant of malformed/legacy payloads:
 * anything unparseable resolves to the empty session rather than throwing,
 * so a corrupt entry can never wedge the app at boot.
 */
export function loadSession(): SessionState {
  const storage = safeStorage();
  if (!storage) return emptySession;
  const raw = storage.getItem(STORAGE_KEY);
  if (!raw) return emptySession;
  try {
    const parsed = JSON.parse(raw) as StoredSession;
    const device =
      parsed.device != null
        ? fromJson(DeviceSchema, parsed.device, { ignoreUnknownFields: true })
        : null;
    return {
      deviceToken: parsed.deviceToken ?? null,
      device,
      ownerToken: parsed.ownerToken ?? null,
    };
  } catch {
    return emptySession;
  }
}

export function saveSession(state: SessionState): void {
  const storage = safeStorage();
  if (!storage) return;
  const stored: StoredSession = {
    deviceToken: state.deviceToken,
    device: state.device ? toJson(DeviceSchema, state.device) : null,
    ownerToken: state.ownerToken,
  };
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(stored));
  } catch {
    // Quota / disabled storage — non-fatal; the in-memory context still holds.
  }
}

export function clearSession(): void {
  const storage = safeStorage();
  if (!storage) return;
  try {
    storage.removeItem(STORAGE_KEY);
  } catch {
    // ignore
  }
}

/**
 * Read just the credentials, fresh from storage. `authedFetch` calls this on
 * every request so a pairing or sign-in mid-session is picked up without
 * rebuilding the Connect transport.
 */
export function readSessionCredentials(): SessionCredentials {
  const { deviceToken, ownerToken } = loadSession();
  return { deviceToken, ownerToken };
}

import { permissionRegistryClient } from "./permissionRegistry";
import { readInstallPostureFromRegistry, writeUpdateControlThroughRegistry, type PermissionRegistryClient, type UpdateControlMode } from "./installPostureContract";

export {
  type InstallPostureSnapshot,
  type SessionMode,
  type TrustPosture,
  type UpdateControlMode,
} from "./installPostureContract";

export function readInstallPosture() {
  return readInstallPostureFromRegistry(permissionRegistryClient as unknown as PermissionRegistryClient);
}

export function writeUpdateControl(value: UpdateControlMode, expectedRevision: string, idempotencyKey: string) {
  return writeUpdateControlThroughRegistry(permissionRegistryClient as unknown as PermissionRegistryClient, value, expectedRevision, idempotencyKey);
}

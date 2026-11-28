import { permissionRegistryClient } from "./permissionRegistry";
import { readApprovedDependencySnapshot, type PermissionRegistryReader } from "./approvedDependenciesContract";

export function listApprovedDependencies() {
  return readApprovedDependencySnapshot(permissionRegistryClient as unknown as PermissionRegistryReader);
}

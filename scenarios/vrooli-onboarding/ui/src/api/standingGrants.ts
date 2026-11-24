import { permissionRegistryClient } from "./permissionRegistry";
import { readStandingGrantProjection, type PermissionRegistryReader } from "./standingGrantsContract";
import type { AgentManagerPermissionSnapshot } from "./standingGrantsContract";

export type { AgentManagerPermissionSnapshot, ScopeDeclaration, StandingGrantMetadata } from "./standingGrantsContract";

// Listing and inspection use the same typed onboarding permission registry as
// the API and CLI; this UI has no issuance or revocation control.
export async function listStandingGrantMetadata(): Promise<AgentManagerPermissionSnapshot> {
  return readStandingGrantProjection(permissionRegistryClient as unknown as PermissionRegistryReader);
}

import { permissionRegistryClient } from "./permissionRegistry";
import { listCredentialGrantMetadata as listThroughRegistry, type CredentialGrantRegistryReader } from "./credentialGrantsContract";
import {
  listCredentialGrantActions as listActionsThroughRegistry,
  writeCredentialGrantAction as writeActionThroughRegistry,
  type CredentialGrantAction,
  type CredentialGrantActionReceipt,
  type CredentialGrantActionRegistry,
} from "./credentialGrantActionsContract";

export type { CredentialGrantMetadata } from "./credentialGrantsContract";
export type { CredentialGrantAction, CredentialGrantActionReceipt } from "./credentialGrantActionsContract";

// Credential grant metadata and write-only actions use separate registry
// adapters; no sealed credential value is accepted or returned by this UI.
export async function listCredentialGrantMetadata() {
  return listThroughRegistry(permissionRegistryClient as unknown as CredentialGrantRegistryReader);
}

// Actions use only write-only descriptors and never enter the metadata reader.
export async function listCredentialGrantActions() {
  return listActionsThroughRegistry(permissionRegistryClient as unknown as CredentialGrantActionRegistry);
}

export async function writeCredentialGrantAction(action: CredentialGrantAction, idempotencyKey: string, value: Record<string, unknown>) {
  return writeActionThroughRegistry(permissionRegistryClient as unknown as CredentialGrantActionRegistry, action, idempotencyKey, value);
}

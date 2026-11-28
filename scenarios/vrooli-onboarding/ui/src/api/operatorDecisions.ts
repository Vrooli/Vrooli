import { permissionRegistryClient } from "./permissionRegistry";
import { listPendingOperatorDecisions as listThroughRegistry, type DecisionPermissionRegistry } from "./operatorDecisionsContract";
import type { PendingDecisionReference } from "./operatorDecisionsContract";

export type { PendingDecisionReference } from "./operatorDecisionsContract";

// Both inventory and reads go through the shared owner permission registry.
// The value contract is an allowlisted reference and contains no decision text.
export async function listPendingOperatorDecisions(): Promise<PendingDecisionReference[]> {
  return listThroughRegistry(permissionRegistryClient as unknown as DecisionPermissionRegistry);
}

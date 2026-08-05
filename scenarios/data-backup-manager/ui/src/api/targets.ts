/**
 * Targets domain — UI ↔ API boundary over the TargetsService Connect-RPC
 * contract. Targets are the runtime state a scenario owns (a database, a
 * filesystem tree, a vector store) that the manager backs up. The UI offers
 * full co-equal CRUD over them alongside scenario self-registration.
 *
 * Thin async wrappers (rather than calling the client inline in hooks) keep the
 * mockable surface in one place — tests `vi.mock("../api/targets")` the same
 * way they mock `api/health`.
 */
import { createClient, type Client } from "@connectrpc/connect";
import { TargetsService } from "@vrooli/proto-types/data-backup-manager/v1/targets/targets_pb";
import type { Target } from "@vrooli/proto-types/data-backup-manager/v1/targets/targets_pb";
import { SourceKind } from "@vrooli/proto-types/data-backup-manager/v1/sources/sources_pb";

import { transport } from "./client";

export const targetsClient: Client<typeof TargetsService> = createClient(
  TargetsService,
  transport,
);

export interface RegisterTargetInput {
  owner: string;
  name: string;
  sourceKind: SourceKind;
  locator: string;
  critical: boolean;
}

export async function listTargets(owner = ""): Promise<Target[]> {
  const res = await targetsClient.listTargets({ owner });
  return res.targets;
}

export async function getTarget(id: string): Promise<Target | undefined> {
  const res = await targetsClient.getTarget({ id });
  return res.target;
}

/** Idempotent owner+name upsert; returns the registered target. */
export async function registerTarget(input: RegisterTargetInput): Promise<Target | undefined> {
  const res = await targetsClient.registerTarget(input);
  return res.target;
}

/** Removes a target by its owner+name key; returns whether a row was removed. */
export async function deregisterTarget(owner: string, name: string): Promise<boolean> {
  const res = await targetsClient.deregisterTarget({ owner, name });
  return res.removed;
}

export { SourceKind };
export type { Target };

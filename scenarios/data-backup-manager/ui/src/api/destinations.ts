/**
 * Destinations domain — UI ↔ API boundary over DestinationsService. A
 * destination is a kopia repository (local filesystem or S3/MinIO) the manager
 * snapshots into. Encryption is always on and secrets live in vault — the UI
 * only ever *displays* `encryptionAlgorithm` and `secretRef`, never edits them.
 * Update is intentionally cap-only (cap_bytes / cap_policy); name, backend, and
 * location are immutable once a repository exists.
 */
import { createClient, type Client } from "@connectrpc/connect";
import { DestinationsService } from "@vrooli/proto-types/data-backup-manager/v1/destinations/destinations_pb";
import {
  BackendKind,
  CapPolicy,
  PreparationAction,
  ReadinessSeverity,
  UsageState,
} from "@vrooli/proto-types/data-backup-manager/v1/destinations/destinations_pb";
import type {
  AnalyzeDestinationRequest,
  Destination,
  DestinationPreparationPlan,
  DestinationReadinessReport,
  ExecuteDestinationPreparationRequest,
  ExecuteDestinationPreparationResponse,
  GetDestinationUsageResponse,
  PlanDestinationPreparationRequest,
} from "@vrooli/proto-types/data-backup-manager/v1/destinations/destinations_pb";

import { transport } from "./client";

export const destinationsClient: Client<typeof DestinationsService> = createClient(
  DestinationsService,
  transport,
);

export interface CreateDestinationInput {
  name: string;
  backendKind: BackendKind;
  location: string;
  capBytes: bigint;
  capPolicy: CapPolicy;
}

export interface UpdateDestinationInput {
  id: string;
  capBytes: bigint;
  capPolicy: CapPolicy;
}

export interface AnalyzeDestinationInput {
  location: string;
  proposedSubdir?: string;
  selectedTargetBytes?: bigint;
  retentionCopies?: number;
  crossPlatformRequired?: boolean;
}

export async function listDestinations(): Promise<Destination[]> {
  const res = await destinationsClient.listDestinations({});
  return res.destinations;
}

export async function getDestination(id: string): Promise<Destination | undefined> {
  const res = await destinationsClient.getDestination({ id });
  return res.destination;
}

export async function createDestination(
  input: CreateDestinationInput,
): Promise<Destination | undefined> {
  const res = await destinationsClient.createDestination(input);
  return res.destination;
}

/** Cap-only update (cap_bytes + cap_policy). */
export async function updateDestination(
  input: UpdateDestinationInput,
): Promise<Destination | undefined> {
  const res = await destinationsClient.updateDestination(input);
  return res.destination;
}

/** Deletes the destination record; `deleteRepository` also drops the kopia repo. */
export async function deleteDestination(id: string, deleteRepository: boolean): Promise<void> {
  await destinationsClient.deleteDestination({ id, deleteRepository });
}

export async function getDestinationUsage(id: string): Promise<GetDestinationUsageResponse> {
  return destinationsClient.getDestinationUsage({ id });
}

export async function analyzeDestination(
  input: AnalyzeDestinationInput,
): Promise<DestinationReadinessReport | undefined> {
  const req: AnalyzeDestinationRequest = {
    $typeName: "vrooli.data_backup_manager.v1.destinations.AnalyzeDestinationRequest",
    location: input.location,
    proposedSubdir: input.proposedSubdir ?? "",
    selectedTargetBytes: input.selectedTargetBytes ?? 0n,
    retentionCopies: input.retentionCopies ?? 0,
    crossPlatformRequired: input.crossPlatformRequired ?? false,
  };
  const res = await destinationsClient.analyzeDestination(req);
  return res.report;
}

export async function planDestinationPreparation(
  input: PlanDestinationPreparationRequest,
): Promise<DestinationPreparationPlan | undefined> {
  const res = await destinationsClient.planDestinationPreparation(input);
  return res.plan;
}

export async function executeDestinationPreparation(
  input: ExecuteDestinationPreparationRequest,
): Promise<ExecuteDestinationPreparationResponse> {
  return destinationsClient.executeDestinationPreparation(input);
}

export { BackendKind, CapPolicy, PreparationAction, ReadinessSeverity, UsageState };
export type {
  Destination,
  DestinationPreparationPlan,
  DestinationReadinessReport,
  ExecuteDestinationPreparationResponse,
  GetDestinationUsageResponse,
};

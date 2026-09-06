import { createClient } from "@connectrpc/connect";
import {
  InvestigationPolicyService,
  type IncidentOccurrence,
  type IncidentRecord,
  type PolicyRecord,
} from "@vrooli/proto-types/plan-manager/v1/investigation/investigation_pb";

import { transport } from "./client";

export const investigationClient = createClient(InvestigationPolicyService, transport);

export async function getInvestigationPolicy(): Promise<PolicyRecord> {
  return investigationClient.getPolicy({});
}

export async function updateInvestigationPolicy(policyJson: string, expectedVersion?: string): Promise<PolicyRecord> {
  return investigationClient.putPolicy({ policyJson, active: true, expectedVersion: expectedVersion ?? "" });
}

export async function listInvestigationIncidents(options?: {
  executionId?: string;
  familyId?: string;
  state?: string;
  limit?: number;
}): Promise<IncidentRecord[]> {
  const response = await investigationClient.listIncidents({
    executionId: options?.executionId ?? "",
    familyId: options?.familyId ?? "",
    state: options?.state ?? "",
    limit: options?.limit ?? 50,
  });
  return response.incidents;
}

export async function getInvestigationIncident(fingerprint: string): Promise<IncidentRecord> {
  return investigationClient.getIncident({ incidentFingerprint: fingerprint });
}

export async function listInvestigationOccurrences(options?: {
  executionId?: string;
  eligibleOnly?: boolean;
  limit?: number;
}): Promise<IncidentOccurrence[]> {
  const response = await investigationClient.listOccurrences({
    executionId: options?.executionId ?? "",
    eligibleOnly: options?.eligibleOnly ?? false,
    limit: options?.limit ?? 50,
  });
  return response.occurrences;
}

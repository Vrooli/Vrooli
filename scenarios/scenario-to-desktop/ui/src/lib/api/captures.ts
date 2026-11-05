import type {
  EvidenceCapture,
  EvidenceCapturesSummary,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";
import { evidenceConnectClient } from "./connect";
import { buildUrl } from "./client";

export async function listCaptures(
  scenario: string,
): Promise<EvidenceCapture[]> {
  return (
    await evidenceConnectClient.listEvidenceCaptures({
      scenarioName: scenario,
    })
  ).captures;
}

export async function getCapturesSummary(
  scenario: string,
): Promise<EvidenceCapturesSummary> {
  return evidenceConnectClient.getEvidenceCapturesSummary({
    scenarioName: scenario,
  });
}

export function buildCaptureFileUrl(
  scenario: string,
  captureId: string,
): string {
  return buildUrl(
    `/captures/${encodeURIComponent(scenario)}/${encodeURIComponent(captureId)}/file`,
  );
}

export async function deleteCapture(
  scenario: string,
  captureId: string,
): Promise<void> {
  await evidenceConnectClient.deleteEvidenceCapture({
    scenarioName: scenario,
    captureId,
  });
}

export async function deleteAllCaptures(scenario: string): Promise<void> {
  await evidenceConnectClient.deleteAllEvidenceCaptures({
    scenarioName: scenario,
  });
}

export function buildCapturesDownloadUrl(
  scenario: string,
  ids: string[],
): string {
  return buildUrl(
    `/captures/${encodeURIComponent(scenario)}/download?ids=${ids.map(encodeURIComponent).join(",")}`,
  );
}

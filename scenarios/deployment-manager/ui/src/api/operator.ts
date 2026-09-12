import { create, fromJson, toJson, type DescMessage, type MessageShape } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { scenarioTransport } from "./transport";
import { ValueSchema, type Value } from "@bufbuild/protobuf/wkt";
import {
  DependenciesService,
} from "@vrooli/proto-types/deployment-manager/v1/dependencies/dependencies_pb";
import { FitnessService } from "@vrooli/proto-types/deployment-manager/v1/fitness/fitness_pb";
import { DeploymentsService } from "@vrooli/proto-types/deployment-manager/v1/deployments/deployments_pb";
import { SwapsService } from "@vrooli/proto-types/deployment-manager/v1/swaps/swaps_pb";
import { TelemetryService } from "@vrooli/proto-types/deployment-manager/v1/telemetry/telemetry_pb";
import { MigrationService } from "@vrooli/proto-types/deployment-manager/v1/migration/migration_pb";
import { ApprovalsService } from "@vrooli/proto-types/deployment-manager/v1/approvals/approvals_pb";
import { LPBSService } from "@vrooli/proto-types/deployment-manager/v1/lpbs/lpbs_pb";
import { ReleasesService } from "@vrooli/proto-types/deployment-manager/v1/releases/releases_pb";
import {
  GetReleaseRequestSchema,
  GetReleaseResponseSchema,
  GetReleaseDossierRequestSchema,
  GetReleaseDossierResponseSchema,
  GetReleaseOperationRequestSchema,
  GetReleaseOperationResponseSchema,
  ListReleasesRequestSchema,
  ListReleasesResponseSchema,
  ReverifyReleaseRequestSchema,
  ReverifyReleaseResponseSchema,
  StartReleaseRequestSchema,
  StartReleaseResponseSchema,
} from "@vrooli/proto-types/deployment-manager/v1/releases/contracts_pb";
import {
  EvidenceService,
  GetEvidenceReviewRequestSchema,
} from "@vrooli/proto-types/deployment-manager/v1/evidence/evidence_pb";

const transport = scenarioTransport;
const dependenciesClient = createClient(DependenciesService, transport);
const fitnessClient = createClient(FitnessService, transport);
const deploymentsClient = createClient(DeploymentsService, transport);
const swapsClient = createClient(SwapsService, transport);
const telemetryClient = createClient(TelemetryService, transport);
const migrationClient = createClient(MigrationService, transport);
const approvalsClient = createClient(ApprovalsService, transport);
const lpbsClient = createClient(LPBSService, transport);
const releasesClient = createClient(ReleasesService, transport);
const evidenceClient = createClient(EvidenceService, transport);

const value = (payload: unknown): Value => fromJson(ValueSchema, payload as Parameters<typeof fromJson>[1]);
const plain = (payload: Value): unknown => toJson(ValueSchema, payload);
const typedPlain = <Desc extends DescMessage>(schema: Desc, payload: MessageShape<Desc>): unknown =>
  toJson(schema, payload, { useProtoFieldName: true });
const objectValue = (payload: unknown): Record<string, unknown> => {
  if (typeof payload !== "object" || payload === null || Array.isArray(payload)) {
    throw new Error("typed release response is not an object");
  }
  return Object.fromEntries(Object.entries(payload));
};

export const operatorApi = {
  analyzeDependencies: (scenario: string) => dependenciesClient.analyze(value({ scenario })).then((r) => plain(r)),
  scoreFitness: (payload: unknown) => fitnessClient.score(value(payload)).then((r) => plain(r)),
  deploy: (profileId: string) => deploymentsClient.deploy(value({ profile_id: profileId })).then((r) => plain(r)),
  deploymentStatus: (deploymentId: string) => deploymentsClient.status(value({ deployment_id: deploymentId })).then((r) => plain(r)),
  swapAnalyze: (from: string, to: string) => swapsClient.analyze(value({ from, to })).then((r) => plain(r)),
  swapCascade: (from: string, to: string) => swapsClient.cascade(value({ from, to })).then((r) => plain(r)),
  telemetry: () => telemetryClient.list(value({})).then((r) => plain(r)),
  telemetryUpload: (payload: unknown) => telemetryClient.upload(value(payload)).then((r) => plain(r)),
  migrationReport: (payload: unknown) => migrationClient.report(value(payload)).then((r) => plain(r)),
  migrationStatus: (name: string, kind: string) => migrationClient.status(value({ name, kind })).then((r) => plain(r)),
  approvals: (profileId: string, commit?: string) => approvalsClient.list(value({ profile_id: profileId, git_commit_hash: commit ?? "" })).then((r) => plain(r)),
  approval: (id: string) => approvalsClient.get(value({ id })).then((r) => plain(r)),
  createApproval: (profileId: string, payload: unknown) => approvalsClient.create(value({ profile_id: profileId, ...(payload as object) })).then((r) => plain(r)),
  decideApproval: (id: string, payload: unknown) => approvalsClient.decide(value({ id, ...(payload as object) })).then((r) => plain(r)),
  releaseGate: (profileId: string, commit: string) => approvalsClient.checkReleaseGate(value({ profile_id: profileId, git_commit_hash: commit })).then((r) => plain(r)),
  setRequiredPlatforms: (profileId: string, platforms: string[]) => approvalsClient.setRequiredPlatforms(value({ profile_id: profileId, platforms })).then((r) => plain(r)),
  requiredPlatforms: (profileId: string) => approvalsClient.getRequiredPlatforms(value({ profile_id: profileId })).then((r) => plain(r)),
  evidenceReview: (profileId: string, commit: string) => evidenceClient.getEvidenceReview(create(GetEvidenceReviewRequestSchema, { profileId, gitCommitHash: commit })),
  lpbsConfig: (profileId: string) => lpbsClient.getConfig(value({ profile_id: profileId })).then((r) => plain(r)),
  saveLpbsConfig: (profileId: string, payload: unknown) => lpbsClient.saveConfig(value({ profile_id: profileId, ...(payload as object) })).then((r) => plain(r)),
  releases: (profileId: string, limit: number) => releasesClient.list(create(ListReleasesRequestSchema, { profileId, limit })).then((r) => typedPlain(ListReleasesResponseSchema, r)),
  release: (releaseId: string) => releasesClient.get(create(GetReleaseRequestSchema, { releaseId })).then((r) => objectValue(typedPlain(GetReleaseResponseSchema, r)).release),
  releaseDossier: (releaseId: string) => releasesClient.dossier(create(GetReleaseDossierRequestSchema, { releaseId })).then((r) => objectValue(typedPlain(GetReleaseDossierResponseSchema, r)).dossier),
  releaseOperation: (operationId: string) => releasesClient.getOperation(create(GetReleaseOperationRequestSchema, { operationId })).then((r) => objectValue(typedPlain(GetReleaseOperationResponseSchema, r)).operation),
  reverify: (releaseId: string, deep: boolean) => releasesClient.reverify(create(ReverifyReleaseRequestSchema, { releaseId, deep })).then((r) => objectValue(typedPlain(ReverifyReleaseResponseSchema, r)).release),
  startRelease: (profileId: string, payload: unknown) => {
    const request = payload as Record<string, unknown>;
    return releasesClient.start(create(StartReleaseRequestSchema, {
      profileId,
      channel: String(request.channel ?? ""),
      gitCommitHash: String(request.git_commit_hash ?? ""),
      artifactDigest: String(request.artifact_digest ?? ""),
      candidateId: String(request.candidate_id ?? ""),
      destinationRevisionId: String(request.destination_revision_id ?? ""),
      authorizationEpoch: BigInt(Number(request.authorization_epoch ?? 0)),
      idempotencyKey: String(request.idempotency_key ?? ""),
      readinessReviewKey: String(request.readiness_review_key ?? ""),
      releaseVersion: String(request.release_version ?? ""),
      releaseNotes: String(request.release_notes ?? ""),
      platforms: Array.isArray(request.platforms) ? request.platforms.map(String) : [],
      cloudManifestJson: typeof request.cloud_manifest === "string" ? request.cloud_manifest : request.cloud_manifest ? JSON.stringify(request.cloud_manifest) : "",
      cloudDeploymentName: String(request.cloud_deployment_name ?? ""),
      cloudBundlePath: String(request.cloud_bundle_path ?? ""),
      cloudBundleSha256: String(request.cloud_bundle_sha256 ?? ""),
      cloudBundleSizeBytes: BigInt(Number(request.cloud_bundle_size_bytes ?? 0)),
      cloudRunPreflight: Boolean(request.cloud_run_preflight),
    })).then((r) => typedPlain(StartReleaseResponseSchema, r));
  },
};

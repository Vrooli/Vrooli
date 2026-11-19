import { createClient } from "@connectrpc/connect";
import {
  ComponentsService,
  type Component,
  type ListComponentsResponse,
  type ResolveLibraryImportResponse,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import { transport, authoringTransport } from "./client";
import type { ClaimMeasurement } from "@vrooli/proto-types/experience-manager/v1/contract/contract_pb";

export const DesignAffinity = {
  UNSPECIFIED: 0,
  NATIVE: 1,
  COMPATIBLE: 2,
  DISCOURAGED: 3,
} as const;

export const StyleFitVerdictKind = {
  UNSPECIFIED: 0,
  OK: 1,
  INFO: 2,
  WARN: 3,
} as const;

export type DesignAffinity = (typeof DesignAffinity)[keyof typeof DesignAffinity];
export type StyleFitVerdictKind = (typeof StyleFitVerdictKind)[keyof typeof StyleFitVerdictKind];

export interface ValidateStyleFitRequest {
  componentId: string;
  scenario: string;
  version?: string;
}

export interface ValidateStyleFitResponse {
  kind: StyleFitVerdictKind;
  componentId: string;
  version: string;
  scenario: string;
  scenarioStyle: string;
  affinity: DesignAffinity;
  detail: string;
}

const baseComponentsClient = createClient(ComponentsService, transport);

/** Mutating authoring calls share a longer deadline because checks compile
 * the story harness. Reads stay on the interactive transport. */
const authoringComponentsClient = createClient(ComponentsService, authoringTransport);

export const componentsClient = baseComponentsClient as typeof baseComponentsClient & {
  validateStyleFit(input: ValidateStyleFitRequest): Promise<ValidateStyleFitResponse>;
};

export interface ComponentStory {
  id: string;
  componentId: string;
  libraryId: string;
  version: string;
  schemaVersion: number;
  kind: string;
  title: string;
  argsJson: string;
  environmentJson: string;
  storiesJson: string;
  contractJson: string;
  sourcePath: string;
}

export interface ListComponentStoriesRequest {
  componentId: string;
  version?: string;
  limit?: number;
}

export interface ListComponentStoriesResponse {
  stories: ComponentStory[];
  warnings?: string[];
}

export interface PreviewFrameCandidate {
  asset: string;
  version: string;
  region: string;
  capability: string;
  fixture: string;
  label: string;
  compatible: boolean;
  diagnosticCode: string;
  diagnostic: string;
}

export interface ListPreviewFramesRequest {
  componentId: string;
  version?: string;
  storyId?: string;
}

export interface ListPreviewFramesResponse {
  candidates: PreviewFrameCandidate[];
}

export interface PersistPreviewFrameRequest {
  componentId: string;
  version?: string;
  storyId: string;
  asset: string;
  frameVersion: string;
  region: string;
  capability?: string;
  fixture?: string;
}

export interface PersistPreviewFrameResponse {
  componentId: string;
  version: string;
  storyId: string;
  storyJson: string;
  sourcePath: string;
}

export interface ComponentExperienceState {
  id: string;
  exampleName: string;
  description: string;
}

export interface ComponentExperienceClaim {
  id: string;
  type: string;
  statement: string;
  tier: string;
  states: string[];
}

export interface ComponentExperienceEvidence {
  claimId: string;
  verdict: string;
  stateId: string;
  exampleName: string;
  captureRef: string;
  checkedAt: string;
  message: string;
  viewport: string;
  viewportWidth: number;
  viewportHeight: number;
  measurement?: ClaimMeasurement;
}

export interface ComponentExperience {
  componentId: string;
  libraryId: string;
  version: string;
  contractId: string;
  title: string;
  purpose: string;
  states: ComponentExperienceState[];
  claims: ComponentExperienceClaim[];
  evidence: ComponentExperienceEvidence[];
  evidenceStatus: string;
  evidenceMessage: string;
}

export async function getComponentExperience(componentId: string): Promise<ComponentExperience> {
  const response = await baseComponentsClient.getComponent({
    id: componentId,
    includeExperience: true,
  });
  if (!response.experience) throw new Error("component experience was not returned");
  return response.experience;
}

export async function listComponentStories(
  input: ListComponentStoriesRequest,
): Promise<ListComponentStoriesResponse> {
  const client = baseComponentsClient as unknown as {
    listComponentStories(
      request: ListComponentStoriesRequest,
    ): Promise<ListComponentStoriesResponse>;
  };
  return client.listComponentStories(input);
}

export async function listPreviewFrames(
  input: ListPreviewFramesRequest,
): Promise<ListPreviewFramesResponse> {
  const client = baseComponentsClient as unknown as {
    listPreviewFrames(request: { componentId: string; version: string; storyId: string }): Promise<{
      candidates: Array<{
        asset: string;
        version: string;
        region: string;
        capability: string;
        fixture: string;
        label: string;
        compatible: boolean;
        diagnosticCode: string;
        diagnostic: string;
      }>;
    }>;
  };
  const response = await client.listPreviewFrames({
    componentId: input.componentId,
    version: input.version ?? "",
    storyId: input.storyId ?? "",
  });
  return {
    candidates: response.candidates.map((candidate) => ({
      asset: candidate.asset,
      version: candidate.version,
      region: candidate.region,
      capability: candidate.capability,
      fixture: candidate.fixture,
      label: candidate.label,
      compatible: candidate.compatible,
      diagnosticCode: candidate.diagnosticCode,
      diagnostic: candidate.diagnostic,
    })),
  };
}

export async function persistPreviewFrame(
  input: PersistPreviewFrameRequest,
): Promise<PersistPreviewFrameResponse> {
  const client = baseComponentsClient as unknown as {
    persistPreviewFrame(request: PersistPreviewFrameRequest): Promise<PersistPreviewFrameResponse>;
  };
  return client.persistPreviewFrame(input);
}

export type { Component, ListComponentsResponse };

// Catalog callers use the generated Connect contract directly. Keeping this
// alias makes the asset terminology explicit without a raw JSON fallback.
export type CatalogAsset = Component;

export async function listCatalogAssets(input: {
  limit?: number;
  match?: string;
  assetKind?: 1 | 2;
  catalogKinds?: string[];
}): Promise<ListComponentsResponse> {
  return baseComponentsClient.listComponents(input);
}

export async function getCatalogAsset(id: string): Promise<{ component?: CatalogAsset }> {
  return baseComponentsClient.getComponent({ id });
}

export type LibraryImportResolution = ResolveLibraryImportResponse;

export async function resolveLibraryImport(input: {
  specifier: string;
  fromPath?: string;
}): Promise<LibraryImportResolution> {
  return baseComponentsClient.resolveLibraryImport({
    specifier: input.specifier,
    fromPath: input.fromPath ?? "",
  });
}

/* ── Component authoring ───────────────────────────────────────────────────
 *
 * These calls are the browser half of the same governed loop the CLI uses:
 * begin a mutable draft, write one artifact with an optimistic-concurrency
 * guard, run the focused check, then publish through the check again. The
 * server rejects writes when no draft is active, so a browser save can never
 * mutate a released version.
 */

export interface BeginComponentDraftInput {
  /** Stable libraryId or catalog UUID. */
  component: string;
  /** major | minor | patch. Ignored when `version` names a release. */
  bump?: string;
  version?: string;
}

export interface BeginComponentDraftResult {
  component: Component;
  version: string;
  sourcePath: string;
  artifactPaths: string[];
  previewPath: string;
}

export interface ComponentVersionCheckResult {
  component: Component;
  version: string;
  passed: boolean;
  checks: Array<{
    stage: string;
    verdict: string;
    message: string;
    remediation: string;
  }>;
  previewPath: string;
}

export interface PublishComponentDraftInput {
  component: string;
  draftVersion: string;
  version?: string;
  changelogMd?: string;
  acknowledgeParityWaiver?: boolean;
}

export interface PublishComponentDraftResult {
  component: Component;
  version: string;
  sourcePath: string;
  artifactPaths: string[];
  previewPath: string;
}

export async function beginComponentDraft(
  input: BeginComponentDraftInput,
): Promise<BeginComponentDraftResult> {
  const response = await authoringComponentsClient.beginComponentVersion({
    component: input.component,
    bump: input.bump ?? "",
    version: input.version ?? "",
  });
  if (!response.component) throw new Error("begin draft returned no component");
  return {
    component: response.component,
    version: response.version?.version ?? "",
    sourcePath: response.sourcePath,
    artifactPaths: response.artifactPaths,
    previewPath: response.previewPath,
  };
}

export interface UpdateComponentContentInput {
  /** Stable libraryId or catalog UUID of the draft owner. */
  id: string;
  content: string;
  /** Empty = no guard. Always pass the digest returned by the read. */
  expectedSha256?: string;
  /** Empty = component entry file; otherwise a companion in the same folder. */
  path?: string;
}

export async function updateComponentContent(
  input: UpdateComponentContentInput,
): Promise<{ sha256: string; sourcePath: string }> {
  return authoringComponentsClient.updateComponentContent({
    id: input.id,
    content: input.content,
    expectedSha256: input.expectedSha256 ?? "",
    path: input.path ?? "",
  });
}

export async function checkComponentVersion(input: {
  component: string;
  version: string;
}): Promise<ComponentVersionCheckResult> {
  const response = await authoringComponentsClient.checkComponentVersion({
    component: input.component,
    version: input.version,
  });
  if (!response.component) throw new Error("version check returned no component");
  return {
    component: response.component,
    version: response.version,
    passed: response.passed,
    checks: response.checks.map((check) => ({
      stage: check.stage,
      verdict: check.verdict,
      message: check.message,
      remediation: check.remediation,
    })),
    previewPath: response.previewPath,
  };
}

export async function publishComponentDraft(
  input: PublishComponentDraftInput,
): Promise<PublishComponentDraftResult> {
  const response = await authoringComponentsClient.publishComponentVersion({
    component: input.component,
    draftVersion: input.draftVersion,
    version: input.version ?? "",
    changelogMd: input.changelogMd ?? "",
    acknowledgeParityWaiver: input.acknowledgeParityWaiver ?? false,
  });
  if (!response.component) throw new Error("publish returned no component");
  return {
    component: response.component,
    version: response.version?.version ?? "",
    sourcePath: response.sourcePath,
    artifactPaths: response.artifactPaths,
    previewPath: response.previewPath,
  };
}

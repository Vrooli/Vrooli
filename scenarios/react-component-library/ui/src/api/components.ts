import { createClient } from "@connectrpc/connect";
import {
  ComponentsService,
  type Component,
  type ListComponentsResponse,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import { transport } from "./client";
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

export const componentsClient = baseComponentsClient as typeof baseComponentsClient & {
  validateStyleFit(input: ValidateStyleFitRequest): Promise<ValidateStyleFitResponse>;
};

export interface ComponentStory {
  id: string;
  componentId: string;
  libraryId: string;
  version: string;
  schemaVersion: number;
  kind: "component" | "hook" | string;
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
  tier: "machine" | "manual" | "aspirational" | string;
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
  assetKind: 1 | 2;
}): Promise<ListComponentsResponse> {
  return baseComponentsClient.listComponents(input);
}

export async function getCatalogAsset(id: string): Promise<{ component?: CatalogAsset }> {
  return baseComponentsClient.getComponent({ id });
}

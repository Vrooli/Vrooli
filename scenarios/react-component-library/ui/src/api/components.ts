import { createClient } from "@connectrpc/connect";
import {
  ComponentsService,
  type Component,
  type ListComponentsResponse,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import { transport } from "./client";

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

export interface ListComponentExamplesRequest {
  componentId: string;
  version?: string;
  limit?: number;
}

export interface ComponentExample {
  id: string;
  componentId: string;
  libraryId: string;
  version: string;
  name: string;
  displayName: string;
  propsJson: string;
  setupJson: string;
  expectJson: string;
  controlsJson: string;
  sourcePath: string;
}

export interface ListComponentExamplesResponse {
  examples: ComponentExample[];
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
  const response = await baseComponentsClient.getComponent({ id: componentId, includeExperience: true });
  if (!response.experience) throw new Error("component experience was not returned");
  return response.experience as ComponentExperience;
}

export async function listComponentExamples(
  input: ListComponentExamplesRequest,
): Promise<ListComponentExamplesResponse> {
  const client = baseComponentsClient as unknown as {
    listComponentExamples(request: ListComponentExamplesRequest): Promise<ListComponentExamplesResponse>;
  };
  return client.listComponentExamples(input);
}

export type { Component, ListComponentsResponse };

// Catalog callers use the generated Connect contract directly. Keeping this
// alias makes the asset terminology explicit without a raw JSON fallback.
export type CatalogAsset = Component;

export async function listCatalogAssets(input: { limit?: number; match?: string; assetKind: 1 | 2 }): Promise<ListComponentsResponse> {
  return baseComponentsClient.listComponents(input);
}

export async function getCatalogAsset(id: string): Promise<{ component?: CatalogAsset }> {
  return baseComponentsClient.getComponent({ id });
}

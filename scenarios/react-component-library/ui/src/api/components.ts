import { createClient } from "@connectrpc/connect";
import {
  ComponentsService,
  type Component,
  type ListComponentsResponse,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import { transport } from "./client";
import { API_BASE, decodeApiError } from "./client";
import { buildApiUrl } from "@vrooli/api-base";

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
  sourcePath: string;
}

export interface ListComponentExamplesResponse {
  examples: ComponentExample[];
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

// Asset is the forward-compatible catalog projection. The local generated
// package can lag a freshly generated server contract during development, so
// this narrow RCL-only JSON reader preserves the typed client for all existing
// operations while exposing the new server-projected fields without a browser
// call to any external scenario.
export type CatalogAsset = Omit<Component, "assetKind" | "metrics"> & {
  assetKind: 1 | 2;
  metrics?: { directAdoptionCount: number; versionCount: number };
};

export async function listCatalogAssets(input: { limit?: number; match?: string; assetKind: 1 | 2 }): Promise<{ components: CatalogAsset[] }> {
  const response = await fetch(buildApiUrl("/vrooli.react_component_library.v1.components.ComponentsService/ListComponents", { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!response.ok) throw await decodeApiError(response);
  return await response.json() as { components: CatalogAsset[] };
}

export async function getCatalogAsset(id: string): Promise<{ component?: CatalogAsset }> {
  const response = await fetch(buildApiUrl("/vrooli.react_component_library.v1.components.ComponentsService/GetComponent", { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
  if (!response.ok) throw await decodeApiError(response);
  return await response.json() as { component?: CatalogAsset };
}

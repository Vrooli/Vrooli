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

export type { Component, ListComponentsResponse };

import { createClient } from "@connectrpc/connect";
import { buildApiUrl } from "@vrooli/api-base";
import {
  CatalogService,
  type CoverageReport,
  type GetHealthOverviewResponse,
  type ListNextWorkResponse,
} from "@vrooli/proto-types/react-component-library/v1/catalog/catalog_pb";

import { API_BASE, transport } from "./client";

export type { CoverageReport, GetHealthOverviewResponse, ListNextWorkResponse };

const catalogClient = createClient(CatalogService, transport);

export async function getCatalogCoverage(): Promise<CoverageReport> {
  const response = await catalogClient.getCoverage({});
  if (!response.report) throw new Error("catalog coverage was not returned");
  return response.report;
}

export async function listCatalogNextWork(limit = 10): Promise<ListNextWorkResponse> {
  return catalogClient.listNextWork({ limit });
}

export async function getCatalogHealthOverview(): Promise<GetHealthOverviewResponse> {
  return catalogClient.getHealthOverview({});
}

export interface CapabilityDefinition {
  id: string;
  name: string;
  description: string;
  dependencyKind: string;
  dependencySlug: string;
  features?: string[];
  actionKind?: string;
  actionLabel?: string;
  operatorCommand?: string;
}

export interface CapabilityState extends CapabilityDefinition {
  status: string;
  message?: string;
  reasonCode?: string;
  checkedAt?: string;
}

export interface CapabilityDescription {
  definitions: CapabilityDefinition[];
  states: CapabilityState[];
}

export async function describeCapabilities(): Promise<CapabilityDescription> {
  const response = await fetch(
    buildApiUrl("/api/v1/capabilities/describe", { baseUrl: API_BASE }),
    { cache: "no-store" },
  );
  if (!response.ok) throw new Error(`capability registry returned ${response.status}`);
  return (await response.json()) as CapabilityDescription;
}

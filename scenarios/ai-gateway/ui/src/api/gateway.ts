import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import {
  Profile,
  PrivacyClass,
  RequestKind,
  GatewayRequestSchema,
  type GatewayRequest,
} from "@vrooli/proto-types/ai-gateway/v1/shared/gateway_pb";
import {
  ConformanceService,
  ScanScenarioRequestSchema,
  type ScanScenarioResponse,
} from "@vrooli/proto-types/ai-gateway/v1/conformance/conformance_pb";
import {
  InventoryService,
  ListProviderRolesRequestSchema,
  SmokeProviderRequestSchema,
  type ListProviderRolesResponse,
  type SmokeProviderResponse,
} from "@vrooli/proto-types/ai-gateway/v1/inventory/inventory_pb";
import {
  ListRouteEvidenceRequestSchema,
  PreviewRouteRequestSchema,
  RoutingService,
  type ListRouteEvidenceResponse,
  type PreviewRouteResponse,
} from "@vrooli/proto-types/ai-gateway/v1/routing/routing_pb";

import { PROFILE_CATALOG } from "../consts/profileCatalog.generated";
import { strings } from "../consts/strings";

import { transport } from "./client";

const inventoryClient = createClient(InventoryService, transport);
const routingClient = createClient(RoutingService, transport);
const conformanceClient = createClient(ConformanceService, transport);

export interface PreviewRouteInput {
  scenario: string;
  operation: string;
  role: string;
  profile: Profile;
  privacyClass: PrivacyClass;
  timeoutMs: number;
  maxCostUsd: number;
  maxOutputTokens: number;
}

export const profileOptions = PROFILE_CATALOG;

export const privacyOptions = [
  { value: PrivacyClass.PUBLIC, labelKey: strings.privacyClasses.public },
  { value: PrivacyClass.INTERNAL, labelKey: strings.privacyClasses.internal },
  { value: PrivacyClass.CONFIDENTIAL, labelKey: strings.privacyClasses.confidential },
  { value: PrivacyClass.SECRET, labelKey: strings.privacyClasses.secret },
] as const;

export const defaultPreviewInput: PreviewRouteInput = {
  scenario: "ai-gateway",
  operation: "operator.route-preview",
  role: "chat.default",
  profile: Profile.LOCAL_FIRST,
  privacyClass: PrivacyClass.INTERNAL,
  timeoutMs: 30_000,
  maxCostUsd: 0.05,
  maxOutputTokens: 512,
};

export function buildGatewayRequest(input: PreviewRouteInput): GatewayRequest {
  return create(GatewayRequestSchema, {
    kind: RequestKind.TEXT_GENERATION,
    role: input.role,
    profile: input.profile,
    privacyClass: input.privacyClass,
    operation: input.operation,
    scenario: input.scenario,
    timeoutMs: input.timeoutMs,
    maxCostUsd: input.maxCostUsd,
    maxOutputTokens: input.maxOutputTokens,
    requestId: `ui-${Date.now()}`,
    metadata: {},
  });
}

export async function listProviderRoles(provider = ""): Promise<ListProviderRolesResponse> {
  return inventoryClient.listProviderRoles(create(ListProviderRolesRequestSchema, { provider }));
}

export async function smokeProvider(provider: string): Promise<SmokeProviderResponse> {
  return inventoryClient.smokeProvider(create(SmokeProviderRequestSchema, { provider }));
}

export async function previewRoute(input: PreviewRouteInput): Promise<PreviewRouteResponse> {
  return routingClient.previewRoute(
    create(PreviewRouteRequestSchema, {
      request: buildGatewayRequest(input),
    }),
  );
}

export async function listRouteEvidence(
  limit = 10,
  scenario = "",
): Promise<ListRouteEvidenceResponse> {
  return routingClient.listRouteEvidence(
    create(ListRouteEvidenceRequestSchema, {
      limit,
      scenario,
    }),
  );
}

export async function scanScenario(
  scenario: string,
  path = "",
): Promise<ScanScenarioResponse> {
  return conformanceClient.scanScenario(create(ScanScenarioRequestSchema, { scenario, path }));
}

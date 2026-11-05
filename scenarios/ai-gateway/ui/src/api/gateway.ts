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

export const profileOptions = [
  { value: Profile.LOCAL_ONLY, label: "Local only" },
  { value: Profile.LOCAL_FIRST, label: "Local first" },
  { value: Profile.REMOTE_ONLY, label: "Remote only" },
  { value: Profile.QUALITY_FIRST, label: "Quality first" },
  { value: Profile.CHEAP_FIRST, label: "Cheap first" },
  { value: Profile.PRIVACY_SENSITIVE, label: "Privacy sensitive" },
] as const;

export const privacyOptions = [
  { value: PrivacyClass.PUBLIC, label: "Public" },
  { value: PrivacyClass.INTERNAL, label: "Internal" },
  { value: PrivacyClass.CONFIDENTIAL, label: "Confidential" },
  { value: PrivacyClass.SECRET, label: "Secret" },
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

import { createClient } from "@connectrpc/connect";
import {
  BehaviorOverride,
  IntegrationsService,
  type StatusResponse,
} from "@vrooli/proto-types/portal/v1/integrations/integrations_pb";
import { BehaviorMode, IntegrationState } from "@vrooli/proto-types/portal/v1/shared/common_pb";

import { transport } from "./client";

const integrationsClient = createClient(IntegrationsService, transport);

export async function fetchIntegrationsStatus(): Promise<StatusResponse> {
  return integrationsClient.status({});
}

export async function updateBehaviorOverride(override: BehaviorOverride): Promise<StatusResponse> {
  const response = await integrationsClient.updateOverride({ override });
  return response.status ?? fetchIntegrationsStatus();
}

export function behaviorModeLabel(mode: BehaviorMode): string {
  switch (mode) {
    case BehaviorMode.OFF:
      return "off";
    case BehaviorMode.PASSIVE:
      return "passive";
    case BehaviorMode.FULL:
      return "full";
    default:
      return "unknown";
  }
}

export function integrationStateLabel(state: IntegrationState): string {
  switch (state) {
    case IntegrationState.AVAILABLE:
      return "available";
    case IntegrationState.DEGRADED:
      return "degraded";
    case IntegrationState.UNAVAILABLE:
      return "unavailable";
    case IntegrationState.UNKNOWN:
      return "unknown";
    default:
      return "unspecified";
  }
}

export { BehaviorOverride, BehaviorMode, IntegrationState };
export type { StatusResponse };

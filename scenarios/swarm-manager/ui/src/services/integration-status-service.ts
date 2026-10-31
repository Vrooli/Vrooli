import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

export type IntegrationAvailability =
  | "available"
  | "unavailable"
  | "stale"
  | "unconfigured";

export interface IntegrationStatus {
  id: string;
  required: boolean;
  configured: boolean;
  availability: IntegrationAvailability;
  checkedAt: string;
  freshUntil: string;
  degradedBehavior: string;
  diagnostic: string;
  affectedTransitions: string[];
}

export interface IntegrationStatusResponse {
  integrations: IntegrationStatus[];
}

export interface IIntegrationStatusService {
  get(): Promise<IntegrationStatusResponse>;
}

export function createIntegrationStatusService(
  apiClient: IApiClient = defaultApiClient,
): IIntegrationStatusService {
  return {
    get: () => apiClient.get<IntegrationStatusResponse>(API_ENDPOINTS.integrations),
  };
}

export const integrationStatusService = createIntegrationStatusService();

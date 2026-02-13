import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { ExecutionMode, ExecutionPolicy } from "../types";

interface ExecutionPolicyResponse {
  policy: {
    default_mode: ExecutionMode;
    default_delay_seconds: number;
  };
}

export interface IExecutionPolicyService {
  get(): Promise<ExecutionPolicy>;
  update(policy: ExecutionPolicy): Promise<ExecutionPolicy>;
}

const mapPolicy = (response: ExecutionPolicyResponse): ExecutionPolicy => ({
  defaultMode: response.policy.default_mode,
  defaultDelaySeconds: response.policy.default_delay_seconds,
});

export function createExecutionPolicyService(apiClient: IApiClient = defaultApiClient): IExecutionPolicyService {
  return {
    async get(): Promise<ExecutionPolicy> {
      const response = await apiClient.get<ExecutionPolicyResponse>(API_ENDPOINTS.executionPolicy);
      return mapPolicy(response);
    },
    async update(policy: ExecutionPolicy): Promise<ExecutionPolicy> {
      const response = await apiClient.put<ExecutionPolicyResponse>(API_ENDPOINTS.executionPolicy, {
        default_mode: policy.defaultMode,
        default_delay_seconds: policy.defaultDelaySeconds,
      });
      return mapPolicy(response);
    },
  };
}

export const executionPolicyService = createExecutionPolicyService();

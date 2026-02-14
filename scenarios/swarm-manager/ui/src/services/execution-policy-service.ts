import { ExecutionPolicySchema } from "@vrooli/proto-types/swarm-manager/v1/domain/execution_pb";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { ExecutionPolicy } from "../types";
import {
  buildMessage,
  parseProtoResponse,
  requireProtoField,
  toProtoJson,
  executionPolicyResponseSchema,
  mapProtoExecutionPolicy,
} from "./proto-contracts";

export interface IExecutionPolicyService {
  get(): Promise<ExecutionPolicy>;
  update(policy: ExecutionPolicy): Promise<ExecutionPolicy>;
}

export function createExecutionPolicyService(apiClient: IApiClient = defaultApiClient): IExecutionPolicyService {
  const parsePolicy = (data: unknown): ExecutionPolicy => {
    const resp = parseProtoResponse(executionPolicyResponseSchema, data, "execution policy");
    return mapProtoExecutionPolicy(requireProtoField(resp.policy, "execution policy"));
  };

  return {
    async get(): Promise<ExecutionPolicy> {
      const response = await apiClient.get<unknown>(API_ENDPOINTS.executionPolicy);
      return parsePolicy(response);
    },
    async update(policy: ExecutionPolicy): Promise<ExecutionPolicy> {
      const msg = buildMessage(ExecutionPolicySchema, {
        defaultMode: policy.defaultMode,
        defaultDelaySeconds: BigInt(policy.defaultDelaySeconds),
      });
      const response = await apiClient.put<unknown>(
        API_ENDPOINTS.executionPolicy,
        toProtoJson(ExecutionPolicySchema, msg)
      );
      return parsePolicy(response);
    },
  };
}

export const executionPolicyService = createExecutionPolicyService();

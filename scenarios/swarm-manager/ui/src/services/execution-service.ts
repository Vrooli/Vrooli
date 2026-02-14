import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { BacklogKind, ExecutionMode, ExecutionRecord, ExecutionStatus } from "../types";
import {
  parseProtoResponse,
  requireProtoField,
  listExecutionResponseSchema,
  executionResponseSchema,
  mapProtoExecutionRecord,
  toProtoJson,
  buildMessage,
  CreateExecutionRequestSchema,
} from "./proto-contracts";

export interface CreateExecutionRequest {
  backlogKind: BacklogKind;
  backlogName: string;
  mode: ExecutionMode;
  delaySeconds?: number;
  startedBy?: string;
  operation?: "generator" | "improver";
}

export interface ListExecutionFilters {
  status?: ExecutionStatus;
  mode?: ExecutionMode;
  backlogKind?: BacklogKind;
  backlogName?: string;
  startedBy?: string;
  createdFrom?: string;
  createdTo?: string;
}

export interface IExecutionService {
  list(filters?: ListExecutionFilters): Promise<ExecutionRecord[]>;
  get(executionId: string): Promise<ExecutionRecord>;
  create(request: CreateExecutionRequest): Promise<ExecutionRecord>;
  start(executionId: string): Promise<ExecutionRecord>;
  cancel(executionId: string): Promise<ExecutionRecord>;
  retry(executionId: string): Promise<ExecutionRecord>;
}

export function createExecutionService(apiClient: IApiClient = defaultApiClient): IExecutionService {
  const parseExecution = (data: unknown): ExecutionRecord => {
    const resp = parseProtoResponse(executionResponseSchema, data, "execution");
    return mapProtoExecutionRecord(requireProtoField(resp.execution, "execution"));
  };

  const mutate = async (endpoint: string): Promise<ExecutionRecord> => {
    const data = await apiClient.post<unknown>(endpoint, {});
    return parseExecution(data);
  };

  return {
    async list(filters?: ListExecutionFilters): Promise<ExecutionRecord[]> {
      const query = new URLSearchParams();
      if (filters?.status) {
        query.set("status", filters.status);
      }
      if (filters?.mode) {
        query.set("mode", filters.mode);
      }
      if (filters?.backlogKind) {
        query.set("backlog_kind", filters.backlogKind);
      }
      if (filters?.backlogName) {
        query.set("backlog_name", filters.backlogName);
      }
      if (filters?.startedBy) {
        query.set("started_by", filters.startedBy);
      }
      if (filters?.createdFrom) {
        query.set("created_from", filters.createdFrom);
      }
      if (filters?.createdTo) {
        query.set("created_to", filters.createdTo);
      }
      const suffix = query.toString() ? `?${query.toString()}` : "";
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.execution}${suffix}`);
      const resp = parseProtoResponse(listExecutionResponseSchema, data, "execution list");
      return resp.items.map(mapProtoExecutionRecord);
    },

    async get(executionId: string): Promise<ExecutionRecord> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.executionById(executionId));
      return parseExecution(data);
    },

    async create(request: CreateExecutionRequest): Promise<ExecutionRecord> {
      const msg = buildMessage(CreateExecutionRequestSchema, {
        backlogKind: request.backlogKind,
        backlogName: request.backlogName,
        mode: request.mode,
        ...(request.delaySeconds !== undefined ? { delaySeconds: BigInt(request.delaySeconds) } : {}),
        ...(request.startedBy ? { startedBy: request.startedBy } : {}),
        ...(request.operation ? { operation: request.operation } : {}),
      });
      const body = toProtoJson(CreateExecutionRequestSchema, msg);
      const data = await apiClient.post<unknown>(API_ENDPOINTS.execution, body);
      return parseExecution(data);
    },

    async start(executionId: string): Promise<ExecutionRecord> {
      return mutate(API_ENDPOINTS.executionStart(executionId));
    },

    async cancel(executionId: string): Promise<ExecutionRecord> {
      return mutate(API_ENDPOINTS.executionCancel(executionId));
    },

    async retry(executionId: string): Promise<ExecutionRecord> {
      return mutate(API_ENDPOINTS.executionRetry(executionId));
    },
  };
}

export const executionService = createExecutionService();

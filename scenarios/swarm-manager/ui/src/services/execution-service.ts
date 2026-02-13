import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { BacklogKind, ExecutionMode, ExecutionRecord, ExecutionStatus } from "../types";

interface ExecutionListResponse {
  items: ExecutionRecordDTO[];
}

interface ExecutionItemResponse {
  execution: ExecutionRecordDTO;
}

interface ExecutionRecordDTO {
  execution_id: string;
  backlog_kind: BacklogKind;
  backlog_name: string;
  task_id?: string;
  run_id?: string;
  status: ExecutionStatus;
  mode: ExecutionMode;
  scheduled_at?: string;
  started_at?: string;
  finished_at?: string;
  failure_reason?: string;
  started_by?: string;
  operation?: "generator" | "improver";
  created_at: string;
  updated_at: string;
}

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

const mapRecord = (dto: ExecutionRecordDTO): ExecutionRecord => ({
  executionId: dto.execution_id,
  backlogKind: dto.backlog_kind,
  backlogName: dto.backlog_name,
  taskId: dto.task_id,
  runId: dto.run_id,
  status: dto.status,
  mode: dto.mode,
  scheduledAt: dto.scheduled_at,
  startedAt: dto.started_at,
  finishedAt: dto.finished_at,
  failureReason: dto.failure_reason,
  startedBy: dto.started_by,
  operation: dto.operation,
  createdAt: dto.created_at,
  updatedAt: dto.updated_at,
});

export function createExecutionService(apiClient: IApiClient = defaultApiClient): IExecutionService {
  const mutate = async (endpoint: string): Promise<ExecutionRecord> => {
    const data = await apiClient.post<ExecutionItemResponse>(endpoint, {});
    return mapRecord(data.execution);
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
      const data = await apiClient.get<ExecutionListResponse>(`${API_ENDPOINTS.execution}${suffix}`);
      return data.items.map(mapRecord);
    },

    async get(executionId: string): Promise<ExecutionRecord> {
      const data = await apiClient.get<ExecutionItemResponse>(API_ENDPOINTS.executionById(executionId));
      return mapRecord(data.execution);
    },

    async create(request: CreateExecutionRequest): Promise<ExecutionRecord> {
      const data = await apiClient.post<ExecutionItemResponse>(API_ENDPOINTS.execution, {
        backlog_kind: request.backlogKind,
        backlog_name: request.backlogName,
        mode: request.mode,
        delay_seconds: request.delaySeconds,
        started_by: request.startedBy,
        operation: request.operation,
      });
      return mapRecord(data.execution);
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

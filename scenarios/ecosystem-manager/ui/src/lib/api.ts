/**
 * API Client for Ecosystem Manager
 * Centralized HTTP communication with type safety.
 *
 * Uses proto-contracts for response validation and type mapping
 * at the API boundary.
 */

import { resolveApiBase } from "@vrooli/api-base";
import type {
  Task,
  TaskFilters,
  CreateTaskInput,
  UpdateTaskInput,
  TaskStatus,
  QueueStatus,
  RunningProcess,
  Settings,
  Resource,
  Scenario,
  Operation,
  Category,
  LogEntry,
  ExecutionHistory,
  ExecutionPrompt,
  ExecutionOutput,
  PromptPreviewConfig,
  PromptPreviewResult,
  PromptFileInfo,
  PromptFile,
  SkillResponse,
  SkillsSyncResult,
  AutoSteerProfile,
  AutoSteerTemplate,
  ProfilePerformance,
  ExecutionFeedbackEntry,
  ExecutionFeedbackEntryPayload,
  HealthResponse,
  AutoSteerExecutionState,
  ActiveTarget,
  Campaign,
  InsightReport,
  SystemInsightReport,
  GenerateInsightOptions,
  ApplySuggestionResult,
} from '../types/api';
import {
  parseTaskResponse,
  parseExecutionResponse,
  parseSettingsResponse,
  parseQueueStatusResponse,
  parseRunningProcessResponse,
  parseResourceResponse,
  parseScenarioResponse,
  parseActiveTargetResponse,
  mapUiSettingsToProtoJson,
} from './proto-contracts';

// API base resolution (synchronous, matches standard pattern used by all other scenarios)
const API_BASE = resolveApiBase({ appendSuffix: false });

export class ApiError extends Error {
  status: number;
  body?: string;

  constructor(status: number, message: string, body?: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.body = body;
  }
}

class ApiClient {
  /**
   * Generic fetch wrapper with error handling
   */
  private async fetchJSON<T>(url: string, options: RequestInit = {}): Promise<T> {
    const fullUrl = url.startsWith('http') ? url : `${API_BASE}${url}`;

    const response = await fetch(fullUrl, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const errorText = await response.text();
      const message = errorText
        ? `API Error (${response.status}): ${errorText}`
        : `API Error (${response.status})`;
      throw new ApiError(response.status, message, errorText);
    }

    return response.json();
  }

  // ==================== Health ====================

  async getHealth(): Promise<HealthResponse> {
    return this.fetchJSON<HealthResponse>(`/health`);
  }

  // ==================== Task Management ====================

  async getTasks(filters: TaskFilters = {}): Promise<Task[]> {
    const params = new URLSearchParams();

    if (filters.status) params.append('status', filters.status);
    if (filters.type) params.append('type', filters.type);
    if (filters.operation) params.append('operation', filters.operation);
    if (filters.priority) params.append('priority', filters.priority);
    if (filters.sort) params.append('sort', filters.sort);

    const queryString = params.toString();
    const url = `/api/tasks${queryString ? '?' + queryString : ''}`;

    const response = await this.fetchJSON<{ tasks: unknown[]; count: number }>(url);
    const tasks = response.tasks || [];
    return tasks.map(task => {
      const normalized = parseTaskResponse(task);
      // Trust the directory we fetched (query status) over stale file metadata
      if (filters.status) {
        normalized.status = filters.status as TaskStatus;
      }
      return normalized;
    });
  }

  async getTask(taskId: string): Promise<Task> {
    const raw = await this.fetchJSON<unknown>(`/api/tasks/${taskId}`);
    return parseTaskResponse(raw);
  }

  async createTask(taskData: CreateTaskInput): Promise<Task> {
    const payload: Record<string, unknown> = {
      ...taskData,
      processor_auto_requeue: taskData.auto_requeue,
    };

    // Normalize target payloads for API compatibility
    if (Array.isArray(taskData.target)) {
      payload.targets = taskData.target;
      payload.target = taskData.target[0] ?? '';
    } else if (taskData.target) {
      payload.target = taskData.target;
    }

    delete (payload as any).auto_requeue;

    const raw = await this.fetchJSON<unknown>(`/api/tasks`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return parseTaskResponse(raw);
  }

  async updateTask(taskId: string, updates: UpdateTaskInput): Promise<Task> {
    const payload: Record<string, unknown> = {
      ...updates,
      processor_auto_requeue: updates.auto_requeue,
    };

    if (Array.isArray((updates as any).target)) {
      payload.targets = (updates as any).target;
      payload.target = (updates as any).target[0] ?? '';
    }

    delete (payload as any).auto_requeue;

    const response = await this.fetchJSON<Record<string, unknown>>(`/api/tasks/${taskId}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });

    const raw = (response as any)?.task ?? response;
    return parseTaskResponse(raw);
  }

  async updateTaskStatus(
    taskId: string,
    status: TaskStatus,
    additionalData: Record<string, unknown> = {}
  ): Promise<Task> {
    const raw = await this.fetchJSON<unknown>(`/api/tasks/${taskId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, ...additionalData }),
    });
    return parseTaskResponse(raw);
  }

  async setQueuePosition(
    taskId: string,
    position: number
  ): Promise<{ success: boolean; position: number; mode: string }> {
    return this.fetchJSON<{ success: boolean; position: number; mode: string }>(
      `/api/tasks/${taskId}/queue-position`,
      {
        method: 'PUT',
        body: JSON.stringify({ position }),
      }
    );
  }

  async deleteTask(taskId: string): Promise<void> {
    return this.fetchJSON<void>(`/api/tasks/${taskId}`, {
      method: 'DELETE',
    });
  }

  async getTaskLogs(taskId: string): Promise<LogEntry[]> {
    const response = await this.fetchJSON<LogEntry[] | { entries?: LogEntry[] }>(
      `/api/tasks/${taskId}/logs`,
    );

    if (Array.isArray(response)) {
      return response;
    }

    if (response && Array.isArray((response as any).entries)) {
      return (response as any).entries as LogEntry[];
    }

    return [];
  }

  async getTaskPrompt(taskId: string): Promise<Record<string, unknown>> {
    return this.fetchJSON<Record<string, unknown>>(`/api/tasks/${taskId}/prompt`);
  }

  async getAssembledPrompt(taskId: string): Promise<string> {
    const response = await this.fetchJSON<string | { prompt?: string }>(
      `/api/tasks/${taskId}/prompt/assembled`,
    );

    if (typeof response === 'string') {
      return response;
    }

    if (response && typeof (response as any).prompt === 'string') {
      return (response as any).prompt as string;
    }

    return JSON.stringify(response ?? {}, null, 2);
  }

  async getActiveTargets(type?: string, operation?: string): Promise<ActiveTarget[]> {
    const params = new URLSearchParams();
    if (type) params.append('type', type);
    if (operation) params.append('operation', operation);

    const response = await this.fetchJSON<unknown[]>(
      `/api/tasks/active-targets?${params.toString()}`
    );

    return (Array.isArray(response) ? response : [])
      .map(parseActiveTargetResponse)
      .filter((entry): entry is ActiveTarget => entry !== null);
  }

  // ==================== Queue Management ====================

  async getQueueStatus(): Promise<QueueStatus> {
    const raw = await this.fetchJSON<unknown>(`/api/queue/status`);
    return parseQueueStatusResponse(raw);
  }

  async triggerQueueProcessing(): Promise<void> {
    return this.fetchJSON<void>(`/api/queue/trigger`, {
      method: 'POST',
    });
  }

  async startQueueProcessor(): Promise<void> {
    return this.fetchJSON<void>(`/api/queue/start`, {
      method: 'POST',
    });
  }

  async stopQueueProcessor(): Promise<void> {
    return this.fetchJSON<void>(`/api/queue/stop`, {
      method: 'POST',
    });
  }

  async resetRateLimit(): Promise<void> {
    return this.fetchJSON<void>(`/api/queue/reset-rate-limit`, {
      method: 'POST',
    });
  }

  async getRunningProcesses(): Promise<RunningProcess[]> {
    const response = await this.fetchJSON<{ processes: unknown[]; count: number }>(`/api/processes/running`);
    const processes = response.processes || [];
    return processes.map(parseRunningProcessResponse);
  }

  async terminateProcess(taskId: string): Promise<void> {
    return this.fetchJSON<void>(`/api/queue/processes/terminate`, {
      method: 'POST',
      body: JSON.stringify({ task_id: taskId }),
    });
  }

  // ==================== Settings Management ====================

  async getSettings(): Promise<Settings> {
    const raw = await this.fetchJSON<unknown>(`/api/settings`);
    return parseSettingsResponse(raw);
  }

  async updateSettings(settings: Settings): Promise<Settings> {
    const payload = mapUiSettingsToProtoJson(settings);
    const raw = await this.fetchJSON<unknown>(`/api/settings`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
    return parseSettingsResponse(raw);
  }

  async resetSettings(): Promise<Settings> {
    const raw = await this.fetchJSON<unknown>(`/api/settings/reset`, {
      method: 'POST',
    });
    return parseSettingsResponse(raw);
  }

  async getRecyclerModels(provider: string): Promise<string[]> {
    const params = new URLSearchParams({ provider });
    return this.fetchJSON<string[]>(`/api/settings/recycler/models?${params.toString()}`);
  }

  // ==================== Discovery ====================

  async getResources(): Promise<Resource[]> {
    const raw = await this.fetchJSON<unknown>(`/api/resources`);
    const wrapper = raw as Record<string, unknown> | unknown[];
    const list: unknown[] = Array.isArray(wrapper) ? wrapper : Array.isArray((wrapper as Record<string, unknown>)?.resources) ? (wrapper as Record<string, unknown>).resources as unknown[] : [];
    return list.map(parseResourceResponse);
  }

  async getScenarios(): Promise<Scenario[]> {
    const raw = await this.fetchJSON<unknown>(`/api/scenarios`);
    const wrapper = raw as Record<string, unknown> | unknown[];
    const list: unknown[] = Array.isArray(wrapper) ? wrapper : Array.isArray((wrapper as Record<string, unknown>)?.scenarios) ? (wrapper as Record<string, unknown>).scenarios as unknown[] : [];
    return list.map(parseScenarioResponse);
  }

  async getResourceStatus(resourceName: string): Promise<Resource> {
    return this.fetchJSON<Resource>(`/api/resources/${resourceName}/status`);
  }

  async getScenarioStatus(scenarioName: string): Promise<Scenario> {
    return this.fetchJSON<Scenario>(`/api/scenarios/${scenarioName}/status`);
  }

  async getOperations(): Promise<Operation[]> {
    return this.fetchJSON<Operation[]>(`/api/operations`);
  }

  async getCategories(): Promise<Category[]> {
    return this.fetchJSON<Category[]>(`/api/categories`);
  }

  // ==================== Logs ====================

  async getSystemLogs(limit = 500): Promise<LogEntry[]> {
    const response = await this.fetchJSON<LogEntry[] | { entries?: LogEntry[] }>(
      `/api/logs?limit=${limit}`,
    );

    if (Array.isArray(response)) {
      return response;
    }

    if (response && Array.isArray((response as any).entries)) {
      return (response as any).entries as LogEntry[];
    }

    return [];
  }

  // ==================== Execution History ====================

  async getAllExecutionHistory(): Promise<ExecutionHistory[]> {
    const response = await this.fetchJSON<unknown[] | { executions?: unknown[] }>(
      `/api/executions`,
    );

    const list = Array.isArray(response)
      ? response
      : Array.isArray((response as any)?.executions)
        ? (response as any).executions
        : [];

    return list.map(parseExecutionResponse);
  }

  async getExecutionHistory(taskId: string): Promise<ExecutionHistory[]> {
    const response = await this.fetchJSON<unknown[] | { executions?: unknown[] }>(
      `/api/tasks/${taskId}/executions`,
    );

    const list = Array.isArray(response)
      ? response
      : Array.isArray((response as any)?.executions)
        ? (response as any).executions
        : [];

    return list.map(parseExecutionResponse);
  }

  async getExecutionPrompt(taskId: string, executionId: string): Promise<ExecutionPrompt> {
    const response = await this.fetchJSON<any>(
      `/api/tasks/${taskId}/executions/${executionId}/prompt`
    );
    const prompt = response?.prompt ?? response?.content ?? '';
    return {
      ...response,
      prompt,
      content: prompt,
    };
  }

  async getExecutionOutput(taskId: string, executionId: string): Promise<ExecutionOutput> {
    const response = await this.fetchJSON<any>(
      `/api/tasks/${taskId}/executions/${executionId}/output`
    );
    const output = response?.output ?? response?.content ?? '';
    return {
      ...response,
      output,
      content: output,
    };
  }

  async getExecutionMetadata(taskId: string, executionId: string): Promise<Record<string, unknown>> {
    return this.fetchJSON<Record<string, unknown>>(
      `/api/tasks/${taskId}/executions/${executionId}/metadata`
    );
  }

  async getPromptPreview(taskConfig: PromptPreviewConfig): Promise<PromptPreviewResult> {
    return this.fetchJSON<PromptPreviewResult>(`/api/prompt-viewer`, {
      method: 'POST',
      body: JSON.stringify(taskConfig),
    });
  }

  // Alias for hook compatibility
  async previewPrompt(taskConfig: PromptPreviewConfig): Promise<PromptPreviewResult> {
    return this.getPromptPreview(taskConfig);
  }

  // ==================== Prompt Files ====================

  async listPromptFiles(): Promise<PromptFileInfo[]> {
    return this.fetchJSON<PromptFileInfo[]>(`/api/prompts`);
  }

  async getPromptFile(id: string): Promise<PromptFile> {
    const path = this.encodePromptPath(id);
    return this.fetchJSON<PromptFile>(`/api/prompts/${path}`);
  }

  async updatePromptFile(id: string, content: string): Promise<PromptFile> {
    const path = this.encodePromptPath(id);
    return this.fetchJSON<PromptFile>(`/api/prompts/${path}`, {
      method: 'PUT',
      body: JSON.stringify({ content }),
    });
  }

  async createPromptFile(path: string, content: string): Promise<PromptFile> {
    return this.fetchJSON<PromptFile>(`/api/prompts`, {
      method: 'POST',
      body: JSON.stringify({ path, content }),
    });
  }

  // ==================== Skills (from prompt-manager) ====================

  async listSkills(): Promise<SkillResponse[]> {
    return this.fetchJSON<SkillResponse[]>(`/api/skills`);
  }

  async syncSkills(): Promise<SkillsSyncResult> {
    return this.fetchJSON<SkillsSyncResult>(`/api/skills/sync`, {
      method: 'POST',
    });
  }

  // ==================== Auto Steer ====================

  async getAutoSteerProfiles(): Promise<AutoSteerProfile[]> {
    return this.fetchJSON<AutoSteerProfile[]>(`/api/auto-steer/profiles`);
  }

  async createAutoSteerProfile(profile: Omit<AutoSteerProfile, 'id' | 'created_at' | 'updated_at'>): Promise<AutoSteerProfile> {
    return this.fetchJSON<AutoSteerProfile>(`/api/auto-steer/profiles`, {
      method: 'POST',
      body: JSON.stringify(profile),
    });
  }

  async getAutoSteerProfile(id: string): Promise<AutoSteerProfile> {
    return this.fetchJSON<AutoSteerProfile>(`/api/auto-steer/profiles/${id}`);
  }

  async getAutoSteerExecutionState(taskId: string): Promise<AutoSteerExecutionState> {
    return this.fetchJSON<AutoSteerExecutionState>(`/api/auto-steer/execution/${taskId}`);
  }

  async updateAutoSteerProfile(id: string, profile: Partial<AutoSteerProfile>): Promise<AutoSteerProfile> {
    return this.fetchJSON<AutoSteerProfile>(`/api/auto-steer/profiles/${id}`, {
      method: 'PUT',
      body: JSON.stringify(profile),
    });
  }

  async deleteAutoSteerProfile(id: string): Promise<void> {
    return this.fetchJSON<void>(`/api/auto-steer/profiles/${id}`, {
      method: 'DELETE',
    });
  }

  async getAutoSteerTemplates(): Promise<AutoSteerTemplate[]> {
    return this.fetchJSON<AutoSteerTemplate[]>(`/api/auto-steer/templates`);
  }

  async getAutoSteerHistory(filters: { profile_id?: string; scenario?: string } = {}): Promise<ProfilePerformance[]> {
    const params = new URLSearchParams();
    if (filters.profile_id) params.append('profile_id', filters.profile_id);
    if (filters.scenario) params.append('scenario', filters.scenario);

    const query = params.toString();
    const url = query ? `/api/auto-steer/history?${query}` : `/api/auto-steer/history`;

    const response = await this.fetchJSON<ProfilePerformance[] | { history?: ProfilePerformance[] }>(url);

    if (Array.isArray(response)) {
      return response;
    }

    if (response && Array.isArray((response as any).history)) {
      return (response as any).history as ProfilePerformance[];
    }

    return [];
  }

  async getAutoSteerExecution(executionId: string): Promise<ProfilePerformance> {
    return this.fetchJSON<ProfilePerformance>(`/api/auto-steer/history/${executionId}`);
  }

  async submitExecutionFeedbackEntry(
    executionId: string,
    payload: ExecutionFeedbackEntryPayload
  ): Promise<ExecutionFeedbackEntry> {
    return this.fetchJSON<ExecutionFeedbackEntry>(`/api/auto-steer/history/${executionId}/feedback/entries`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async resetAutoSteerExecution(taskId: string): Promise<{ success: boolean; message?: string }> {
    return this.fetchJSON<{ success: boolean; message?: string }>(`/api/auto-steer/execution/reset`, {
      method: 'POST',
      body: JSON.stringify({ task_id: taskId }),
    });
  }

  async seekAutoSteerExecution(
    taskId: string,
    phaseIndex: number,
    phaseIteration: number,
    profileId?: string,
    scenarioName?: string
  ): Promise<AutoSteerExecutionState> {
    return this.fetchJSON<AutoSteerExecutionState>(`/api/auto-steer/execution/seek`, {
      method: 'POST',
      body: JSON.stringify({
        task_id: taskId,
        phase_index: phaseIndex,
        phase_iteration: phaseIteration,
        ...(profileId && { profile_id: profileId }),
        ...(scenarioName && { scenario_name: scenarioName }),
      }),
    });
  }

  // ==================== Utilities ====================

  private encodePromptPath(id: string): string {
    return id
      .split('/')
      .map((segment) => encodeURIComponent(segment))
      .join('/');
  }

  // ==================== Maintenance ====================

  async setMaintenanceState(active: boolean): Promise<void> {
    return this.fetchJSON<void>(`/api/maintenance/state`, {
      method: 'POST',
      body: JSON.stringify({ active }),
    });
  }

  // ==================== Visited Tracker Integration ====================

  async getVisitedTrackerUIPort(): Promise<{ port: string; url: string }> {
    return this.fetchJSON<{ port: string; url: string }>(`/api/visited-tracker/ui-port`);
  }

  async getCampaignsForTarget(target: string): Promise<Campaign[]> {
    const params = new URLSearchParams({ target });
    return this.fetchJSON<Campaign[]>(`/api/visited-tracker/campaigns/by-target?${params.toString()}`);
  }

  async getCampaign(campaignId: string): Promise<Campaign> {
    return this.fetchJSON<Campaign>(`/api/visited-tracker/campaigns/${campaignId}`);
  }

  async deleteCampaign(campaignId: string): Promise<void> {
    return this.fetchJSON<void>(`/api/visited-tracker/campaigns/${campaignId}`, {
      method: 'DELETE',
    });
  }

  async resetCampaign(campaignId: string): Promise<void> {
    return this.fetchJSON<void>(`/api/visited-tracker/campaigns/${campaignId}/reset`, {
      method: 'POST',
    });
  }

  // ==================== Insights ====================

  async getTaskInsights(taskId: string): Promise<InsightReport[]> {
    const response = await this.fetchJSON<{ insights: InsightReport[]; count: number }>(
      `/api/tasks/${taskId}/insights`
    );
    return response.insights || [];
  }

  async getInsightReport(taskId: string, reportId: string): Promise<InsightReport> {
    return this.fetchJSON<InsightReport>(`/api/tasks/${taskId}/insights/${reportId}`);
  }

  async generateInsightReport(
    taskId: string,
    options: GenerateInsightOptions = {}
  ): Promise<void> {
    const params = new URLSearchParams();
    if (options.limit) params.append('limit', options.limit.toString());
    if (options.status_filter) params.append('status_filter', options.status_filter);
    if (options.include_files) params.append('include_files', options.include_files.join(','));

    const queryString = params.toString();
    const url = `/api/tasks/${taskId}/insights/generate${queryString ? '?' + queryString : ''}`;

    return this.fetchJSON<void>(url, {
      method: 'POST',
    });
  }

  async applySuggestion(
    taskId: string,
    reportId: string,
    suggestionId: string
  ): Promise<ApplySuggestionResult> {
    return this.fetchJSON<ApplySuggestionResult>(
      `/api/tasks/${taskId}/insights/${reportId}/suggestions/${suggestionId}/apply`,
      {
        method: 'POST',
      }
    );
  }

  async getSystemInsights(sinceDays: number = 7): Promise<SystemInsightReport> {
    const params = new URLSearchParams();
    params.append('since_days', sinceDays.toString());

    return this.fetchJSON<SystemInsightReport>(
      `/api/insights/system?${params.toString()}`
    );
  }

  async generateSystemInsights(sinceDays: number = 7): Promise<SystemInsightReport> {
    const params = new URLSearchParams();
    params.append('since_days', sinceDays.toString());

    const response = await this.fetchJSON<{ report: SystemInsightReport }>(
      `/api/insights/system/generate?${params.toString()}`,
      {
        method: 'POST',
      }
    );
    return response.report;
  }

  async previewInsightPrompt(
    taskId: string,
    options: GenerateInsightOptions = {}
  ): Promise<{ prompt: string; task_id: string; status_filter: string; limit: number; executions: number }> {
    const params = new URLSearchParams();
    if (options.limit) params.append('limit', options.limit.toString());
    if (options.status_filter) params.append('status_filter', options.status_filter);
    if (options.include_files) params.append('include_files', options.include_files.join(','));

    const queryString = params.toString();
    const url = `/api/tasks/${taskId}/insights/preview${queryString ? '?' + queryString : ''}`;

    return await this.fetchJSON<{ prompt: string; task_id: string; status_filter: string; limit: number; executions: number }>(url);
  }

  async generateInsightReportWithPrompt(
    taskId: string,
    options: GenerateInsightOptions & { custom_prompt: string }
  ): Promise<void> {
    const params = new URLSearchParams();
    if (options.limit) params.append('limit', options.limit.toString());
    if (options.status_filter) params.append('status_filter', options.status_filter);
    if (options.include_files) params.append('include_files', options.include_files.join(','));

    const queryString = params.toString();
    const url = `/api/tasks/${taskId}/insights/generate${queryString ? '?' + queryString : ''}`;

    return this.fetchJSON<void>(url, {
      method: 'POST',
      body: JSON.stringify({ custom_prompt: options.custom_prompt }),
    });
  }

  // ==================== Aliases for consistency ====================

  // Queue management aliases (for hook compatibility)
  async startProcessor(): Promise<void> {
    return this.startQueueProcessor();
  }

  async stopProcessor(): Promise<void> {
    return this.stopQueueProcessor();
  }

  async triggerQueue(): Promise<void> {
    return this.triggerQueueProcessing();
  }
}

// Export singleton instance
export const api = new ApiClient();

// Export for testing or custom instances
export { ApiClient };

// Legacy compatibility
export async function fetchHealth(): Promise<HealthResponse> {
  return api.getHealth();
}

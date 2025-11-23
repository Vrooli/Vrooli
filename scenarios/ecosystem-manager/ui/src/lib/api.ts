/**
 * API Client for Ecosystem Manager
 * Centralized HTTP communication with type safety
 */

import { resolveWithConfig } from "@vrooli/api-base";
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
  RecyclerTestPayload,
  RecyclerTestResult,
  PromptPreviewConfig,
  PromptPreviewResult,
  AutoSteerProfile,
  AutoSteerTemplate,
  HealthResponse,
} from '../types/api';

// API base resolution (async, uses /config endpoint)
let apiBasePromise: Promise<string> | null = null;

function getApiBase(): Promise<string> {
  if (!apiBasePromise) {
    apiBasePromise = resolveWithConfig({
      appendSuffix: false,
      configEndpoint: '/config'
    });
  }
  return apiBasePromise;
}

class ApiClient {
  private apiBase: string | null = null;

  async ensureApiBase(): Promise<string> {
    if (!this.apiBase) {
      this.apiBase = await getApiBase();
    }
    return this.apiBase;
  }

  /**
   * Generic fetch wrapper with error handling
   */
  private async fetchJSON<T>(url: string, options: RequestInit = {}): Promise<T> {
    const baseUrl = await this.ensureApiBase();
    const fullUrl = url.startsWith('http') ? url : `${baseUrl}${url}`;

    const response = await fetch(fullUrl, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`API Error (${response.status}): ${errorText}`);
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

    const queryString = params.toString();
    const url = `/api/tasks${queryString ? '?' + queryString : ''}`;

    const response = await this.fetchJSON<{ tasks: Task[]; count: number }>(url);
    return response.tasks || [];
  }

  async getTask(taskId: string): Promise<Task> {
    return this.fetchJSON<Task>(`/api/tasks/${taskId}`);
  }

  async createTask(taskData: CreateTaskInput): Promise<Task> {
    return this.fetchJSON<Task>(`/api/tasks`, {
      method: 'POST',
      body: JSON.stringify(taskData),
    });
  }

  async updateTask(taskId: string, updates: UpdateTaskInput): Promise<Task> {
    return this.fetchJSON<Task>(`/api/tasks/${taskId}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async updateTaskStatus(
    taskId: string,
    status: TaskStatus,
    additionalData: Record<string, unknown> = {}
  ): Promise<Task> {
    return this.fetchJSON<Task>(`/api/tasks/${taskId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, ...additionalData }),
    });
  }

  async deleteTask(taskId: string): Promise<void> {
    return this.fetchJSON<void>(`/api/tasks/${taskId}`, {
      method: 'DELETE',
    });
  }

  async getTaskLogs(taskId: string): Promise<LogEntry[]> {
    return this.fetchJSON<LogEntry[]>(`/api/tasks/${taskId}/logs`);
  }

  async getTaskPrompt(taskId: string): Promise<Record<string, unknown>> {
    return this.fetchJSON<Record<string, unknown>>(`/api/tasks/${taskId}/prompt`);
  }

  async getAssembledPrompt(taskId: string): Promise<string> {
    return this.fetchJSON<string>(`/api/tasks/${taskId}/prompt/assembled`);
  }

  async getActiveTargets(type?: string, operation?: string): Promise<string[]> {
    const params = new URLSearchParams();
    if (type) params.append('type', type);
    if (operation) params.append('operation', operation);

    return this.fetchJSON<string[]>(`/api/tasks/active-targets?${params.toString()}`);
  }

  // ==================== Queue Management ====================

  async getQueueStatus(): Promise<QueueStatus> {
    return this.fetchJSON<QueueStatus>(`/api/queue/status`);
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
    const response = await this.fetchJSON<{ processes: RunningProcess[]; count: number }>(`/api/processes/running`);
    return response.processes || [];
  }

  async terminateProcess(taskId: string): Promise<void> {
    return this.fetchJSON<void>(`/api/queue/processes/terminate`, {
      method: 'POST',
      body: JSON.stringify({ task_id: taskId }),
    });
  }

  // ==================== Settings Management ====================

  async getSettings(): Promise<Settings> {
    return this.fetchJSON<Settings>(`/api/settings`);
  }

  async updateSettings(settings: Settings): Promise<Settings> {
    return this.fetchJSON<Settings>(`/api/settings`, {
      method: 'PUT',
      body: JSON.stringify(settings),
    });
  }

  async resetSettings(): Promise<Settings> {
    return this.fetchJSON<Settings>(`/api/settings/reset`, {
      method: 'POST',
    });
  }

  async getRecyclerModels(provider: string): Promise<string[]> {
    const params = new URLSearchParams({ provider });
    return this.fetchJSON<string[]>(`/api/settings/recycler/models?${params.toString()}`);
  }

  // ==================== Discovery ====================

  async getResources(): Promise<Resource[]> {
    return this.fetchJSON<Resource[]>(`/api/resources`);
  }

  async getScenarios(): Promise<Scenario[]> {
    return this.fetchJSON<Scenario[]>(`/api/scenarios`);
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
    return this.fetchJSON<LogEntry[]>(`/api/logs?limit=${limit}`);
  }

  // ==================== Execution History ====================

  async getAllExecutionHistory(): Promise<ExecutionHistory[]> {
    return this.fetchJSON<ExecutionHistory[]>(`/api/executions`);
  }

  async getExecutionHistory(taskId: string): Promise<ExecutionHistory[]> {
    return this.fetchJSON<ExecutionHistory[]>(`/api/tasks/${taskId}/executions`);
  }

  async getExecutionPrompt(taskId: string, executionId: string): Promise<ExecutionPrompt> {
    return this.fetchJSON<ExecutionPrompt>(
      `/api/tasks/${taskId}/executions/${executionId}/prompt`
    );
  }

  async getExecutionOutput(taskId: string, executionId: string): Promise<ExecutionOutput> {
    return this.fetchJSON<ExecutionOutput>(
      `/api/tasks/${taskId}/executions/${executionId}/output`
    );
  }

  async getExecutionMetadata(taskId: string, executionId: string): Promise<Record<string, unknown>> {
    return this.fetchJSON<Record<string, unknown>>(
      `/api/tasks/${taskId}/executions/${executionId}/metadata`
    );
  }

  // ==================== Recycler & Testing ====================

  async testRecycler(payload: RecyclerTestPayload): Promise<RecyclerTestResult> {
    return this.fetchJSON<RecyclerTestResult>(`/api/recycler/test`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  // Alias with simplified input for hook compatibility
  async testRecyclerOutput(data: { output: string }): Promise<{
    should_recycle: boolean;
    completion_score: number;
    confidence?: number;
    reasoning?: string;
  }> {
    const result = await this.testRecycler({ output_text: data.output });
    return {
      should_recycle: result.suggested_status !== 'completed-finalized',
      completion_score: result.confidence * 10, // Convert 0-1 to 0-10
      confidence: result.confidence,
      reasoning: result.reasoning,
    };
  }

  async getRecyclerPromptPreview(outputText: string): Promise<string> {
    return this.fetchJSON<string>(`/api/recycler/preview-prompt`, {
      method: 'POST',
      body: JSON.stringify({ output_text: outputText }),
    });
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

  // ==================== Maintenance ====================

  async setMaintenanceState(active: boolean): Promise<void> {
    return this.fetchJSON<void>(`/api/maintenance/state`, {
      method: 'POST',
      body: JSON.stringify({ active }),
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

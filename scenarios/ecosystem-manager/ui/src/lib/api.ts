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
  PromptPreviewConfig,
  PromptPreviewResult,
  PromptFileInfo,
  PromptFile,
  PhaseInfo,
  AutoSteerProfile,
  AutoSteerTemplate,
  ProfilePerformance,
  HealthResponse,
  AutoSteerExecutionState,
  ActiveTarget,
  Campaign,
  InsightReport,
  SystemInsightReport,
  GenerateInsightOptions,
  ApplySuggestionResult,
} from '../types/api';

// Default settings fallback (matches legacy API defaults)
const DEFAULT_SETTINGS: Settings = {
  processor: {
    concurrent_slots: 1,
    cooldown_seconds: 30,
    active: false,
  },
  agent: {
    max_turns: 60,
    allowed_tools: 'Read,Write,Edit,Bash,LS,Glob,Grep',
    skip_permissions: true,
    task_timeout_minutes: 30,
    idle_timeout_cap_minutes: 30,
  },
  display: {
    theme: 'dark',
    condensed_mode: false,
  },
  recycler: {
    enabled_for: 'off',
    recycle_interval: 60,
    max_retries: 3,
    retry_delay_seconds: 2,
    model_provider: 'ollama',
    model_name: 'llama3.1:8b',
    completion_threshold: 3,
    failure_threshold: 5,
  },
};

type ApiSettingsPayload = {
  theme?: string;
  slots?: number;
  refresh_interval?: number; // legacy
  cooldown_seconds?: number;
  active?: boolean;
  max_turns?: number;
  allowed_tools?: string;
  skip_permissions?: boolean;
  task_timeout?: number;
  idle_timeout_cap?: number;
  condensed_mode?: boolean;
  recycler?: {
    enabled_for?: string;
    interval_seconds?: number;
    max_retries?: number;
    retry_delay_seconds?: number;
    model_provider?: string;
    model_name?: string;
    completion_threshold?: number;
    failure_threshold?: number;
  };
};

const isTheme = (value: unknown): value is Settings['display']['theme'] =>
  value === 'light' || value === 'dark' || value === 'auto';

const isEnabledFor = (value: unknown): value is Settings['recycler']['enabled_for'] =>
  value === 'off' || value === 'resources' || value === 'scenarios' || value === 'both';

const isModelProvider = (value: unknown): value is Settings['recycler']['model_provider'] =>
  value === 'ollama' || value === 'openrouter';

function mapApiSettingsToUi(raw: ApiSettingsPayload | Settings | { settings?: ApiSettingsPayload }): Settings {
  // Handle nested response from GET /api/settings (has "settings" wrapper)
  const source = (raw as any)?.settings ?? raw ?? {};
  const recycler = (source as ApiSettingsPayload)?.recycler ?? {};
  const theme = (source as ApiSettingsPayload)?.theme;
  const condensed = (source as ApiSettingsPayload)?.condensed_mode;
  const cooldown =
    (source as ApiSettingsPayload)?.cooldown_seconds ??
    (source as ApiSettingsPayload)?.refresh_interval ??
    DEFAULT_SETTINGS.processor.cooldown_seconds;

  return {
    processor: {
      concurrent_slots:
        (source as ApiSettingsPayload)?.slots ?? DEFAULT_SETTINGS.processor.concurrent_slots,
      cooldown_seconds: cooldown,
      active: (source as ApiSettingsPayload)?.active ?? DEFAULT_SETTINGS.processor.active,
    },
    agent: {
      max_turns: (source as ApiSettingsPayload)?.max_turns ?? DEFAULT_SETTINGS.agent.max_turns,
      allowed_tools:
        (source as ApiSettingsPayload)?.allowed_tools ?? DEFAULT_SETTINGS.agent.allowed_tools,
      skip_permissions:
        (source as ApiSettingsPayload)?.skip_permissions ?? DEFAULT_SETTINGS.agent.skip_permissions,
      task_timeout_minutes:
        (source as ApiSettingsPayload)?.task_timeout ?? DEFAULT_SETTINGS.agent.task_timeout_minutes,
      idle_timeout_cap_minutes:
        (source as ApiSettingsPayload)?.idle_timeout_cap ??
        DEFAULT_SETTINGS.agent.idle_timeout_cap_minutes,
    },
    display: {
      theme: isTheme(theme) ? theme : DEFAULT_SETTINGS.display.theme,
      condensed_mode: condensed ?? DEFAULT_SETTINGS.display.condensed_mode,
    },
    recycler: {
      enabled_for: isEnabledFor(recycler?.enabled_for)
        ? recycler.enabled_for
        : DEFAULT_SETTINGS.recycler.enabled_for,
      recycle_interval:
        recycler?.interval_seconds ??
        // Allow already-converted value to pass through for safety
        (recycler as unknown as Settings['recycler'])?.recycle_interval ??
        DEFAULT_SETTINGS.recycler.recycle_interval,
      max_retries:
        typeof recycler?.max_retries === 'number'
          ? recycler.max_retries
          : (recycler as any)?.max_retries ??
            (recycler as unknown as Settings['recycler'])?.max_retries ??
            DEFAULT_SETTINGS.recycler.max_retries,
      retry_delay_seconds:
        typeof recycler?.retry_delay_seconds === 'number'
          ? recycler.retry_delay_seconds
          : (recycler as any)?.retry_delay_seconds ??
            (recycler as unknown as Settings['recycler'])?.retry_delay_seconds ??
            DEFAULT_SETTINGS.recycler.retry_delay_seconds,
      model_provider: isModelProvider(recycler?.model_provider)
        ? recycler.model_provider
        : DEFAULT_SETTINGS.recycler.model_provider,
      model_name: recycler?.model_name ?? DEFAULT_SETTINGS.recycler.model_name,
      completion_threshold:
        recycler?.completion_threshold ?? DEFAULT_SETTINGS.recycler.completion_threshold,
      failure_threshold:
        recycler?.failure_threshold ?? DEFAULT_SETTINGS.recycler.failure_threshold,
    },
  };
}

// Normalize legacy target fields (string/array) into a consistent array
const normalizeTaskTargets = (task: any): Task => {
  const rawTargets = Array.isArray(task.target)
    ? task.target
    : Array.isArray(task.targets)
      ? task.targets
      : task.target
        ? [task.target]
        : [];

  const currentProcess = task.current_process || task.process || task.running_process;
  const normalizedProcess = currentProcess
    ? normalizeRunningProcess({
        ...currentProcess,
        task_id: currentProcess.task_id ?? task.id ?? task.task_id,
        task_title: currentProcess.task_title ?? task.title,
      })
    : undefined;
  const completionCount =
    typeof task.completion_count === 'number'
      ? task.completion_count
      : typeof task.completionCount === 'number'
        ? task.completionCount
        : 0;
  const lastCompletedAt = task.last_completed_at ?? task.lastCompletedAt ?? '';
  const cooldownUntil = task.cooldown_until ?? task.cooldownUntil ?? '';
  const autoSteerMode =
    task.auto_steer_mode ??
    task.autoSteerMode ??
    task.AutoSteerMode;
  const autoRequeue =
    typeof task.auto_requeue === 'boolean'
      ? task.auto_requeue
      : typeof task.processor_auto_requeue === 'boolean'
        ? task.processor_auto_requeue
        : true;

  return {
    ...task,
    target: rawTargets.filter(Boolean),
    current_process: normalizedProcess,
    completion_count: completionCount,
    last_completed_at: lastCompletedAt,
    cooldown_until: cooldownUntil,
    // Backend uses processor_auto_requeue; UI expects auto_requeue
    auto_requeue: autoRequeue,
    auto_steer_mode: autoSteerMode,
  };
};

const normalizeExecution = (raw: any): ExecutionHistory => {
  const id =
    raw?.id ??
    raw?.execution_id ??
    raw?.executionId ??
    raw?.executionID ??
    raw?.ExecutionID ??
    raw?.task_execution_id ??
    raw?.taskExecutionId ??
    raw?.taskExecutionID ??
    raw?.start_time ??
    '';

  const startTime =
    raw?.start_time ??
    raw?.startTime ??
    raw?.StartTime ??
    '';

  const endTime =
    raw?.end_time ??
    raw?.endTime ??
    raw?.EndTime ??
    undefined;

  const duration =
    raw?.duration ??
    raw?.Duration ??
    raw?.execution_time ??
    raw?.ExecutionTime ??
    undefined;

  const timeoutAllowed =
    raw?.timeout_allowed ??
    raw?.TimeoutAllowed ??
    raw?.timeout ??
    raw?.Timeout ??
    undefined;

  const exitReason = raw?.exit_reason ?? raw?.exitReason ?? raw?.ExitReason ?? undefined;
  const rateLimited = raw?.rate_limited ?? raw?.RateLimited ?? exitReason === 'rate_limited';
  const success = raw?.success ?? raw?.Success;
  const rawStatus = typeof raw?.status === 'string' ? raw.status.toLowerCase() : raw?.status;

  const status: ExecutionHistory['status'] =
    rawStatus === 'rate_limited' || rateLimited
      ? 'rate_limited'
      : rawStatus === 'completed' || success === true
        ? 'completed'
        : rawStatus === 'failed' || success === false || raw?.failed
          ? 'failed'
          : 'running';

  return {
    id: String(id),
    task_id: raw?.task_id ?? raw?.taskId ?? raw?.TaskID ?? raw?.task ?? '',
    task_title: raw?.task_title ?? raw?.taskTitle ?? raw?.TaskTitle,
    task_type: raw?.task_type ?? raw?.taskType ?? raw?.TaskType,
    task_operation: raw?.task_operation ?? raw?.taskOperation ?? raw?.TaskOperation,
    agent_tag: raw?.agent_tag ?? raw?.agentTag ?? raw?.AgentTag,
    process_id: raw?.process_id ?? raw?.processId ?? raw?.ProcessID,
    start_time: startTime,
    end_time: endTime,
    duration,
    status,
    exit_code: raw?.exit_code ?? raw?.exitCode ?? raw?.ExitCode,
    exit_reason: exitReason,
    prompt_size: raw?.prompt_size ?? raw?.PromptSize,
    prompt_path: raw?.prompt_path ?? raw?.PromptPath ?? raw?.promptPath,
    output_path: raw?.output_path ?? raw?.OutputPath ?? raw?.outputPath,
    clean_output_path: raw?.clean_output_path ?? raw?.CleanOutputPath ?? raw?.cleanOutputPath,
    last_message_path: raw?.last_message_path ?? raw?.LastMessagePath ?? raw?.lastMessagePath,
    transcript_path: raw?.transcript_path ?? raw?.TranscriptPath ?? raw?.transcriptPath,
    auto_steer_profile_id: raw?.auto_steer_profile_id ?? raw?.autoSteerProfileId ?? raw?.AutoSteerProfileID,
    auto_steer_iteration: raw?.auto_steer_iteration ?? raw?.autoSteerIteration ?? raw?.AutoSteerIteration,
    steer_mode: raw?.steer_mode ?? raw?.steerMode ?? raw?.SteerMode,
    steer_phase_index: raw?.steer_phase_index ?? raw?.steerPhaseIndex ?? raw?.SteerPhaseIndex,
    steer_phase_iteration: raw?.steer_phase_iteration ?? raw?.steerPhaseIteration ?? raw?.SteerPhaseIteration,
    steering_source: raw?.steering_source ?? raw?.steeringSource ?? raw?.SteeringSource,
    timeout_allowed: timeoutAllowed,
    rate_limited: rateLimited,
    retry_after: raw?.retry_after ?? raw?.RetryAfter,
    success,
    metadata: raw?.metadata,
  };
};

const normalizeQueueStatus = (raw: any): QueueStatus => {
  const rateInfo = raw?.rate_limit_info || {};
  const running = Boolean(
    raw?.processor_active ??
    raw?.processor_running ??
    raw?.active ??
    raw?.settings_active
  );

  const slotsUsed =
    raw?.slots_used ??
    raw?.executing_count ??
    raw?.running_count ??
    0;

  const maxConcurrent =
    raw?.max_slots ??
    raw?.max_concurrent ??
    raw?.maxConcurrent ??
    0;

  const availableSlots =
    raw?.available_slots ??
    raw?.availableSlots ??
    maxConcurrent - slotsUsed;

  const pendingCount = raw?.pending_count ?? 0;
  const readyInProgress = raw?.ready_in_progress ?? 0;

  return {
    active: running,
    slots_used: Number(slotsUsed) || 0,
    max_concurrent: Number(maxConcurrent) || 0,
    tasks_remaining: Number(pendingCount + readyInProgress) || 0,
    cooldown_seconds:
      raw?.cooldown_seconds ??
      raw?.refresh_interval ??
      0,
    rate_limited: Boolean(rateInfo.paused ?? raw?.rate_limited ?? false),
    rate_limit_retry_after:
      rateInfo.remaining_secs ??
      raw?.rate_limit_retry_after ??
      0,
    rate_limit_pause_until:
      rateInfo.pause_until ??
      raw?.rate_limit_pause_until,
    available_slots: Number(availableSlots) || 0,
  };
};

function normalizeRunningProcess(raw: any): RunningProcess {
  const processId =
    raw?.process_id ??
    raw?.ProcessID ??
    raw?.processId ??
    raw?.pid ??
    raw?.id ??
    '';

  const taskId =
    raw?.task_id ??
    raw?.TaskID ??
    raw?.taskId ??
    raw?.id ??
    '';

  const startTime =
    raw?.start_time ??
    raw?.StartTime ??
    raw?.startTime ??
    '';

  const elapsed =
    raw?.elapsed_seconds ??
    raw?.duration_seconds ??
    raw?.DurationSeconds ??
    0;

  return {
    task_id: taskId,
    task_title: raw?.task_title ?? raw?.taskTitle ?? '',
    process_id: String(processId),
    agent_id: raw?.agent_id ?? raw?.AgentID ?? raw?.agentTag ?? '',
    start_time: startTime,
    elapsed_seconds: Number(elapsed) || 0,
  };
};

function mapUiSettingsToApi(settings: Settings): ApiSettingsPayload {
  return {
    // Flat structure matching backend Settings struct
    theme: settings.display.theme,
    condensed_mode: settings.display.condensed_mode,
    slots: settings.processor.concurrent_slots,
    cooldown_seconds: settings.processor.cooldown_seconds,
    refresh_interval: settings.processor.cooldown_seconds, // legacy compatibility
    active: settings.processor.active,
    max_turns: settings.agent.max_turns,
    allowed_tools: settings.agent.allowed_tools,
    skip_permissions: settings.agent.skip_permissions,
    task_timeout: settings.agent.task_timeout_minutes,
    idle_timeout_cap: settings.agent.idle_timeout_cap_minutes,
    recycler: {
      enabled_for: settings.recycler.enabled_for,
      interval_seconds: settings.recycler.recycle_interval,
      max_retries: settings.recycler.max_retries,
      retry_delay_seconds: settings.recycler.retry_delay_seconds,
      model_provider: settings.recycler.model_provider,
      model_name: settings.recycler.model_name,
      completion_threshold: settings.recycler.completion_threshold,
      failure_threshold: settings.recycler.failure_threshold,
    },
  };
}

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
    if (filters.sort) params.append('sort', filters.sort);

    const queryString = params.toString();
    const url = `/api/tasks${queryString ? '?' + queryString : ''}`;

    const response = await this.fetchJSON<{ tasks: Task[]; count: number }>(url);
    const tasks = response.tasks || [];
    return tasks.map(task => {
      const normalized = normalizeTaskTargets(task);
      // Trust the directory we fetched (query status) over stale file metadata
      if (filters.status) {
        normalized.status = filters.status as TaskStatus;
      }
      return normalized;
    });
  }

  async getTask(taskId: string): Promise<Task> {
    const task = await this.fetchJSON<Task>(`/api/tasks/${taskId}`);
    return normalizeTaskTargets(task);
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

    const task = await this.fetchJSON<Task>(`/api/tasks`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return normalizeTaskTargets(task);
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

    const response = await this.fetchJSON<Task | { task?: Task }>(`/api/tasks/${taskId}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });

    const task = (response as any)?.task ?? response;
    return normalizeTaskTargets(task as Task);
  }

  async updateTaskStatus(
    taskId: string,
    status: TaskStatus,
    additionalData: Record<string, unknown> = {}
  ): Promise<Task> {
    const task = await this.fetchJSON<Task>(`/api/tasks/${taskId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, ...additionalData }),
    });
    return normalizeTaskTargets(task);
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

    const response = await this.fetchJSON<
      Array<ActiveTarget | { target?: string; task_id?: string; status?: string; title?: string }>
    >(`/api/tasks/active-targets?${params.toString()}`);

    return (Array.isArray(response) ? response : [])
      .map(entry => {
        if (!entry) return null;
        const target = (entry as any).target ?? '';
        const taskId = (entry as any).task_id ?? '';
        const status = (entry as any).status ?? '';
        return target && taskId && status
          ? {
              target,
              task_id: taskId,
              status: status as TaskStatus,
              title: (entry as any).title ?? undefined,
            }
          : null;
      })
      .filter(Boolean) as ActiveTarget[];
  }

  // ==================== Queue Management ====================

  async getQueueStatus(): Promise<QueueStatus> {
    const status = await this.fetchJSON<Record<string, unknown>>(`/api/queue/status`);
    return normalizeQueueStatus(status);
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
    const processes = response.processes || [];
    return processes.map(normalizeRunningProcess);
  }

  async terminateProcess(taskId: string): Promise<void> {
    return this.fetchJSON<void>(`/api/queue/processes/terminate`, {
      method: 'POST',
      body: JSON.stringify({ task_id: taskId }),
    });
  }

  // ==================== Settings Management ====================

  async getSettings(): Promise<Settings> {
    const response = await this.fetchJSON<ApiSettingsPayload>(`/api/settings`);
    return mapApiSettingsToUi(response);
  }

  async updateSettings(settings: Settings): Promise<Settings> {
    const payload = mapUiSettingsToApi(settings);
    const response = await this.fetchJSON<ApiSettingsPayload>(`/api/settings`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
    return mapApiSettingsToUi(response);
  }

  async resetSettings(): Promise<Settings> {
    const response = await this.fetchJSON<ApiSettingsPayload>(`/api/settings/reset`, {
      method: 'POST',
    });
    return mapApiSettingsToUi(response);
  }

  async getRecyclerModels(provider: string): Promise<string[]> {
    const params = new URLSearchParams({ provider });
    return this.fetchJSON<string[]>(`/api/settings/recycler/models?${params.toString()}`);
  }

  // ==================== Discovery ====================

  async getResources(): Promise<Resource[]> {
    const raw = await this.fetchJSON<any>(`/api/resources`);
    const list: any[] = Array.isArray(raw) ? raw : Array.isArray((raw as any)?.resources) ? (raw as any).resources : [];
    return (list || []).map((item) => {
      const name = (item.name ?? item.Name ?? '').toString();
      const display = (item.display_name ?? item.DisplayName ?? '').toString();
      return {
        name,
        display_name: display || name,
        description: item.description ?? item.Description,
        path: item.path ?? item.Path,
        port: item.port ?? item.Port,
        category: item.category ?? item.Category,
        version: item.version ?? item.Version,
        healthy: item.healthy ?? item.Healthy,
        status: item.status ?? item.Status,
      } as Resource;
    });
  }

  async getScenarios(): Promise<Scenario[]> {
    const raw = await this.fetchJSON<any>(`/api/scenarios`);
    const list: any[] = Array.isArray(raw) ? raw : Array.isArray((raw as any)?.scenarios) ? (raw as any).scenarios : [];
    return (list || []).map((item) => {
      const name = (item.name ?? item.Name ?? '').toString();
      const display = (item.display_name ?? item.DisplayName ?? '').toString();
      return {
        name,
        display_name: display || name,
        status: item.status ?? item.Status,
        description: item.description ?? item.Description,
        path: item.path ?? item.Path,
        category: item.category ?? item.Category,
        version: item.version ?? item.Version,
      } as Scenario;
    });
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
    const response = await this.fetchJSON<ExecutionHistory[] | { executions?: ExecutionHistory[] }>(
      `/api/executions`,
    );

    if (Array.isArray(response)) {
      return response.map(normalizeExecution);
    }

    if (response && Array.isArray((response as any).executions)) {
      return (response as any).executions.map(normalizeExecution) as ExecutionHistory[];
    }

    return [];
  }

  async getExecutionHistory(taskId: string): Promise<ExecutionHistory[]> {
    const response = await this.fetchJSON<ExecutionHistory[] | { executions?: ExecutionHistory[] }>(
      `/api/tasks/${taskId}/executions`,
    );

    if (Array.isArray(response)) {
      return response.map(normalizeExecution);
    }

    if (response && Array.isArray((response as any).executions)) {
      return (response as any).executions.map(normalizeExecution) as ExecutionHistory[];
    }

    return [];
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

  async listPhaseNames(): Promise<PhaseInfo[]> {
    return this.fetchJSON<PhaseInfo[]>(`/api/prompts/phases/names`);
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

  async getSystemInsights(sinceDays: number = 7): Promise<any> {
    const params = new URLSearchParams();
    params.append('since_days', sinceDays.toString());

    return this.fetchJSON<any>(
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

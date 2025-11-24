/**
 * React Query Key Factory
 * Centralized query key definitions for consistent caching
 */

import type { TaskFilters } from '../types/api';

export const queryKeys = {
  // Tasks
  tasks: {
    all: ['tasks'] as const,
    lists: () => [...queryKeys.tasks.all, 'list'] as const,
    list: (filters: TaskFilters) => [...queryKeys.tasks.lists(), filters] as const,
    details: () => [...queryKeys.tasks.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.tasks.details(), id] as const,
    logs: (id: string) => [...queryKeys.tasks.detail(id), 'logs'] as const,
    prompt: (id: string) => [...queryKeys.tasks.detail(id), 'prompt'] as const,
    assembledPrompt: (id: string) => [...queryKeys.tasks.detail(id), 'assembled-prompt'] as const,
    executions: (id: string) => [...queryKeys.tasks.detail(id), 'executions'] as const,
    activeTargets: (type?: string, operation?: string) =>
      [...queryKeys.tasks.all, 'active-targets', { type, operation }] as const,
  },

  // Queue
  queue: {
    all: ['queue'] as const,
    status: () => [...queryKeys.queue.all, 'status'] as const,
  },

  // Processes
  processes: {
    all: ['processes'] as const,
    running: () => [...queryKeys.processes.all, 'running'] as const,
  },

  // Settings
  settings: {
    all: ['settings'] as const,
    get: () => [...queryKeys.settings.all, 'current'] as const,
    recyclerModels: (provider: string) => [...queryKeys.settings.all, 'recycler-models', provider] as const,
  },

  // Discovery
  resources: {
    all: ['resources'] as const,
    list: () => [...queryKeys.resources.all, 'list'] as const,
    detail: (name: string) => [...queryKeys.resources.all, 'detail', name] as const,
    status: (name: string) => [...queryKeys.resources.detail(name), 'status'] as const,
  },

  scenarios: {
    all: ['scenarios'] as const,
    list: () => [...queryKeys.scenarios.all, 'list'] as const,
    detail: (name: string) => [...queryKeys.scenarios.all, 'detail', name] as const,
    status: (name: string) => [...queryKeys.scenarios.detail(name), 'status'] as const,
  },

  operations: {
    all: ['operations'] as const,
    list: () => [...queryKeys.operations.all, 'list'] as const,
  },

  categories: {
    all: ['categories'] as const,
    list: () => [...queryKeys.categories.all, 'list'] as const,
  },

  prompts: {
    all: ['prompts'] as const,
    list: () => [...queryKeys.prompts.all, 'list'] as const,
    file: (id: string) => [...queryKeys.prompts.all, 'file', id] as const,
  },

  // Logs
  logs: {
    all: ['logs'] as const,
    system: (limit: number) => [...queryKeys.logs.all, 'system', limit] as const,
  },

  // Executions
  executions: {
    all: ['executions'] as const,
    list: () => [...queryKeys.executions.all, 'list'] as const,
    task: (taskId: string) => [...queryKeys.executions.all, 'task', taskId] as const,
    detail: (taskId: string, executionId: string) =>
      [...queryKeys.executions.task(taskId), 'detail', executionId] as const,
    prompt: (taskId: string, executionId: string) =>
      [...queryKeys.executions.detail(taskId, executionId), 'prompt'] as const,
    output: (taskId: string, executionId: string) =>
      [...queryKeys.executions.detail(taskId, executionId), 'output'] as const,
    metadata: (taskId: string, executionId: string) =>
      [...queryKeys.executions.detail(taskId, executionId), 'metadata'] as const,
  },

  // Auto Steer
  autoSteer: {
    all: ['autoSteer'] as const,
    profiles: () => [...queryKeys.autoSteer.all, 'profiles'] as const,
    profile: (id: string) => [...queryKeys.autoSteer.profiles(), 'detail', id] as const,
    templates: () => [...queryKeys.autoSteer.all, 'templates'] as const,
    history: (profileId?: string, scenarioName?: string) =>
      [...queryKeys.autoSteer.all, 'history', profileId ?? 'all', scenarioName ?? 'all'] as const,
    historyDetail: (executionId: string) =>
      [...queryKeys.autoSteer.all, 'history', 'detail', executionId] as const,
  },

  // Health
  health: ['health'] as const,
};

// Responsibility: manage profile and task records through the existing typed API.
import { useCallback, useEffect } from "react";

import { create } from "@bufbuild/protobuf";

import type { AgentProfile, ProfileFormData, Task, TaskFormData } from "../types";

import { AgentProfileSchema } from "@vrooli/proto-types/agent-manager/v1/domain/profile_pb";

import { TaskSchema } from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";

import { CreateProfileRequestSchema, CreateProfileResponseSchema, CreateTaskRequestSchema, CreateTaskResponseSchema, EnsureProfileResponseSchema, UpdateTaskRequestSchema, UpdateTaskResponseSchema, GetTaskResponseSchema, ListProfilesResponseSchema, ListTasksResponseSchema, UpdateProfileRequestSchema, UpdateProfileResponseSchema } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

import { ExtraFlagListSchema, FeatureFlagsSchema, SandboxConfigSchema } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

import { useApiState, apiRequest, parseProto, toProtoJson, durationFromMinutes, sandboxModeFromForm, networkAccessToProto } from "./useApiTransport";

function generateProfileKey(name: string): string {
  const base = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  if (base) {
    return base;
  }
  const rand = Math.random().toString(36).slice(2, 8);
  return `profile-${rand}`;
}

function resolveProfileKey(profile: ProfileFormData): string {
  const provided = profile.profileKey?.trim();
  return provided && provided.length > 0 ? provided : generateProfileKey(profile.name);
}

function buildProfile(profile: ProfileFormData): AgentProfile {
  return create(AgentProfileSchema, {
    name: profile.name,
    profileKey: resolveProfileKey(profile),
    description: profile.description ?? "",
    roleRef: profile.roleRef.trim(),
    maxTurns: profile.maxTurns ?? 0,
    timeout: durationFromMinutes(profile.timeoutMinutes),
    effort: profile.effort ?? "",
    allowedTools: profile.allowedTools ?? [],
    deniedTools: profile.deniedTools ?? [],
    skipPermissionPrompt: profile.skipPermissionPrompt ?? false,
    sandboxConfig: profile.sandboxMode
      ? create(SandboxConfigSchema, { mode: sandboxModeFromForm(profile.sandboxMode) })
      : undefined,
    networkAccess: networkAccessToProto(profile.networkAccess ?? "localhost"),
    allowedPaths: profile.allowedPaths ?? [],
    deniedPaths: profile.deniedPaths ?? [],
    features: profile.features?.enableBrowser
      ? create(FeatureFlagsSchema, { enableBrowser: true })
      : undefined,
    extraFlags: profile.extraFlags
      ? Object.fromEntries(
          Object.entries(profile.extraFlags).map(([rt, flags]) => [
            rt,
            create(ExtraFlagListSchema, { flags }),
          ])
        )
      : undefined,
  });
}

function buildTask(task: TaskFormData): Task {
  return create(TaskSchema, {
    title: task.title,
    description: task.description ?? "",
    scopePath: task.scopePath,
    projectRoot: task.projectRoot ?? "",
    contextAttachments: (task.contextAttachments ?? []).map((att) => ({
      ...att,
      // Map snake_case to camelCase for proto-es
      attachmentId: att.attachment_id ?? "",
    })),
  });
}

// Profiles hook
export function useProfiles(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<AgentProfile[]>([]);

  const fetchProfiles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<unknown>("/profiles");
      const message = parseProto(ListProfilesResponseSchema, data);
      setData(message.profiles ?? []);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const createProfile = useCallback(
    async (profile: ProfileFormData): Promise<AgentProfile> => {
      const request = create(CreateProfileRequestSchema, { profile: buildProfile(profile) });
      const created = await apiRequest<unknown>("/profiles", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateProfileRequestSchema, request)),
      });
      const message = parseProto(CreateProfileResponseSchema, created);
      const mapped = message.profile as AgentProfile;
      await fetchProfiles();
      return mapped;
    },
    [fetchProfiles]
  );

  const updateProfile = useCallback(
    async (id: string, profile: ProfileFormData): Promise<AgentProfile> => {
      const payload = create(UpdateProfileRequestSchema, {
        profileId: id,
        profile: { ...buildProfile(profile), id },
      });
      const updated = await apiRequest<unknown>("/profiles/" + id, {
        method: "PUT",
        body: JSON.stringify(toProtoJson(UpdateProfileRequestSchema, payload)),
      });
      const message = parseProto(UpdateProfileResponseSchema, updated);
      const mapped = message.profile as AgentProfile;
      await fetchProfiles();
      return mapped;
    },
    [fetchProfiles]
  );

  const deleteProfile = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/profiles/" + id, { method: "DELETE" });
      await fetchProfiles();
    },
    [fetchProfiles]
  );

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void fetchProfiles();
  }, [enabled, fetchProfiles]);

  return {
    data, loading, error,
    refetch: fetchProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
  };
}

// Tasks hook
export function useTasks(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<Task[]>([]);

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<unknown>("/tasks");
      const message = parseProto(ListTasksResponseSchema, data);
      setData(message.tasks ?? []);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const createTask = useCallback(
    async (task: TaskFormData): Promise<Task> => {
      const request = create(CreateTaskRequestSchema, { task: buildTask(task) });
      const created = await apiRequest<unknown>("/tasks", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateTaskRequestSchema, request)),
      });
      const message = parseProto(CreateTaskResponseSchema, created);
      const mapped = message.task as Task;
      await fetchTasks();
      return mapped;
    },
    [fetchTasks]
  );

  const updateTask = useCallback(
    async (id: string, task: TaskFormData): Promise<Task> => {
      const payload = create(UpdateTaskRequestSchema, {
        taskId: id,
        task: { ...buildTask(task), id },
      });
      const updated = await apiRequest<unknown>("/tasks/" + id, {
        method: "PUT",
        body: JSON.stringify(toProtoJson(UpdateTaskRequestSchema, payload)),
      });
      const message = parseProto(UpdateTaskResponseSchema, updated);
      const mapped = message.task as Task;
      await fetchTasks();
      return mapped;
    },
    [fetchTasks]
  );

  const getTask = useCallback(async (id: string): Promise<Task> => {
    const task = await apiRequest<unknown>("/tasks/" + id);
    const message = parseProto(GetTaskResponseSchema, task);
    return message.task as Task;
  }, []);

  const cancelTask = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/tasks/" + id + "/cancel", { method: "POST" });
      await fetchTasks();
    },
    [fetchTasks]
  );

  const deleteTask = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/tasks/" + id, { method: "DELETE" });
      await fetchTasks();
    },
    [fetchTasks]
  );

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void fetchTasks();
  }, [enabled, fetchTasks]);

  return {
    data, loading, error,
    refetch: fetchTasks,
    createTask,
    updateTask,
    getTask,
    cancelTask,
    deleteTask,
  };
}

// Ensure profile exists (standalone function)
// Creates the profile with defaults if it doesn't exist, returns existing profile otherwise
export async function ensureProfile(profileKey: string): Promise<AgentProfile> {
  const data = await apiRequest<unknown>("/profiles/ensure", {
    method: "POST",
    body: JSON.stringify({ profileKey }),
  });
  const message = parseProto(EnsureProfileResponseSchema, data);
  return message.profile as AgentProfile;
}

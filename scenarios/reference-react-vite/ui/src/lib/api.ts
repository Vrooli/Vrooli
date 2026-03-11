import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

// ============================================================================
// Types
// ============================================================================

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  color?: string;
  status: "active" | "paused" | "complete" | "archived";
  created_at: string;
  updated_at: string;
}

export interface Task {
  id: string;
  title: string;
  description?: string;
  status: "pending" | "in_progress" | "completed" | "archived";
  priority: number;
  due_date?: string;
  project_id?: string;
  created_at: string;
  updated_at: string;
}

export interface Note {
  id: string;
  content: string;
  author?: string;
  task_id: string;
  created_at: string;
  updated_at: string;
}

export interface PaginationMeta {
  total: number;
  limit: number;
  offset: number;
}

export interface ListResponse<T> {
  data: T[];
  pagination: PaginationMeta;
}

export interface ErrorResponse {
  code: string;
  message: string;
  recovery?: string;
  retryable?: boolean;
  request_id?: string;
  details?: Record<string, string>;
}

// ============================================================================
// API Error Handling
// ============================================================================

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly status: number,
    public readonly recovery?: string,
    public readonly retryable?: boolean
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let errorData: ErrorResponse | undefined;
    try {
      errorData = (await res.json()) as ErrorResponse;
    } catch {
      // Could not parse error response
    }
    throw new ApiError(
      errorData?.message ?? `Request failed: ${res.status}`,
      errorData?.code ?? "UNKNOWN_ERROR",
      res.status,
      errorData?.recovery,
      errorData?.retryable
    );
  }
  // Handle 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

// ============================================================================
// Health
// ============================================================================

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<HealthResponse>(res);
}

// ============================================================================
// Projects
// ============================================================================

export interface CreateProjectInput {
  name: string;
  description?: string;
  color?: string;
}

export interface UpdateProjectInput {
  name?: string;
  description?: string;
  color?: string;
  status?: Project["status"];
}

export async function fetchProjects(params?: {
  limit?: number;
  offset?: number;
  status?: string;
}): Promise<ListResponse<Project>> {
  const searchParams = new URLSearchParams();
  if (params?.limit !== undefined) searchParams.set("limit", String(params.limit));
  if (params?.offset !== undefined) searchParams.set("offset", String(params.offset));
  if (params?.status) searchParams.set("status", params.status);

  const query = searchParams.toString();
  const url = buildApiUrl(`/projects${query ? `?${query}` : ""}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<ListResponse<Project>>(res);
}

export async function fetchProject(id: string): Promise<Project> {
  const url = buildApiUrl(`/projects/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<Project>(res);
}

export async function createProject(input: CreateProjectInput): Promise<Project> {
  const url = buildApiUrl("/projects", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return handleResponse<Project>(res);
}

export async function updateProject(id: string, input: UpdateProjectInput): Promise<Project> {
  const url = buildApiUrl(`/projects/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return handleResponse<Project>(res);
}

export async function deleteProject(id: string): Promise<void> {
  const url = buildApiUrl(`/projects/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<void>(res);
}

// ============================================================================
// Tasks
// ============================================================================

export interface CreateTaskInput {
  title: string;
  description?: string;
  priority?: number;
  due_date?: string;
  project_id?: string;
}

export interface UpdateTaskInput {
  title?: string;
  description?: string;
  status?: Task["status"];
  priority?: number;
  due_date?: string;
  project_id?: string;
}

export async function fetchTasks(params?: {
  limit?: number;
  offset?: number;
  status?: string;
  project_id?: string;
  priority?: number;
}): Promise<ListResponse<Task>> {
  const searchParams = new URLSearchParams();
  if (params?.limit !== undefined) searchParams.set("limit", String(params.limit));
  if (params?.offset !== undefined) searchParams.set("offset", String(params.offset));
  if (params?.status) searchParams.set("status", params.status);
  if (params?.project_id) searchParams.set("project_id", params.project_id);
  if (params?.priority !== undefined) searchParams.set("priority", String(params.priority));

  const query = searchParams.toString();
  const url = buildApiUrl(`/tasks${query ? `?${query}` : ""}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<ListResponse<Task>>(res);
}

export async function fetchTask(id: string): Promise<Task> {
  const url = buildApiUrl(`/tasks/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<Task>(res);
}

export async function createTask(input: CreateTaskInput): Promise<Task> {
  const url = buildApiUrl("/tasks", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return handleResponse<Task>(res);
}

export async function updateTask(id: string, input: UpdateTaskInput): Promise<Task> {
  const url = buildApiUrl(`/tasks/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return handleResponse<Task>(res);
}

export async function deleteTask(id: string): Promise<void> {
  const url = buildApiUrl(`/tasks/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<void>(res);
}

// ============================================================================
// Notes
// ============================================================================

export interface CreateNoteInput {
  content: string;
  author?: string;
}

export interface UpdateNoteInput {
  content?: string;
}

export async function fetchNotes(taskId: string, params?: {
  limit?: number;
  offset?: number;
}): Promise<ListResponse<Note>> {
  const searchParams = new URLSearchParams();
  if (params?.limit !== undefined) searchParams.set("limit", String(params.limit));
  if (params?.offset !== undefined) searchParams.set("offset", String(params.offset));

  const query = searchParams.toString();
  const url = buildApiUrl(`/tasks/${taskId}/notes${query ? `?${query}` : ""}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<ListResponse<Note>>(res);
}

export async function fetchNote(id: string): Promise<Note> {
  const url = buildApiUrl(`/notes/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<Note>(res);
}

export async function createNote(taskId: string, input: CreateNoteInput): Promise<Note> {
  const url = buildApiUrl(`/tasks/${taskId}/notes`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return handleResponse<Note>(res);
}

export async function updateNote(id: string, input: UpdateNoteInput): Promise<Note> {
  const url = buildApiUrl(`/notes/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return handleResponse<Note>(res);
}

export async function deleteNote(id: string): Promise<void> {
  const url = buildApiUrl(`/notes/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<void>(res);
}

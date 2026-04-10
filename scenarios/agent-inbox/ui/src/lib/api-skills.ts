/**
 * Skills CRUD, suggestions, sync API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { Skill } from "@/lib/types/templates";

// =============================================================================
// Skill Types
// =============================================================================

export interface SkillResponse extends Skill {
  createdAt?: string;
  updatedAt?: string;
}

export interface SkillListResponse {
  skills: SkillResponse[];
  count: number;
}

// =============================================================================
// Skill CRUD
// =============================================================================

/**
 * Fetch all skills (defaults merged with user overrides).
 */
export async function fetchSkills(): Promise<SkillListResponse> {
  const url = buildApiUrl("/skills", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch skills: ${res.status}`);
  }

  return jsonResponse<SkillListResponse>(res);
}

/**
 * Fetch a single skill by ID.
 * @param id - Skill ID
 */
export async function fetchSkill(id: string): Promise<SkillResponse> {
  const url = buildApiUrl(`/skills/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch skill: ${res.status}`);
  }

  return jsonResponse<SkillResponse>(res);
}

export type CreateSkillInput = Omit<Skill, "id">;

/**
 * Create a new user skill.
 * @param skill - Skill data (id will be generated)
 */
export async function createSkill(skill: CreateSkillInput): Promise<SkillResponse> {
  const url = buildApiUrl("/skills", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(skill)
  });

  if (!res.ok) {
    throw new Error(`Failed to create skill: ${res.status}`);
  }

  return jsonResponse<SkillResponse>(res);
}

export type UpdateSkillInput = Partial<Omit<Skill, "id">>;

/**
 * Update an existing skill via prompt-manager.
 * @param id - Skill ID
 * @param updates - Fields to update
 */
export async function updateSkill(id: string, updates: UpdateSkillInput): Promise<SkillResponse> {
  const url = buildApiUrl(`/skills/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates)
  });

  if (!res.ok) {
    throw new Error(`Failed to update skill: ${res.status}`);
  }

  return jsonResponse<SkillResponse>(res);
}

/**
 * Delete a user skill or user override.
 * @param id - Skill ID
 */
export async function deleteSkill(id: string): Promise<void> {
  const url = buildApiUrl(`/skills/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete skill: ${res.status}`);
  }
}

/**
 * Import multiple skills from a JSON array.
 * @param skills - Array of skills to import
 */
export async function importSkills(skills: Skill[]): Promise<{ imported: number }> {
  const url = buildApiUrl("/skills/import", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(skills)
  });

  if (!res.ok) {
    throw new Error(`Failed to import skills: ${res.status}`);
  }

  return jsonResponse<{ imported: number }>(res);
}

/**
 * Export all user skills.
 */
export async function exportSkills(): Promise<Skill[]> {
  const url = buildApiUrl("/skills/export", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to export skills: ${res.status}`);
  }

  return jsonResponse<Skill[]>(res);
}

// =============================================================================
// Skill Suggestions
// =============================================================================

export interface SuggestedSkill {
  id: string;
  name: string;
  description: string;
  tags?: string[];
  modes?: string[];
  score: number;
  scorePercent: number;
}

export interface SkillSuggestResponse {
  suggestions: SuggestedSkill[];
  queryCount: number;
  method: string;
}

/**
 * Fetch AI-powered skill suggestions based on conversation context.
 * Returns empty suggestions on any error (graceful degradation).
 */
export async function fetchSkillSuggestions(params: {
  inputText?: string;
  chatId?: string;
  excludeSkillIds?: string[];
  signal?: AbortSignal;
}): Promise<SkillSuggestResponse> {
  const url = buildApiUrl("/skills/suggest", { baseUrl: API_BASE });

  try {
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        inputText: params.inputText,
        chatId: params.chatId,
        excludeSkillIds: params.excludeSkillIds,
      }),
      signal: params.signal ?? (params.inputText ? AbortSignal.timeout(20000) : undefined),
    });

    if (!res.ok) {
      return { suggestions: [], queryCount: 0, method: "error" };
    }

    return jsonResponse<SkillSuggestResponse>(res);
  } catch {
    return { suggestions: [], queryCount: 0, method: "error" };
  }
}

// =============================================================================
// Skill Sync
// =============================================================================

/**
 * Sync status response from the server.
 */
export interface SyncStatus {
  success: boolean;
  skillCount: number;
  localCount: number;
  hash: string;
  error?: string;
}

/**
 * Trigger an immediate sync of skills from prompt-manager.
 * This fetches the latest skills and updates the local cache.
 */
export async function syncSkills(): Promise<SyncStatus> {
  const url = buildApiUrl("/skills/sync", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to sync skills: ${res.status}`);
  }

  return jsonResponse<SyncStatus>(res);
}

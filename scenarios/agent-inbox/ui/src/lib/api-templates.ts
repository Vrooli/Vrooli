/**
 * Templates CRUD, import/export API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";

// =============================================================================
// Template Types
// =============================================================================

export interface TemplateVariable {
  name: string;
  label: string;
  type: "text" | "textarea" | "select";
  placeholder?: string;
  options?: string[];
  required?: boolean;
  defaultValue?: string;
}

export interface Template {
  id: string;
  name: string;
  description: string;
  icon?: string;
  modes?: string[];
  content: string;
  variables: TemplateVariable[];
  suggestedSkillIds?: string[];
  suggestedToolIds?: string[];
  draft?: boolean;
}

export type TemplateSource = "default" | "user" | "modified";

export interface TemplateResponse extends Template {
  source: TemplateSource;
  hasDefault: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface TemplateListResponse {
  templates: TemplateResponse[];
  defaults_count: number;
  user_count: number;
  modified_defaults_count: number;
}

// =============================================================================
// Template API Functions
// =============================================================================

/**
 * Fetch all templates (defaults merged with user overrides).
 */
export async function fetchTemplates(): Promise<TemplateListResponse> {
  const url = buildApiUrl("/templates", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch templates: ${res.status}`);
  }

  return jsonResponse<TemplateListResponse>(res);
}

/**
 * Fetch a single template by ID.
 * @param id - Template ID
 */
export async function fetchTemplate(id: string): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch template: ${res.status}`);
  }

  return jsonResponse<TemplateResponse>(res);
}

export type CreateTemplateInput = Omit<Template, "id">;

/**
 * Create a new user template.
 * @param template - Template data (id will be generated)
 */
export async function createTemplate(template: CreateTemplateInput): Promise<TemplateResponse> {
  const url = buildApiUrl("/templates", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(template)
  });

  if (!res.ok) {
    throw new Error(`Failed to create template: ${res.status}`);
  }

  return jsonResponse<TemplateResponse>(res);
}

export type UpdateTemplateInput = Partial<Omit<Template, "id">>;

/**
 * Update an existing template.
 * If it's a default template, creates a user override.
 * @param id - Template ID
 * @param updates - Fields to update
 */
export async function updateTemplate(id: string, updates: UpdateTemplateInput): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates)
  });

  if (!res.ok) {
    throw new Error(`Failed to update template: ${res.status}`);
  }

  return jsonResponse<TemplateResponse>(res);
}

/**
 * Delete a user template or user override.
 * @param id - Template ID
 */
export async function deleteTemplate(id: string): Promise<void> {
  const url = buildApiUrl(`/templates/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete template: ${res.status}`);
  }
}

/**
 * Reset a modified default template to its original state.
 * Deletes the user override to reveal the default.
 * @param id - Template ID
 */
export async function resetTemplate(id: string): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}/reset`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to reset template: ${res.status}`);
  }

  return jsonResponse<TemplateResponse>(res);
}

/**
 * Update the actual default template (not a user override).
 * This modifies the shipped default template files directly.
 * @param id - Template ID
 * @param updates - Template fields to update
 */
export async function updateDefaultTemplate(id: string, updates: UpdateTemplateInput): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}/update-default`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates)
  });

  if (!res.ok) {
    throw new Error(`Failed to update default template: ${res.status}`);
  }

  return jsonResponse<TemplateResponse>(res);
}

/**
 * Import multiple templates from a JSON array.
 * @param templates - Array of templates to import
 */
export async function importTemplates(templates: Template[]): Promise<{ imported: number }> {
  const url = buildApiUrl("/templates/import", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(templates)
  });

  if (!res.ok) {
    throw new Error(`Failed to import templates: ${res.status}`);
  }

  return jsonResponse<{ imported: number }>(res);
}

/**
 * Export all user templates.
 */
export async function exportTemplates(): Promise<Template[]> {
  const url = buildApiUrl("/templates/export", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to export templates: ${res.status}`);
  }

  return jsonResponse<Template[]>(res);
}

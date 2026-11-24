import type { Template } from "./spec";

const LEGACY_CACHE_KEY = "rcl.workbench.templates.v1";

function isTemplate(value: unknown): value is Template {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return (
    typeof record.id === "string" &&
    typeof record.name === "string" &&
    typeof record.spec === "object" &&
    record.spec !== null &&
    typeof record.palette === "object" &&
    record.palette !== null
  );
}

/**
 * Read-only bridge for old component-context bookmarks. Durable server
 * templates are resolved through their imported workspace IDs first; this
 * reads only browser-local drafts that never reached the server before cutover.
 */
export function readLegacyTemplateForMigration(id: string): Template | undefined {
  try {
    const raw = window.localStorage.getItem(LEGACY_CACHE_KEY);
    if (!raw) return undefined;
    const templates: unknown = JSON.parse(raw);
    if (!Array.isArray(templates)) return undefined;
    return templates.find((entry): entry is Template => isTemplate(entry) && entry.id === id);
  } catch {
    return undefined;
  }
}

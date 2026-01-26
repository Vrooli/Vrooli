// DOC: docs/reference/api-endpoints.md#documentation-viewer
import type { DocContentResponse } from "../services/documentationApi";

export type DocResetDefaults = {
  maxAgeDays: number;
  keepMinEntries: number;
};

export type DocMetaViewModel = {
  path: string;
  docTypeLabel: string;
  sizeLabel: string;
  modifiedLabel: string;
  canReset: boolean;
  resetDefaults: DocResetDefaults;
};

const DEFAULT_RESET: DocResetDefaults = {
  maxAgeDays: 30,
  keepMinEntries: 3,
};

const formatBytes = (value: number) => {
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let size = value;
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index += 1;
  }
  return `${size.toFixed(size >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
};

const formatDateTime = (value?: string) => {
  if (!value) return "Unknown";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Unknown";
  return date.toLocaleString();
};

export function buildDocMetaViewModel(content?: DocContentResponse | null): DocMetaViewModel | null {
  if (!content) return null;
  const resetConfig = content.reset_config ?? {};
  return {
    path: content.path,
    docTypeLabel: content.doc_type ? content.doc_type : "unclassified",
    sizeLabel: formatBytes(content.size),
    modifiedLabel: formatDateTime(content.modified_at),
    canReset: content.can_reset,
    resetDefaults: {
      maxAgeDays: resetConfig.max_age_days ?? DEFAULT_RESET.maxAgeDays,
      keepMinEntries: resetConfig.keep_min_entries ?? DEFAULT_RESET.keepMinEntries,
    },
  };
}

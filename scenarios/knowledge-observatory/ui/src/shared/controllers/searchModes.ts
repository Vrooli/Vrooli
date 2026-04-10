// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
export const SEARCH_MODES = ["semantic", "files", "text", "unified", "deep"] as const;

export type SearchMode = (typeof SEARCH_MODES)[number];

export type SearchModeOption = {
  mode: SearchMode;
  label: string;
  description: string;
};

export const DEFAULT_SEARCH_MODE: SearchMode = "semantic";

const MODE_DESCRIPTIONS: Record<SearchMode, SearchModeOption> = {
  semantic: {
    mode: "semantic",
    label: "Semantic",
    description: "Natural-language knowledge search across vector collections.",
  },
  files: {
    mode: "files",
    label: "File",
    description: "Find documentation files by path or filename pattern.",
  },
  text: {
    mode: "text",
    label: "Text",
    description: "Full-text search across documentation content.",
  },
  unified: {
    mode: "unified",
    label: "Unified",
    description: "Blend file, text, and semantic results into one ranking.",
  },
  deep: {
    mode: "deep",
    label: "Deep",
    description: "Spawn an agent to read and follow documentation references.",
  },
};

const MODE_ALIASES: Record<string, SearchMode> = {
  semantic: "semantic",
  file: "files",
  files: "files",
  text: "text",
  unified: "unified",
  deep: "deep",
};

export const SEARCH_MODE_OPTIONS: SearchModeOption[] = SEARCH_MODES.map(
  (mode) => MODE_DESCRIPTIONS[mode]
);

export function normalizeSearchMode(value: unknown): SearchMode {
  if (typeof value !== "string") {
    return DEFAULT_SEARCH_MODE;
  }
  const normalized = value.trim().toLowerCase();
  return MODE_ALIASES[normalized] ?? DEFAULT_SEARCH_MODE;
}

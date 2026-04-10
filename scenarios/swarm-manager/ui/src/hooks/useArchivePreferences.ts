import { useState, useEffect, useMemo } from "react";
import type { ScenarioFile, PreserveFilesPreset } from "../types";

export const ARCHIVE_PRESET_OPTIONS: { value: PreserveFilesPreset; label: string; description: string }[] = [
  { value: "documentation", label: "Documentation", description: "PRD.md, README.md, docs/**, *.md" },
  { value: "requirements", label: "Requirements", description: "PRD.md, requirements/**, specs/**, REQUIREMENTS.md" },
  { value: "planning", label: "Planning", description: "PRD.md, .vrooli/**, planning/**, design/**" },
  {
    value: "all-planning",
    label: "All Planning Files",
    description: "All docs, requirements, specs, planning, design, and markdown files",
  },
];

const ARCHIVE_PREFERENCES_STORAGE_KEY = "swarm-manager.archive.preferences.v1";

export interface ArchivePreferences {
  mode: "preset" | "custom";
  preset: PreserveFilesPreset;
  customPaths: string[];
  specSync: boolean;
}

export const ARCHIVE_PRESET_PATTERNS: Record<PreserveFilesPreset, string[]> = {
  documentation: ["PRD.md", "README.md", "docs/**", "*.md"],
  requirements: ["PRD.md", "requirements/**", "specs/**", "REQUIREMENTS.md"],
  planning: ["PRD.md", ".vrooli/**", "planning/**", "design/**"],
  "all-planning": [
    "PRD.md",
    "README.md",
    "docs/**",
    "requirements/**",
    "specs/**",
    "planning/**",
    "design/**",
    ".vrooli/**",
    "*.md",
  ],
};

const ARCHIVE_IGNORED_DIRS = new Set([
  "node_modules",
  ".git",
  "dist",
  "build",
  "coverage",
  ".next",
  ".turbo",
  "target",
  "vendor",
]);

export function isIgnoredArchivePath(path: string): boolean {
  return path.split("/").some((segment) => ARCHIVE_IGNORED_DIRS.has(segment));
}

export function matchesPattern(path: string, pattern: string): boolean {
  if (!pattern.includes("*")) {
    return path === pattern || path.endsWith("/" + pattern);
  }
  if (pattern.includes("**")) {
    const prefix = pattern.split("**")[0];
    return !prefix || path.startsWith(prefix);
  }
  if (pattern.startsWith("*.")) {
    return path.endsWith(pattern.slice(1));
  }
  if (pattern.endsWith("/*")) {
    const prefix = pattern.slice(0, -2);
    return path.startsWith(prefix + "/") && !path.slice(prefix.length + 1).includes("/");
  }
  return false;
}

export function collectPaths(file: ScenarioFile): string[] {
  if (file.type === "file") return [file.path];
  if (!file.children) return [];
  return file.children.flatMap(collectPaths);
}

function getDefaultArchivePreferences(): ArchivePreferences {
  return {
    mode: "preset",
    preset: "planning",
    customPaths: [],
    specSync: false,
  };
}

function isValidPreset(value: string): value is PreserveFilesPreset {
  return value in ARCHIVE_PRESET_PATTERNS;
}

export function loadArchivePreferences(): ArchivePreferences {
  if (typeof window === "undefined") {
    return getDefaultArchivePreferences();
  }
  try {
    const raw = window.localStorage.getItem(ARCHIVE_PREFERENCES_STORAGE_KEY);
    if (!raw) return getDefaultArchivePreferences();
    const parsed = JSON.parse(raw) as Partial<ArchivePreferences>;
    const preset = parsed.preset && isValidPreset(parsed.preset) ? parsed.preset : "planning";
    const mode = parsed.mode === "custom" ? "custom" : "preset";
    const customPaths = Array.isArray(parsed.customPaths)
      ? parsed.customPaths.filter((path): path is string => typeof path === "string" && path.length > 0)
      : [];
    const specSync = parsed.specSync === true;
    return { mode, preset, customPaths, specSync };
  } catch {
    return getDefaultArchivePreferences();
  }
}

export function persistArchivePreferences(preferences: ArchivePreferences): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(ARCHIVE_PREFERENCES_STORAGE_KEY, JSON.stringify(preferences));
}

export function useArchivePreferences(scenarioFiles: ScenarioFile[], archiveOnDelete: boolean) {
  const [archivePreset, setArchivePreset] = useState<PreserveFilesPreset>(() => loadArchivePreferences().preset);
  const [archiveMode, setArchiveMode] = useState<"preset" | "custom">(() => loadArchivePreferences().mode);
  const [customPaths, setCustomPaths] = useState<string[]>(() => loadArchivePreferences().customPaths);
  const [specSyncEnabled, setSpecSyncEnabled] = useState(() => loadArchivePreferences().specSync);
  const [fileTreeLoaded, setFileTreeLoaded] = useState(false);

  useEffect(() => {
    persistArchivePreferences({
      mode: archiveMode,
      preset: archivePreset,
      customPaths,
      specSync: specSyncEnabled,
    });
  }, [archiveMode, archivePreset, customPaths, specSyncEnabled]);

  const allScenarioFilePaths = useMemo(
    () => scenarioFiles.flatMap(collectPaths).sort((a, b) => a.localeCompare(b)),
    [scenarioFiles]
  );
  const scenarioPathSet = useMemo(() => new Set(allScenarioFilePaths), [allScenarioFilePaths]);
  const canUseCustomSelection = useMemo(() => {
    if (archiveMode !== "custom" || customPaths.length === 0) return false;
    if (!fileTreeLoaded) return true;
    return customPaths.every((path) => scenarioPathSet.has(path));
  }, [archiveMode, customPaths, fileTreeLoaded, scenarioPathSet]);
  const effectiveArchiveMode: "preset" | "custom" = archiveMode === "custom" && canUseCustomSelection ? "custom" : "preset";
  const previewPaths = useMemo(() => {
    if (!archiveOnDelete) return [];
    if (effectiveArchiveMode === "custom") return [...customPaths].sort((a, b) => a.localeCompare(b));
    const patterns = ARCHIVE_PRESET_PATTERNS[archivePreset];
    return allScenarioFilePaths.filter(
      (path) => !isIgnoredArchivePath(path) && patterns.some((pattern) => matchesPattern(path, pattern))
    );
  }, [archiveOnDelete, effectiveArchiveMode, customPaths, archivePreset, allScenarioFilePaths]);
  const previewList = previewPaths.slice(0, 10);

  return {
    archivePreset,
    setArchivePreset,
    archiveMode,
    setArchiveMode,
    customPaths,
    setCustomPaths,
    specSyncEnabled,
    setSpecSyncEnabled,
    fileTreeLoaded,
    setFileTreeLoaded,
    effectiveArchiveMode,
    previewPaths,
    previewList,
  };
}

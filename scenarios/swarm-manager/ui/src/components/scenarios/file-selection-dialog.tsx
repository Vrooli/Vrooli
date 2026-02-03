/**
 * FileSelectionDialog Component
 *
 * A modal dialog for selecting files to preserve when archiving a scenario.
 * Supports preset selections (documentation, requirements, planning, all-planning)
 * and custom file selection via checkboxes.
 */

import { useState, useEffect, useMemo } from "react";
import { X, Files, CheckSquare, Square } from "lucide-react";
import { Button } from "../ui/button";
import { FileTree } from "../ui/file-tree";
import type { ScenarioFile, PreserveFilesPreset } from "../../types";

const PRESETS: { value: PreserveFilesPreset | ""; label: string; description: string }[] = [
  { value: "", label: "Custom selection", description: "Select specific files manually" },
  { value: "documentation", label: "Documentation", description: "PRD.md, README.md, docs/**, *.md" },
  { value: "requirements", label: "Requirements", description: "PRD.md, requirements/**, specs/**, REQUIREMENTS.md" },
  { value: "planning", label: "Planning", description: "PRD.md, .vrooli/**, planning/**, design/**" },
  {
    value: "all-planning",
    label: "All Planning Files",
    description: "All docs, requirements, specs, planning, and markdown files",
  },
];

/** Patterns for each preset (matching backend archivePresets) */
const PRESET_PATTERNS: Record<PreserveFilesPreset, string[]> = {
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

export interface FileSelectionResult {
  preset?: PreserveFilesPreset;
  paths: string[];
}

export interface FileSelectionDialogProps {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Callback when dialog is closed */
  onClose: () => void;
  /** Callback when user confirms selection */
  onConfirm: (selection: FileSelectionResult) => void;
  /** Name of the scenario being archived */
  scenarioName: string;
  /** File tree for the scenario */
  files: ScenarioFile[];
  /** Whether files are loading */
  isLoading?: boolean;
}

/**
 * Check if a file path matches a glob pattern
 * Simple implementation supporting *, ** and exact matches
 */
function matchesPattern(path: string, pattern: string): boolean {
  // Handle exact match
  if (!pattern.includes("*")) {
    return path === pattern || path.endsWith("/" + pattern);
  }

  // Handle ** glob (matches any path)
  if (pattern.includes("**")) {
    const prefix = pattern.split("**")[0];
    if (prefix && !path.startsWith(prefix)) return false;
    return true;
  }

  // Handle *.ext patterns
  if (pattern.startsWith("*.")) {
    const ext = pattern.slice(1);
    return path.endsWith(ext);
  }

  // Handle prefix/* patterns
  if (pattern.endsWith("/*")) {
    const prefix = pattern.slice(0, -2);
    return path.startsWith(prefix + "/") && !path.slice(prefix.length + 1).includes("/");
  }

  return false;
}

/**
 * Collect all file paths in the scenario tree (files only).
 */
function collectFilePaths(file: ScenarioFile): string[] {
  if (file.type === "file") {
    return [file.path];
  }
  if (!file.children) {
    return [];
  }
  return file.children.flatMap(collectFilePaths);
}

function getAllFilePathsFromTree(files: ScenarioFile[]): string[] {
  return files.flatMap(collectFilePaths);
}

/**
 * Get all file paths that match a preset's patterns
 */
function getPathsForPreset(files: ScenarioFile[], preset: PreserveFilesPreset): Set<string> {
  const patterns = PRESET_PATTERNS[preset];
  const allPaths = getAllFilePathsFromTree(files);
  const matchedPaths = new Set<string>();

  for (const path of allPaths) {
    for (const pattern of patterns) {
      if (matchesPattern(path, pattern)) {
        matchedPaths.add(path);
        break;
      }
    }
  }

  return matchedPaths;
}

export function FileSelectionDialog({
  isOpen,
  onClose,
  onConfirm,
  scenarioName,
  files,
  isLoading = false,
}: FileSelectionDialogProps) {
  const [selectedPreset, setSelectedPreset] = useState<PreserveFilesPreset | "">("");
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());

  // Get all file paths for select all functionality
  const allFilePaths = useMemo(() => getAllFilePathsFromTree(files), [files]);

  // Reset selection when dialog opens
  useEffect(() => {
    if (isOpen) {
      setSelectedPreset("");
      setSelectedPaths(new Set());
    }
  }, [isOpen]);

  // When preset changes, update selected paths
  useEffect(() => {
    if (selectedPreset) {
      setSelectedPaths(getPathsForPreset(files, selectedPreset));
    }
  }, [selectedPreset, files]);

  // Handle keyboard events
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  const handlePresetChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value as PreserveFilesPreset | "";
    setSelectedPreset(value);
    if (!value) {
      // Clear selection when switching to custom
      setSelectedPaths(new Set());
    }
  };

  const handleSelectionChange = (paths: Set<string>) => {
    // When user manually changes selection, clear preset
    setSelectedPreset("");
    setSelectedPaths(paths);
  };

  const handleSelectAll = () => {
    setSelectedPreset("");
    setSelectedPaths(new Set(allFilePaths));
  };

  const handleClearAll = () => {
    setSelectedPreset("");
    setSelectedPaths(new Set());
  };

  const handleConfirm = () => {
    onConfirm({
      preset: selectedPreset || undefined,
      paths: Array.from(selectedPaths),
    });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        className="relative z-10 flex max-h-[80vh] w-full max-w-2xl flex-col rounded-xl border border-white/10 bg-slate-900 shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="file-selection-title"
        data-testid="file-selection-dialog"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-white/10 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-cyan-500/20 p-2">
              <Files className="h-5 w-5 text-cyan-400" />
            </div>
            <div>
              <h2 id="file-selection-title" className="text-lg font-semibold text-slate-100">
                Preserve Files
              </h2>
              <p className="text-sm text-slate-400">
                Select files to keep when archiving <span className="text-cyan-400">{scenarioName}</span>
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
            aria-label="Close dialog"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Preset selector */}
        <div className="border-b border-white/10 px-6 py-4">
          <label className="mb-2 block text-sm font-medium text-slate-300">Quick select preset</label>
          <select
            value={selectedPreset}
            onChange={handlePresetChange}
            className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            data-testid="preset-select"
          >
            {PRESETS.map((preset) => (
              <option key={preset.value} value={preset.value}>
                {preset.label}
              </option>
            ))}
          </select>
          {selectedPreset && (
            <p className="mt-1 text-xs text-slate-500">
              {PRESETS.find((p) => p.value === selectedPreset)?.description}
            </p>
          )}
        </div>

        {/* File tree with checkboxes */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {/* Select all / Clear all buttons */}
          <div className="mb-3 flex items-center justify-between">
            <span className="text-sm text-slate-400">
              {selectedPaths.size} of {allFilePaths.length} files selected
            </span>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleSelectAll}
                className="flex items-center gap-1 rounded px-2 py-1 text-xs text-cyan-400 hover:bg-slate-800"
                data-testid="select-all-button"
              >
                <CheckSquare className="h-3 w-3" />
                Select All
              </button>
              <button
                type="button"
                onClick={handleClearAll}
                className="flex items-center gap-1 rounded px-2 py-1 text-xs text-slate-400 hover:bg-slate-800"
                data-testid="clear-all-button"
              >
                <Square className="h-3 w-3" />
                Clear All
              </button>
            </div>
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="h-8 w-8 animate-spin rounded-full border-2 border-cyan-500 border-t-transparent" />
            </div>
          ) : (
            <FileTree
              files={files}
              selectionMode="checkbox"
              selectedPaths={selectedPaths}
              onSelectionChange={handleSelectionChange}
              data-testid="file-selection-tree"
            />
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 border-t border-white/10 px-6 py-4">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="default" onClick={handleConfirm} data-testid="confirm-selection-button">
            Confirm Selection ({selectedPaths.size} files)
          </Button>
        </div>
      </div>
    </div>
  );
}

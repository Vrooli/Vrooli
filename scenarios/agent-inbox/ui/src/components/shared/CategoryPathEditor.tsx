/**
 * CategoryPathEditor - Dynamic hierarchical path editor with suggestions.
 *
 * Used for editing category/mode paths on both Skills and Templates.
 * Each level shows suggestions from existing items and allows custom values.
 * Custom values show a "NEW" badge to indicate a new category will be created.
 */

import { useCallback, useMemo } from "react";
import { Plus } from "lucide-react";
import { cn } from "@/lib/utils";
import { Combobox } from "./Combobox";

interface CategoryPathEditorProps {
  value: string[];
  onChange: (value: string[]) => void;
  getSuggestionsAtLevel: (level: number, parentPath: string[]) => string[];
  label?: string;
  placeholder?: string;
  disabled?: boolean;
  required?: boolean;
  error?: string;
}

export function CategoryPathEditor({
  value,
  onChange,
  getSuggestionsAtLevel,
  label = "Category Path",
  placeholder,
  disabled,
  required,
  error,
}: CategoryPathEditorProps) {
  // Handle value change at a specific level
  const handleLevelChange = useCallback(
    (level: number, newValue: string) => {
      const newPath = [...value];
      if (newValue.trim()) {
        newPath[level] = newValue.trim();
        // Truncate any levels after this one when changing
        onChange(newPath.slice(0, level + 1));
      } else {
        // Clear this level and all after
        onChange(newPath.slice(0, level));
      }
    },
    [value, onChange]
  );

  // Handle delete at a specific level
  const handleDelete = useCallback(
    (level: number) => {
      onChange(value.slice(0, level));
    },
    [value, onChange]
  );

  // Add a new level
  const handleAddLevel = useCallback(() => {
    onChange([...value, ""]);
  }, [value, onChange]);

  // Get suggestions for each level
  const getSuggestions = useCallback(
    (level: number): string[] => {
      const parentPath = value.slice(0, level);
      return getSuggestionsAtLevel(level, parentPath);
    },
    [value, getSuggestionsAtLevel]
  );

  // Determine how many levels to show
  const levels = useMemo(() => {
    if (value.length === 0) return [0];
    return value.map((_, i) => i);
  }, [value]);

  // Path preview text
  const pathPreview = value.filter(Boolean).join(" / ") || "(none)";

  return (
    <div className="space-y-2">
      {/* Label */}
      <label className="block text-sm font-medium text-slate-300">
        {label}
        {required && <span className="text-red-400 ml-1">*</span>}
      </label>

      {/* Combobox levels */}
      <div className="space-y-2">
        {levels.map((level) => (
          <Combobox
            key={level}
            level={level}
            value={value[level] || ""}
            onChange={(v) => handleLevelChange(level, v)}
            suggestions={getSuggestions(level)}
            placeholder={
              level === 0 ? (placeholder || "Select or type category...") : "Subcategory (optional)"
            }
            disabled={disabled}
            onDelete={() => handleDelete(level)}
            showDelete={level > 0 || (level === 0 && !required)}
          />
        ))}
      </div>

      {/* Add level button */}
      {value.length > 0 && value[value.length - 1]?.trim() && (
        <button
          type="button"
          onClick={handleAddLevel}
          disabled={disabled}
          className={cn(
            "flex items-center gap-1.5 px-2 py-1.5 text-xs text-slate-400 hover:text-white",
            "hover:bg-white/5 rounded-lg transition-colors",
            disabled && "opacity-50 cursor-not-allowed"
          )}
        >
          <Plus className="h-3 w-3" />
          Add level
        </button>
      )}

      {/* Path preview */}
      <p className="text-xs text-slate-500">
        Path: {pathPreview}
      </p>

      {/* Error message */}
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  );
}

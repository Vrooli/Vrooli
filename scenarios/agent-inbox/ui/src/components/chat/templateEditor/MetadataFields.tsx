/**
 * Left column metadata fields: Name, Description, Icon, Draft status.
 * Used in the template editor modal.
 */

import { Construction } from "lucide-react";
import { IconSelector, TEMPLATE_ICON_OPTIONS } from "@/components/shared/IconSelector";

interface MetadataFieldsProps {
  name: string;
  onNameChange: (value: string) => void;
  description: string;
  onDescriptionChange: (value: string) => void;
  icon: string;
  onIconChange: (value: string) => void;
  draft: boolean;
  onDraftChange: (value: boolean) => void;
  readOnly: boolean;
  errors: Record<string, string>;
}

export function MetadataFields({
  name,
  onNameChange,
  description,
  onDescriptionChange,
  icon,
  onIconChange,
  draft,
  onDraftChange,
  readOnly,
  errors,
}: MetadataFieldsProps) {
  return (
    <>
      {/* Name */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">
          Name {!readOnly && "*"}
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="e.g., Debug Performance Issue"
          disabled={readOnly}
          className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
            errors.name ? "border-red-500" : "border-white/10"
          } ${readOnly ? "opacity-70 cursor-not-allowed" : ""}`}
        />
        {errors.name && (
          <p className="text-xs text-red-400 mt-1">{errors.name}</p>
        )}
      </div>

      {/* Description */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">
          Description {!readOnly && "*"}
        </label>
        <input
          type="text"
          value={description}
          onChange={(e) => onDescriptionChange(e.target.value)}
          placeholder="Short description of what this template does"
          disabled={readOnly}
          className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
            errors.description ? "border-red-500" : "border-white/10"
          } ${readOnly ? "opacity-70 cursor-not-allowed" : ""}`}
        />
        {errors.description && (
          <p className="text-xs text-red-400 mt-1">{errors.description}</p>
        )}
      </div>

      {/* Icon */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">
          Icon
        </label>
        <IconSelector
          value={icon}
          onChange={onIconChange}
          icons={TEMPLATE_ICON_OPTIONS}
          disabled={readOnly}
        />
      </div>

      {/* Draft Status */}
      {!readOnly && (
        <div className="flex items-center gap-3 p-3 bg-slate-800/50 border border-white/5 rounded-lg">
          <input
            type="checkbox"
            id="draft"
            checked={draft}
            onChange={(e) => onDraftChange(e.target.checked)}
            className="rounded bg-slate-700 border-white/20 text-orange-500 focus:ring-orange-500"
          />
          <div className="flex-1">
            <label htmlFor="draft" className="text-sm text-slate-300 cursor-pointer flex items-center gap-2">
              <Construction className="h-4 w-4 text-orange-400" />
              Mark as draft
            </label>
            <p className="text-xs text-slate-500 mt-0.5">
              Draft templates show a warning that they may not be fully working
            </p>
          </div>
        </div>
      )}
    </>
  );
}

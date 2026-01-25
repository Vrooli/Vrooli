/**
 * App metadata section for GeneratorForm.
 * Handles app display name, icon path, and description inputs.
 * Collapsible by default with a summary preview when collapsed.
 */

import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { Input } from "../ui/input";
import { Label } from "../ui/label";

export interface AppMetadataSectionProps {
  scenarioName: string;
  appDisplayName: string;
  onAppDisplayNameChange: (value: string) => void;
  iconPath: string;
  onIconPathChange: (value: string) => void;
  iconPreviewUrl: string | null;
  iconPreviewError: boolean;
  onIconPreviewError: (error: boolean) => void;
  appDescription: string;
  onAppDescriptionChange: (value: string) => void;
}

/** Build a summary string from current values for collapsed state */
function buildSummary(
  displayName: string,
  iconPath: string,
  description: string
): string {
  const parts: string[] = [];

  if (displayName.trim()) {
    parts.push(displayName.trim());
  }

  if (iconPath.trim()) {
    const filename = iconPath.trim().split(/[\\/]/).pop() || iconPath.trim();
    parts.push(filename);
  }

  if (description.trim()) {
    const truncated =
      description.trim().length > 40
        ? description.trim().slice(0, 40) + "..."
        : description.trim();
    parts.push(truncated);
  }

  return parts.length > 0 ? parts.join(" • ") : "No metadata configured";
}

export function AppMetadataSection({
  scenarioName,
  appDisplayName,
  onAppDisplayNameChange,
  iconPath,
  onIconPathChange,
  iconPreviewUrl,
  iconPreviewError,
  onIconPreviewError,
  appDescription,
  onAppDescriptionChange,
}: AppMetadataSectionProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const summary = buildSummary(appDisplayName, iconPath, appDescription);

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50">
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between p-4 text-left hover:bg-slate-800/30 transition-colors rounded-lg"
      >
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-medium text-slate-200">App Metadata</h3>
          {!isExpanded && (
            <p className="text-xs text-slate-400 mt-1 truncate max-w-md">
              {summary}
            </p>
          )}
        </div>
        <ChevronDown
          className={`h-4 w-4 text-slate-400 transition-transform flex-shrink-0 ml-2 ${
            isExpanded ? "rotate-180" : ""
          }`}
        />
      </button>

      {isExpanded && (
        <div className="px-4 pb-4 space-y-3">
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <Label htmlFor="appDisplayName">App display name</Label>
              <Input
                id="appDisplayName"
                value={appDisplayName}
                onChange={(e) => onAppDisplayNameChange(e.target.value)}
                placeholder={`${scenarioName || "scenario"} Desktop`}
                className="mt-1.5"
              />
              <p className="mt-1 text-xs text-slate-400">
                Controls window titles, installer product name, and tray labels.
              </p>
            </div>
            <div>
              <Label htmlFor="iconPath">Icon path (PNG)</Label>
              <Input
                id="iconPath"
                value={iconPath}
                onChange={(e) => onIconPathChange(e.target.value)}
                placeholder="/home/you/Vrooli/scenarios/picker-wheel/icon.png"
                className="mt-1.5"
              />
              <p className="mt-1 text-xs text-slate-400">
                Optional 256px+ PNG; it will be copied into{" "}
                <code>assets/icon.png</code> for the build.
              </p>
              <div className="mt-3 flex items-center gap-3 rounded-md border border-slate-800 bg-slate-950/60 p-3">
                <div className="flex h-12 w-12 items-center justify-center rounded-md border border-slate-700 bg-slate-900">
                  {iconPreviewUrl && !iconPreviewError ? (
                    <img
                      src={iconPreviewUrl}
                      alt="Icon preview"
                      className="h-10 w-10 rounded object-contain"
                      onError={() => onIconPreviewError(true)}
                    />
                  ) : (
                    <span className="text-xs text-slate-500">No icon</span>
                  )}
                </div>
                <div className="text-xs text-slate-400">
                  {iconPreviewUrl && !iconPreviewError
                    ? "Previewing selected icon."
                    : "Preview will appear once a valid PNG path is set."}
                </div>
              </div>
            </div>
          </div>
          <div>
            <Label htmlFor="appDescription">App description</Label>
            <textarea
              id="appDescription"
              value={appDescription}
              onChange={(e) => onAppDescriptionChange(e.target.value)}
              className="mt-1.5 w-full rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-blue-600 focus:outline-none"
              rows={3}
              placeholder={`Desktop application for ${scenarioName || "your scenario"} scenario`}
            />
            <p className="mt-1 text-xs text-slate-400">
              Shown in generated README and metadata.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

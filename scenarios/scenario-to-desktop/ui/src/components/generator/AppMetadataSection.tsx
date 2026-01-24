/**
 * App metadata section for GeneratorForm.
 * Handles app display name, icon path, and description inputs.
 */

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
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
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
            Optional 256px+ PNG; it will be copied into <code>assets/icon.png</code> for the build.
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
  );
}

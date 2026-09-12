import { Bot, FolderOpen, RotateCcw, CheckCircle2, XCircle, Loader2 } from "lucide-react";
import type { AgentModeSettings as Settings } from "../../hooks/useAgentSettings";
import { usePathValidation } from "../../hooks/usePathValidation";

interface AgentModeSettingsProps {
  /** Current settings */
  settings: Settings;
  /** Called when settings change */
  onSettingsChange: (settings: Partial<Settings>) => void;
  /** Called to reset settings to defaults */
  onReset: () => void;
}

/**
 * Settings panel for configuring agent mode defaults.
 * Stores only the local workspace default; Agent Manager owns role resolution.
 */
export function AgentModeSettings({
  settings,
  onSettingsChange,
  onReset
}: AgentModeSettingsProps) {
  const { isValidating, result: pathValidation } = usePathValidation(settings.defaultProjectPath);
  const pathIsInvalid = !!settings.defaultProjectPath && pathValidation !== null && !pathValidation.valid;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-blue-500/10">
            <Bot className="h-5 w-5 text-blue-400" />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white">Agent Mode</h3>
            <p className="text-sm text-zinc-400">Configure default settings for agentic coding</p>
          </div>
        </div>
        <button
          onClick={onReset}
          className="
            flex items-center gap-1 px-3 py-1.5 rounded-lg
            text-sm text-zinc-400 hover:text-white hover:bg-zinc-700
            transition-colors
          "
        >
          <RotateCcw className="h-4 w-4" />
          Reset
        </button>
      </div>

      {/* Settings form */}
      <div className="space-y-5">
        {/* Default Project Path */}
        <div>
          <label className="flex items-center gap-2 text-sm font-medium text-zinc-300 mb-2">
            <FolderOpen className="h-4 w-4" />
            Default Project Path
          </label>
          <div className="relative">
            <input
              type="text"
              value={settings.defaultProjectPath}
              onChange={(e) => onSettingsChange({ defaultProjectPath: e.target.value })}
              placeholder="/path/to/project (optional)"
              className={`
                w-full px-3 py-2 pr-9 rounded-lg
                bg-zinc-800 border text-white placeholder-zinc-500
                focus:outline-none focus:ring-1
                ${pathIsInvalid
                  ? "border-red-500 focus:border-red-500 focus:ring-red-500"
                  : settings.defaultProjectPath && pathValidation?.valid
                    ? "border-green-500 focus:border-green-500 focus:ring-green-500"
                    : "border-zinc-700 focus:border-blue-500 focus:ring-blue-500"
                }
              `}
            />
            {settings.defaultProjectPath && (
              <span className="absolute right-2.5 top-1/2 -translate-y-1/2">
                {isValidating ? (
                  <Loader2 className="h-4 w-4 text-zinc-400 animate-spin" />
                ) : pathValidation?.valid ? (
                  <CheckCircle2 className="h-4 w-4 text-green-400" />
                ) : pathValidation !== null ? (
                  <XCircle className="h-4 w-4 text-red-400" />
                ) : null}
              </span>
            )}
          </div>
          {pathIsInvalid && pathValidation.message ? (
            <p className="mt-1 text-xs text-red-400">{pathValidation.message}</p>
          ) : (
            <p className="mt-1 text-xs text-zinc-500">
              Default directory for agent operations. Leave empty to prompt each time.
            </p>
          )}
        </div>

        <p className="text-xs text-zinc-500">Agent Manager resolves the portable coding role, concrete runner, model, and limits.</p>
      </div>

      {/* Info box */}
      <div className="p-4 rounded-lg bg-blue-500/10 border border-blue-500/20">
        <h4 className="text-sm font-medium text-blue-400 mb-2">About Agent Mode</h4>
        <p className="text-sm text-zinc-400">
          Agent mode uses coding agents like Claude Code for autonomous task execution.
          The agent can read/write files, run commands, and make changes to your codebase.
          Always review changes before approving.
        </p>
      </div>
    </div>
  );
}

export default AgentModeSettings;

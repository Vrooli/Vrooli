import { Bot, FolderOpen, Cpu, RotateCcw, CheckCircle2, XCircle, Loader2 } from "lucide-react";
import { RUNNER_OPTIONS, type RunnerType } from "../../lib/api";
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
 * Allows users to set their preferred runner, project path, etc.
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
        {/* Default Runner */}
        <div>
          <label className="flex items-center gap-2 text-sm font-medium text-zinc-300 mb-2">
            <Cpu className="h-4 w-4" />
            Default Runner
          </label>
          <select
            value={settings.defaultRunner}
            onChange={(e) => onSettingsChange({ defaultRunner: e.target.value as RunnerType })}
            className="
              w-full px-3 py-2 rounded-lg
              bg-zinc-800 border border-zinc-700
              text-white
              focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
            "
          >
            {RUNNER_OPTIONS.map(option => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <p className="mt-1 text-xs text-zinc-500">
            The default agent runner to use for new agent chats
          </p>
        </div>

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

        {/* Default Model */}
        <div>
          <label className="text-sm font-medium text-zinc-300 mb-2 block">
            Default Model (optional)
          </label>
          <input
            type="text"
            value={settings.defaultModel}
            onChange={(e) => onSettingsChange({ defaultModel: e.target.value })}
            placeholder="Use runner default"
            className="
              w-full px-3 py-2 rounded-lg
              bg-zinc-800 border border-zinc-700
              text-white placeholder-zinc-500
              focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
            "
          />
          <p className="mt-1 text-xs text-zinc-500">
            Override the runner&apos;s default model (e.g., &quot;claude-opus-4&quot;)
          </p>
        </div>

        {/* Default Max Turns */}
        <div>
          <label className="text-sm font-medium text-zinc-300 mb-2 block">
            Default Max Turns
          </label>
          <input
            type="number"
            min="0"
            value={settings.defaultMaxTurns}
            onChange={(e) => onSettingsChange({ defaultMaxTurns: parseInt(e.target.value) || 0 })}
            placeholder="0 (unlimited)"
            className="
              w-full px-3 py-2 rounded-lg
              bg-zinc-800 border border-zinc-700
              text-white placeholder-zinc-500
              focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
            "
          />
          <p className="mt-1 text-xs text-zinc-500">
            Maximum conversation turns before stopping (0 = unlimited)
          </p>
        </div>
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

import { useState } from "react";
import { X, Bot, AlertTriangle } from "lucide-react";
import type { RunnerType } from "../../lib/api";
import type { AgentModeSettings } from "../../hooks/useAgentSettings";
import { AgentStartForm } from "./AgentStartForm";

interface AgentStartModalProps {
  isOpen: boolean;
  onClose: () => void;
  onStart: (config: AgentStartConfig) => void;
  defaultSettings: AgentModeSettings;
  isLoading?: boolean;
  error?: { message: string; recovery?: string } | null;
}

export interface AgentStartConfig {
  runner_type: RunnerType;
  project_path: string;
  model: string;
  max_turns: number;
}

/**
 * Modal for configuring agent mode settings before starting.
 * Shows runner selection, project path, and optional overrides.
 */
export function AgentStartModal({
  isOpen,
  onClose,
  onStart,
  defaultSettings,
  isLoading = false,
  error = null
}: AgentStartModalProps) {
  const [runnerType, setRunnerType] = useState<RunnerType>(defaultSettings.defaultRunner);
  const [projectPath, setProjectPath] = useState(defaultSettings.defaultProjectPath);
  const [model, setModel] = useState(defaultSettings.defaultModel);
  const [maxTurns, setMaxTurns] = useState(defaultSettings.defaultMaxTurns);

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onStart({
      runner_type: runnerType,
      project_path: projectPath,
      model,
      max_turns: maxTurns
    });
  };

  const handleUseDefaults = () => {
    if (!defaultSettings.defaultProjectPath) return;
    onStart({
      runner_type: defaultSettings.defaultRunner,
      project_path: defaultSettings.defaultProjectPath,
      model: defaultSettings.defaultModel,
      max_turns: defaultSettings.defaultMaxTurns
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative w-full max-w-md mx-4 bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-zinc-700">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-blue-500/10">
              <Bot className="h-5 w-5 text-blue-400" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">Start Agent Mode</h2>
              <p className="text-sm text-zinc-400">Configure the coding agent</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-zinc-400 hover:text-white hover:bg-zinc-700 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body - extracted form */}
        <AgentStartForm
          runnerType={runnerType}
          onRunnerTypeChange={setRunnerType}
          projectPath={projectPath}
          onProjectPathChange={setProjectPath}
          model={model}
          onModelChange={setModel}
          maxTurns={maxTurns}
          onMaxTurnsChange={setMaxTurns}
          onSubmit={handleSubmit}
        />

        {/* Error display */}
        {error && (
          <div className="mx-6 mb-2 flex items-start gap-2 rounded-lg bg-red-500/10 border border-red-500/20 p-3">
            <AlertTriangle className="h-4 w-4 text-red-400 mt-0.5 shrink-0" />
            <div className="text-sm">
              <p className="text-red-400">{error.message}</p>
              {error.recovery && (
                <p className="text-zinc-400 mt-1 text-xs">{error.recovery}</p>
              )}
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="flex items-center justify-between gap-3 px-6 py-4 border-t border-zinc-700">
          {defaultSettings.defaultProjectPath ? (
            <button
              type="button"
              onClick={handleUseDefaults}
              disabled={isLoading}
              className="
                px-4 py-2 rounded-lg text-sm font-medium
                text-zinc-300 hover:text-white hover:bg-zinc-700
                transition-colors disabled:opacity-50
              "
            >
              Use Defaults
            </button>
          ) : (
            <div />
          )}
          <div className="flex gap-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="
                px-4 py-2 rounded-lg text-sm font-medium
                text-zinc-300 hover:text-white hover:bg-zinc-700
                transition-colors disabled:opacity-50
              "
            >
              Cancel
            </button>
            <button
              type="submit"
              onClick={handleSubmit}
              disabled={isLoading || !projectPath}
              className="
                px-4 py-2 rounded-lg text-sm font-medium
                bg-blue-600 text-white hover:bg-blue-500
                transition-colors disabled:opacity-50 disabled:cursor-not-allowed
              "
            >
              {isLoading ? "Starting..." : "Start Agent"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default AgentStartModal;

/**
 * AgentStartForm - Form body for the AgentStartModal.
 * Contains runner selection, project path, and advanced options.
 */

import { Cpu, FolderOpen, Zap, Loader2, CheckCircle2, XCircle } from "lucide-react";
import { RUNNER_OPTIONS, type RunnerType } from "../../lib/api";
import { usePathValidation } from "../../hooks/usePathValidation";

interface AgentStartFormProps {
  runnerType: RunnerType;
  onRunnerTypeChange: (value: RunnerType) => void;
  projectPath: string;
  onProjectPathChange: (value: string) => void;
  model: string;
  onModelChange: (value: string) => void;
  maxTurns: number;
  onMaxTurnsChange: (value: number) => void;
  onSubmit: (e: React.FormEvent) => void;
}

export function AgentStartForm({
  runnerType,
  onRunnerTypeChange,
  projectPath,
  onProjectPathChange,
  model,
  onModelChange,
  maxTurns,
  onMaxTurnsChange,
  onSubmit,
}: AgentStartFormProps) {
  const { isValidating, result: pathValidation } = usePathValidation(projectPath);
  const pathIsInvalid = !!projectPath && pathValidation !== null && !pathValidation.valid;

  return (
    <form onSubmit={onSubmit} className="p-6 space-y-5">
      {/* Runner Selection */}
      <div>
        <label className="flex items-center gap-2 text-sm font-medium text-zinc-300 mb-2">
          <Cpu className="h-4 w-4" />
          Runner
        </label>
        <div className="space-y-2">
          {RUNNER_OPTIONS.map(option => (
            <label
              key={option.value}
              className={`
                flex items-start gap-3 p-3 rounded-lg cursor-pointer
                border transition-colors
                ${runnerType === option.value
                  ? "border-blue-500 bg-blue-500/10"
                  : "border-zinc-700 hover:border-zinc-600"
                }
              `}
            >
              <input
                type="radio"
                name="runner"
                value={option.value}
                checked={runnerType === option.value}
                onChange={(e) => onRunnerTypeChange(e.target.value as RunnerType)}
                className="mt-1"
              />
              <div>
                <div className="font-medium text-white">{option.label}</div>
                <div className="text-xs text-zinc-400">{option.description}</div>
              </div>
            </label>
          ))}
        </div>
      </div>

      {/* Project Path */}
      <div>
        <label className="flex items-center gap-2 text-sm font-medium text-zinc-300 mb-2">
          <FolderOpen className="h-4 w-4" />
          Project Path
        </label>
        <div className="relative">
          <input
            type="text"
            value={projectPath}
            onChange={(e) => onProjectPathChange(e.target.value)}
            placeholder="/path/to/project"
            required
            className={`
              w-full px-3 py-2 pr-9 rounded-lg
              bg-zinc-800 border text-white placeholder-zinc-500
              focus:outline-none focus:ring-1
              ${pathIsInvalid
                ? "border-red-500 focus:border-red-500 focus:ring-red-500"
                : projectPath && pathValidation?.valid
                  ? "border-green-500 focus:border-green-500 focus:ring-green-500"
                  : "border-zinc-700 focus:border-blue-500 focus:ring-blue-500"
              }
            `}
          />
          {projectPath && (
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
            Directory where the agent will operate
          </p>
        )}
      </div>

      {/* Advanced Options */}
      <details className="group">
        <summary className="flex items-center gap-2 text-sm font-medium text-zinc-400 cursor-pointer hover:text-zinc-300">
          <Zap className="h-4 w-4" />
          Advanced Options
        </summary>
        <div className="mt-3 space-y-4 pl-6">
          {/* Model */}
          <div>
            <label className="text-sm text-zinc-400 mb-1 block">Model (optional)</label>
            <input
              type="text"
              value={model}
              onChange={(e) => onModelChange(e.target.value)}
              placeholder="Use runner default"
              className="
                w-full px-3 py-2 rounded-lg
                bg-zinc-800 border border-zinc-700
                text-white placeholder-zinc-500
                focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
                text-sm
              "
            />
          </div>

          {/* Max Turns */}
          <div>
            <label className="text-sm text-zinc-400 mb-1 block">Max Turns (0 = unlimited)</label>
            <input
              type="number"
              min="0"
              value={maxTurns}
              onChange={(e) => onMaxTurnsChange(parseInt(e.target.value) || 0)}
              className="
                w-full px-3 py-2 rounded-lg
                bg-zinc-800 border border-zinc-700
                text-white placeholder-zinc-500
                focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500
                text-sm
              "
            />
          </div>
        </div>
      </details>
    </form>
  );
}

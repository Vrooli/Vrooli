/**
 * AgentStartForm - Form body for the AgentStartModal.
 * Contains runner selection, project path, and advanced options.
 */

import { FolderOpen, Loader2, CheckCircle2, XCircle } from "lucide-react";
import { usePathValidation } from "../../hooks/usePathValidation";

interface AgentStartFormProps {
  projectPath: string;
  onProjectPathChange: (value: string) => void;
  onSubmit: (e: React.FormEvent) => void;
}

export function AgentStartForm({
  projectPath,
  onProjectPathChange,
  onSubmit,
}: AgentStartFormProps) {
  const { isValidating, result: pathValidation } = usePathValidation(projectPath);
  const pathIsInvalid = !!projectPath && pathValidation !== null && !pathValidation.valid;

  return (
    <form onSubmit={onSubmit} className="p-6 space-y-5">
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

      <p className="text-xs text-zinc-500">Agent Manager resolves the portable coding role and concrete model for this chat.</p>
    </form>
  );
}

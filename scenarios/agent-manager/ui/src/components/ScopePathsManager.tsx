import { useState } from "react";
import { Check, HelpCircle, Loader2, Plus, X } from "lucide-react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { cn } from "../lib/utils";
import {
  useProjectRootValidation,
  useScopePathValidation,
  type PathValidation,
} from "../hooks/usePathValidation";

interface ScopePathsManagerProps {
  /** The project root path - where the agent can look at code */
  projectRoot: string;
  /** Callback when project root changes */
  onProjectRootChange: (value: string) => void;
  /** Array of scope paths - where the agent can make changes */
  scopePaths: string[];
  /** Callback when scope paths change */
  onScopePathsChange: (paths: string[]) => void;
  /** Default project root (from task/run being investigated) */
  defaultProjectRoot?: string;
  /** Default scope paths (from task/run being investigated) */
  defaultScopePaths?: string[];
  /** Whether to show the project root input */
  showProjectRoot?: boolean;
  /** Custom label for scope paths section */
  scopePathsLabel?: string;
  /** Custom help text for scope paths */
  scopePathsHelp?: string;
  /** Whether scope paths are required (at least one) */
  requireScopePaths?: boolean;
}

/** Renders an inline validation status indicator */
function ValidationIndicator({ validation }: { validation: PathValidation }) {
  if (validation.status === "idle") return null;

  return (
    <div className="flex items-center gap-1.5 mt-1">
      {validation.status === "validating" && (
        <>
          <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
          <span className="text-xs text-muted-foreground">Checking...</span>
        </>
      )}
      {validation.status === "valid" && (
        <>
          <Check className="h-3 w-3 text-emerald-500" />
          <span className="text-xs text-emerald-500">{validation.message}</span>
        </>
      )}
      {validation.status === "invalid" && (
        <>
          <X className="h-3 w-3 text-destructive" />
          <span className="text-xs text-destructive">{validation.message}</span>
        </>
      )}
      {validation.status === "outside" && (
        <>
          <X className="h-3 w-3 text-destructive" />
          <span className="text-xs text-destructive">{validation.message}</span>
        </>
      )}
    </div>
  );
}

/**
 * ScopePathsManager is a reusable component for managing Project Root and multiple Scope Paths.
 *
 * - **Project Root**: The root directory the agent can look at (read access)
 * - **Scope Paths**: Directories where the agent can make changes (write access)
 *
 * Used in Quick Run dialog and Investigation modals.
 */
export function ScopePathsManager({
  projectRoot,
  onProjectRootChange,
  scopePaths,
  onScopePathsChange,
  defaultProjectRoot,
  defaultScopePaths = [],
  showProjectRoot = true,
  scopePathsLabel = "Scope Paths",
  scopePathsHelp = "Directories where the agent can make changes. Leave empty for read-only access.",
  requireScopePaths = false,
}: ScopePathsManagerProps) {
  const [scopePathInput, setScopePathInput] = useState("");

  // Path validation hooks
  const projectRootValidation = useProjectRootValidation(projectRoot);
  const scopeInputValidation = useScopePathValidation(
    scopePathInput,
    projectRoot,
    defaultProjectRoot
  );

  const handleAddScopePath = () => {
    const trimmed = scopePathInput.trim();
    if (!trimmed) return;

    // Don't add duplicates
    if (scopePaths.includes(trimmed)) {
      setScopePathInput("");
      return;
    }

    onScopePathsChange([...scopePaths, trimmed]);
    setScopePathInput("");
  };

  const handleRemoveScopePath = (path: string) => {
    onScopePathsChange(scopePaths.filter((p) => p !== path));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAddScopePath();
    }
  };

  const showDefaultHints = defaultProjectRoot || defaultScopePaths.length > 0;

  return (
    <div className="space-y-4">
      {/* Project Root */}
      {showProjectRoot && (
        <div className="space-y-2">
          <div className="flex items-center gap-1.5">
            <Label htmlFor="projectRoot">Project Root</Label>
            <div className="group relative">
              <HelpCircle className="h-3.5 w-3.5 text-muted-foreground cursor-help" />
              <div className="absolute left-0 bottom-full mb-1 z-10 hidden group-hover:block w-64 p-2 text-xs bg-popover border border-border rounded-md shadow-lg">
                The root directory containing all code the agent can <strong>look at</strong>.
                This determines the agent's read access and is used for copy-on-write sandboxing.
              </div>
            </div>
          </div>
          <Input
            id="projectRoot"
            value={projectRoot}
            onChange={(e) => onProjectRootChange(e.target.value)}
            placeholder={defaultProjectRoot || "/path/to/project"}
            className={cn(
              projectRootValidation.status === "invalid"
                ? "border-destructive focus-visible:ring-destructive"
                : projectRootValidation.status === "valid"
                ? "border-emerald-500 focus-visible:ring-emerald-500"
                : ""
            )}
          />
          <ValidationIndicator validation={projectRootValidation} />
          {defaultProjectRoot && projectRoot === "" && (
            <p className="text-xs text-muted-foreground">
              Defaults to: <code className="bg-muted px-1 py-0.5 rounded">{defaultProjectRoot}</code>
            </p>
          )}
        </div>
      )}

      {/* Scope Paths */}
      <div className="space-y-2">
        <div className="flex items-center gap-1.5">
          <Label htmlFor="scopePath">
            {scopePathsLabel}
            {requireScopePaths && <span className="text-destructive ml-0.5">*</span>}
          </Label>
          <div className="group relative">
            <HelpCircle className="h-3.5 w-3.5 text-muted-foreground cursor-help" />
            <div className="absolute left-0 bottom-full mb-1 z-10 hidden group-hover:block w-64 p-2 text-xs bg-popover border border-border rounded-md shadow-lg">
              Directories where the agent can <strong>make changes</strong>.
              In sandboxed mode, these become reserved paths that are locked for the agent's exclusive use.
            </div>
          </div>
        </div>

        <div className="flex gap-2">
          <Input
            id="scopePath"
            value={scopePathInput}
            onChange={(e) => setScopePathInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={defaultScopePaths[0] || "e.g., /path/to/project/src"}
            className={cn(
              "flex-1",
              scopeInputValidation.status === "invalid" ||
                scopeInputValidation.status === "outside"
                ? "border-destructive focus-visible:ring-destructive"
                : scopeInputValidation.status === "valid"
                ? "border-emerald-500 focus-visible:ring-emerald-500"
                : ""
            )}
          />
          <Button
            type="button"
            variant="secondary"
            onClick={handleAddScopePath}
            disabled={
              !scopePathInput.trim() ||
              scopeInputValidation.status === "invalid" ||
              scopeInputValidation.status === "outside" ||
              scopeInputValidation.status === "validating"
            }
          >
            <Plus className="h-4 w-4 mr-1" />
            Add
          </Button>
        </div>
        <ValidationIndicator validation={scopeInputValidation} />

        {/* Scope paths tags */}
        {scopePaths.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-2">
            {scopePaths.map((path) => (
              <div
                key={path}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs",
                  "border-border bg-muted/50"
                )}
              >
                <code className="truncate max-w-[300px]">{path}</code>
                <button
                  type="button"
                  onClick={() => handleRemoveScopePath(path)}
                  className="text-muted-foreground hover:text-foreground transition-colors"
                  aria-label={`Remove ${path}`}
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        )}

        <p className="text-xs text-muted-foreground">
          {scopePathsHelp}
        </p>

        {/* Default hints */}
        {showDefaultHints && scopePaths.length === 0 && defaultScopePaths.length > 0 && (
          <div className="mt-2 p-2 rounded-md border border-dashed border-border bg-muted/30">
            <p className="text-xs text-muted-foreground mb-2">
              Suggested from original run:
            </p>
            <div className="flex flex-wrap gap-1.5">
              {defaultScopePaths.map((path) => (
                <button
                  key={path}
                  type="button"
                  onClick={() => onScopePathsChange([...scopePaths, path])}
                  className={cn(
                    "inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs",
                    "border-primary/30 bg-primary/5 text-primary hover:bg-primary/10 transition-colors"
                  )}
                >
                  <Plus className="h-3 w-3" />
                  <code className="truncate max-w-[200px]">{path}</code>
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

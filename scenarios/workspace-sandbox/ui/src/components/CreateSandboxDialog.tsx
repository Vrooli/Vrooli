import { useState, useEffect, useMemo } from "react";
import {
  FolderOpen,
  Loader2,
  Plus,
  ChevronDown,
  ChevronRight,
  Check,
  X,
  Clock,
  Eye,
  Pencil,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Input, Label } from "./ui/input";
import { Select } from "./ui/select";
import type { CreateRequest, OwnerType, Sandbox, PathValidationResult } from "../lib/api";
import { validatePath } from "../lib/api";
import { SELECTORS } from "../consts/selectors";

interface CreateSandboxDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (req: CreateRequest) => void;
  isCreating: boolean;
  /** Recent sandboxes for path suggestions */
  recentSandboxes?: Sandbox[];
  /** Existing sandbox scope paths for conflict detection */
  existingScopePaths?: string[];
  /** Default project root from server config (PROJECT_ROOT env var) */
  defaultProjectRoot?: string;
}

const OWNER_TYPE_OPTIONS = [
  { value: "agent", label: "Agent", description: "AI agent making automated changes" },
  { value: "user", label: "User", description: "Human developer session" },
  { value: "task", label: "Task", description: "Tied to a specific task/ticket" },
  { value: "system", label: "System", description: "Infrastructure/automation" },
];

/** Validation states for path fields */
type PathValidation = {
  status: "idle" | "validating" | "valid" | "invalid" | "conflict" | "outside";
  message?: string;
};

/** Checks if childPath is within or equal to parentPath */
function isPathWithin(childPath: string, parentPath: string): boolean {
  const normalizedChild = childPath.replace(/\/+$/, ""); // Remove trailing slashes
  const normalizedParent = parentPath.replace(/\/+$/, "");

  return (
    normalizedChild === normalizedParent ||
    normalizedChild.startsWith(normalizedParent + "/")
  );
}

/** Paths that should never be allowed as sandbox roots */
const DANGEROUS_PATHS = ["/", "/bin", "/sbin", "/usr", "/etc", "/var", "/tmp", "/root", "/home"];

/** Validates the writable directory path using server-side validation */
function useWritablePathValidation(
  writablePath: string,
  projectRoot: string,
  defaultProjectRoot: string,
  existingScopePaths: string[] = []
): PathValidation {
  const [validation, setValidation] = useState<PathValidation>({ status: "idle" });

  useEffect(() => {
    if (!writablePath.trim()) {
      setValidation({ status: "idle" });
      return;
    }

    setValidation({ status: "validating" });

    // Debounce validation - 300ms
    const timer = setTimeout(async () => {
      const trimmedPath = writablePath.trim();

      // Quick client-side checks first (no API call needed)
      if (!trimmedPath.startsWith("/")) {
        setValidation({
          status: "invalid",
          message: "Path must be absolute (start with /)",
        });
        return;
      }

      // Reject dangerous system paths (quick check)
      const normalizedPath = trimmedPath.replace(/\/+$/, "") || "/";
      if (DANGEROUS_PATHS.includes(normalizedPath)) {
        setValidation({
          status: "invalid",
          message: "Cannot use system directories as sandbox root",
        });
        return;
      }

      // Check for conflicts with existing sandboxes (client-side)
      const hasConflict = existingScopePaths.some((existing) => {
        return (
          trimmedPath === existing ||
          trimmedPath.startsWith(existing + "/") ||
          existing.startsWith(trimmedPath + "/")
        );
      });

      if (hasConflict) {
        setValidation({
          status: "conflict",
          message: "Conflicts with an existing sandbox",
        });
        return;
      }

      // Server-side validation - checks if path exists and is a directory
      try {
        const effectiveRoot = projectRoot.trim() || defaultProjectRoot || undefined;
        const result = await validatePath(trimmedPath, effectiveRoot);

        if (!result.valid) {
          // Map server error to appropriate status
          if (result.withinProjectRoot === false) {
            setValidation({
              status: "outside",
              message: result.error || `Must be within ${effectiveRoot}`,
            });
          } else {
            setValidation({
              status: "invalid",
              message: result.error || "Invalid path",
            });
          }
          return;
        }

        setValidation({ status: "valid", message: "Path is valid" });
      } catch {
        // API error - fall back to client-side validation only
        const effectiveRoot = projectRoot.trim() || defaultProjectRoot;
        if (effectiveRoot && !isPathWithin(trimmedPath, effectiveRoot)) {
          setValidation({
            status: "outside",
            message: `Must be within ${effectiveRoot}`,
          });
          return;
        }
        // Can't verify existence, but path format is OK
        setValidation({ status: "valid", message: "Path format is valid" });
      }
    }, 300);

    return () => clearTimeout(timer);
  }, [writablePath, projectRoot, defaultProjectRoot, existingScopePaths]);

  return validation;
}

/** Validates the project root path using server-side validation */
function useProjectRootValidation(projectRoot: string): PathValidation {
  const [validation, setValidation] = useState<PathValidation>({ status: "idle" });

  useEffect(() => {
    if (!projectRoot.trim()) {
      setValidation({ status: "idle" });
      return;
    }

    setValidation({ status: "validating" });

    const timer = setTimeout(async () => {
      const trimmedPath = projectRoot.trim();

      // Quick client-side checks first
      if (!trimmedPath.startsWith("/")) {
        setValidation({
          status: "invalid",
          message: "Path must be absolute (start with /)",
        });
        return;
      }

      // Reject dangerous system paths
      const normalizedPath = trimmedPath.replace(/\/+$/, "") || "/";
      if (DANGEROUS_PATHS.includes(normalizedPath)) {
        setValidation({
          status: "invalid",
          message: "Cannot use system directories as project root",
        });
        return;
      }

      // Server-side validation - checks if path exists and is a directory
      try {
        // For project root, we don't pass a projectRoot param (it IS the root)
        const result = await validatePath(trimmedPath);

        if (!result.valid) {
          setValidation({
            status: "invalid",
            message: result.error || "Invalid path",
          });
          return;
        }

        setValidation({ status: "valid", message: "Path is valid" });
      } catch {
        // API error - fall back to format-only validation
        setValidation({ status: "valid", message: "Path format is valid" });
      }
    }, 300);

    return () => clearTimeout(timer);
  }, [projectRoot]);

  return validation;
}

/** Extracts unique recent paths from sandboxes */
function getRecentPaths(sandboxes: Sandbox[] = []): string[] {
  const paths = new Set<string>();
  sandboxes
    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    .slice(0, 10)
    .forEach((s) => {
      paths.add(s.scopePath);
      if (s.projectRoot && s.projectRoot !== s.scopePath) {
        paths.add(s.projectRoot);
      }
    });
  return Array.from(paths).slice(0, 5);
}

/** Visual path preview component */
function PathPreview({
  writablePath,
  readOnlyPath,
}: {
  writablePath: string;
  readOnlyPath: string;
}) {
  const effectiveReadOnly = readOnlyPath || writablePath;

  // Parse paths into segments for visualization
  const readOnlySegments = effectiveReadOnly.split("/").filter(Boolean);
  const writableSegments = writablePath.split("/").filter(Boolean);

  // Find where writable path diverges from read-only path
  let divergeIndex = 0;
  for (let i = 0; i < readOnlySegments.length; i++) {
    if (readOnlySegments[i] === writableSegments[i]) {
      divergeIndex = i + 1;
    } else {
      break;
    }
  }

  if (!writablePath.trim()) {
    return null;
  }

  return (
    <div className="rounded-md border border-slate-700 bg-slate-800/50 p-3 text-xs font-mono">
      <div className="flex items-center gap-2 mb-2 text-slate-400">
        <Eye className="h-3 w-3" />
        <span className="text-[10px] uppercase tracking-wider">Preview</span>
      </div>
      <div className="space-y-1">
        {readOnlySegments.map((segment, i) => {
          const isWritable =
            i >= divergeIndex - 1 &&
            writableSegments.slice(0, i + 1).join("/") === readOnlySegments.slice(0, i + 1).join("/") &&
            i === writableSegments.length - 1;
          const isInWritablePath = writableSegments
            .slice(0, i + 1)
            .join("/") === readOnlySegments.slice(0, i + 1).join("/");

          return (
            <div
              key={i}
              className="flex items-center gap-1"
              style={{ paddingLeft: `${i * 12}px` }}
            >
              {isWritable ? (
                <Pencil className="h-3 w-3 text-emerald-400" />
              ) : (
                <FolderOpen className="h-3 w-3 text-slate-500" />
              )}
              <span
                className={
                  isWritable
                    ? "text-emerald-400"
                    : isInWritablePath
                    ? "text-slate-300"
                    : "text-slate-500"
                }
              >
                {segment}/
              </span>
              {isWritable && (
                <span className="text-[10px] text-emerald-500 ml-1">writable</span>
              )}
              {!isWritable && i === readOnlySegments.length - 1 && effectiveReadOnly !== writablePath && (
                <span className="text-[10px] text-slate-500 ml-1">read-only</span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function CreateSandboxDialog({
  open,
  onOpenChange,
  onCreate,
  isCreating,
  recentSandboxes = [],
  existingScopePaths = [],
  defaultProjectRoot = "",
}: CreateSandboxDialogProps) {
  const [scopePath, setScopePath] = useState("");
  const [projectRoot, setProjectRoot] = useState("");
  const [owner, setOwner] = useState("");
  const [ownerType, setOwnerType] = useState<OwnerType>("user");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showRecentPaths, setShowRecentPaths] = useState(false);

  // Path validation - writable path must be within project root (or defaultProjectRoot)
  const scopeValidation = useWritablePathValidation(
    scopePath,
    projectRoot,
    defaultProjectRoot,
    existingScopePaths
  );
  const projectRootValidation = useProjectRootValidation(projectRoot);

  // Dynamic placeholders based on server config
  const writablePlaceholder = defaultProjectRoot
    ? `${defaultProjectRoot}/src/components`
    : "/home/user/myproject/src/components";
  const projectRootPlaceholder = defaultProjectRoot || "/home/user/myproject";

  // Recent paths for quick selection
  const recentPaths = useMemo(() => getRecentPaths(recentSandboxes), [recentSandboxes]);

  const canSubmit =
    scopePath.trim() &&
    scopeValidation.status !== "invalid" &&
    scopeValidation.status !== "conflict" &&
    scopeValidation.status !== "outside" &&
    projectRootValidation.status !== "invalid" &&
    !isCreating;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!canSubmit) return;

    onCreate({
      scopePath: scopePath.trim(),
      // Use defaultProjectRoot from server config if user didn't provide a custom one
      projectRoot: projectRoot.trim() || defaultProjectRoot || undefined,
      owner: owner.trim() || undefined,
      ownerType,
    });
  };

  const handleClose = () => {
    if (!isCreating) {
      onOpenChange(false);
      // Reset form
      setScopePath("");
      setProjectRoot("");
      setOwner("");
      setOwnerType("user");
      setShowAdvanced(false);
      setShowRecentPaths(false);
    }
  };

  const handleSelectRecentPath = (path: string, field: "scope" | "projectRoot") => {
    if (field === "scope") {
      setScopePath(path);
    } else {
      setProjectRoot(path);
    }
    setShowRecentPaths(false);
  };

  /** Renders validation indicator */
  const ValidationIndicator = ({ validation }: { validation: PathValidation }) => {
    if (validation.status === "idle") return null;

    return (
      <div className="flex items-center gap-1.5 mt-1">
        {validation.status === "validating" && (
          <>
            <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
            <span className="text-xs text-slate-400">Checking...</span>
          </>
        )}
        {validation.status === "valid" && (
          <>
            <Check className="h-3 w-3 text-emerald-400" />
            <span className="text-xs text-emerald-400">{validation.message}</span>
          </>
        )}
        {validation.status === "invalid" && (
          <>
            <X className="h-3 w-3 text-red-400" />
            <span className="text-xs text-red-400">{validation.message}</span>
          </>
        )}
        {validation.status === "conflict" && (
          <>
            <X className="h-3 w-3 text-amber-400" />
            <span className="text-xs text-amber-400">{validation.message}</span>
          </>
        )}
        {validation.status === "outside" && (
          <>
            <X className="h-3 w-3 text-red-400" />
            <span className="text-xs text-red-400">{validation.message}</span>
          </>
        )}
      </div>
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg" data-testid={SELECTORS.createDialog}>
        <DialogClose onClose={handleClose} />

        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FolderOpen className="h-5 w-5 text-slate-400" />
            Create Sandbox
          </DialogTitle>
          <DialogDescription>
            Create an isolated workspace for safe file modifications. Changes can be
            reviewed and selectively applied.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          {/* Writable Directory - Required (renamed from Scope Path) */}
          <div className="space-y-2">
            <Label htmlFor="scopePath">
              Writable Directory <span className="text-red-400">*</span>
            </Label>
            <div className="relative">
              <Input
                id="scopePath"
                placeholder={writablePlaceholder}
                value={scopePath}
                onChange={(e) => setScopePath(e.target.value)}
                required
                autoFocus
                className={
                  scopeValidation.status === "invalid" ||
                  scopeValidation.status === "conflict" ||
                  scopeValidation.status === "outside"
                    ? "border-red-500 focus:border-red-500"
                    : scopeValidation.status === "valid"
                    ? "border-emerald-500 focus:border-emerald-500"
                    : ""
                }
                data-testid={SELECTORS.scopePathInput}
              />
              {recentPaths.length > 0 && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-1 top-1/2 -translate-y-1/2 h-7 px-2 text-slate-400 hover:text-slate-200"
                  onClick={() => setShowRecentPaths(!showRecentPaths)}
                >
                  <Clock className="h-3.5 w-3.5" />
                </Button>
              )}
            </div>
            <ValidationIndicator validation={scopeValidation} />
            <p className="text-xs text-slate-500">
              Files here can be modified. Must be within the read-only context below (or equal to it).
            </p>

            {/* Recent paths dropdown */}
            {showRecentPaths && recentPaths.length > 0 && (
              <div className="absolute z-10 mt-1 w-full rounded-md border border-slate-700 bg-slate-800 shadow-lg">
                <div className="p-2 text-xs text-slate-400 border-b border-slate-700">
                  Recent paths
                </div>
                {recentPaths.map((path) => (
                  <button
                    key={path}
                    type="button"
                    className="w-full px-3 py-2 text-left text-sm text-slate-300 hover:bg-slate-700 truncate"
                    onClick={() => handleSelectRecentPath(path, "scope")}
                  >
                    {path}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Read-Only Context - Optional (renamed from Project Root) */}
          <div className="space-y-2">
            <Label htmlFor="projectRoot">Read-Only Context (optional)</Label>
            <Input
              id="projectRoot"
              placeholder={projectRootPlaceholder}
              value={projectRoot}
              onChange={(e) => setProjectRoot(e.target.value)}
              className={
                projectRootValidation.status === "invalid"
                  ? "border-red-500 focus:border-red-500"
                  : projectRootValidation.status === "valid"
                  ? "border-emerald-500 focus:border-emerald-500"
                  : ""
              }
              data-testid={SELECTORS.projectRootInput}
            />
            <ValidationIndicator validation={projectRootValidation} />
            <p className="text-xs text-slate-500">
              {defaultProjectRoot
                ? `Parent directory with read-only access. Defaults to ${defaultProjectRoot} if empty.`
                : "Parent directory with read-only access. The writable directory must be inside this path."}
            </p>
          </div>

          {/* Path Preview */}
          {scopePath.trim() && (
            <PathPreview writablePath={scopePath} readOnlyPath={projectRoot} />
          )}

          {/* Advanced Options Toggle */}
          <button
            type="button"
            className="flex items-center gap-1.5 text-sm text-slate-400 hover:text-slate-200 transition-colors"
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            {showAdvanced ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            Advanced Options
          </button>

          {/* Advanced Options - Hidden by default */}
          {showAdvanced && (
            <div className="space-y-4 pl-4 border-l-2 border-slate-700">
              {/* Owner */}
              <div className="space-y-2">
                <Label htmlFor="owner">Owner</Label>
                <Input
                  id="owner"
                  placeholder="claude-agent, matt, task-12345"
                  value={owner}
                  onChange={(e) => setOwner(e.target.value)}
                  data-testid={SELECTORS.ownerInput}
                />
                <p className="text-xs text-slate-500">
                  Identifier for audit trail (who created this sandbox)
                </p>
              </div>

              {/* Owner Type with descriptions */}
              <div className="space-y-2">
                <Label>Owner Type</Label>
                <Select
                  value={ownerType}
                  onValueChange={(v) => setOwnerType(v as OwnerType)}
                  options={OWNER_TYPE_OPTIONS}
                  data-testid={SELECTORS.ownerTypeSelect}
                />
                <p className="text-xs text-slate-500">
                  {OWNER_TYPE_OPTIONS.find((o) => o.value === ownerType)?.description}
                </p>
              </div>
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isCreating}
              data-testid={SELECTORS.cancelCreate}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!canSubmit}
              data-testid={SELECTORS.submitCreate}
            >
              {isCreating ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Sandbox
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

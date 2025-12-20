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
  /** Existing sandbox reserved paths for conflict detection */
  existingReservedPaths?: string[];
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

/** Validates the reserved directory path using server-side validation */
function useReservedPathValidation(
  reservedPath: string,
  projectRoot: string,
  defaultProjectRoot: string,
  existingReservedPaths: string[] = []
): PathValidation {
  const [validation, setValidation] = useState<PathValidation>({ status: "idle" });

  useEffect(() => {
    if (!reservedPath.trim()) {
      setValidation({ status: "idle" });
      return;
    }

    setValidation({ status: "validating" });

    // Debounce validation - 300ms
    const timer = setTimeout(async () => {
      const trimmedPath = reservedPath.trim();

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
      const hasConflict = existingReservedPaths.some((existing) => {
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
  }, [reservedPath, projectRoot, defaultProjectRoot, existingReservedPaths]);

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
      paths.add(s.reservedPath || s.scopePath);
      if (s.scopePath) {
        paths.add(s.scopePath);
      }
      if (s.projectRoot && s.projectRoot !== (s.reservedPath || s.scopePath)) {
        paths.add(s.projectRoot);
      }
    });
  return Array.from(paths).slice(0, 5);
}

/** Visual path preview component */
function PathPreview({
  reservedPath,
  projectRoot,
}: {
  reservedPath: string;
  projectRoot: string;
}) {
  const effectiveRoot = projectRoot || reservedPath;

  // Parse paths into segments for visualization
  const rootSegments = effectiveRoot.split("/").filter(Boolean);
  const reservedSegments = reservedPath.split("/").filter(Boolean);

  // Find where reserved path diverges from root.
  let divergeIndex = 0;
  for (let i = 0; i < rootSegments.length; i++) {
    if (rootSegments[i] === reservedSegments[i]) {
      divergeIndex = i + 1;
    } else {
      break;
    }
  }

  if (!reservedPath.trim()) {
    return null;
  }

  return (
    <div className="rounded-md border border-slate-700 bg-slate-800/50 p-3 text-xs font-mono">
      <div className="flex items-center gap-2 mb-2 text-slate-400">
        <Eye className="h-3 w-3" />
        <span className="text-[10px] uppercase tracking-wider">Preview</span>
      </div>
      <div className="space-y-1">
        {rootSegments.map((segment, i) => {
          const isReserved =
            i >= divergeIndex - 1 &&
            reservedSegments.slice(0, i + 1).join("/") === rootSegments.slice(0, i + 1).join("/") &&
            i === reservedSegments.length - 1;
          const isInReservedPath =
            reservedSegments.slice(0, i + 1).join("/") === rootSegments.slice(0, i + 1).join("/");

          return (
            <div
              key={i}
              className="flex items-center gap-1"
              style={{ paddingLeft: `${i * 12}px` }}
            >
              {isReserved ? (
                <Pencil className="h-3 w-3 text-emerald-400" />
              ) : (
                <FolderOpen className="h-3 w-3 text-slate-500" />
              )}
              <span
                className={
                  isReserved
                    ? "text-emerald-400"
                    : isInReservedPath
                    ? "text-slate-300"
                    : "text-slate-500"
                }
              >
                {segment}/
              </span>
              {isReserved && (
                <span className="text-[10px] text-emerald-500 ml-1">reserved</span>
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
  existingReservedPaths = [],
  defaultProjectRoot = "",
}: CreateSandboxDialogProps) {
  const [reservedPath, setReservedPath] = useState("");
  const [projectRoot, setProjectRoot] = useState("");
  const [owner, setOwner] = useState("");
  const [ownerType, setOwnerType] = useState<OwnerType>("user");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showRecentPaths, setShowRecentPaths] = useState(false);
  const [mountScopePath, setMountScopePath] = useState("");

  // Path validation - reserved path must be within project root (or defaultProjectRoot)
  const reservedValidation = useReservedPathValidation(
    reservedPath,
    projectRoot,
    defaultProjectRoot,
    existingReservedPaths
  );
  const projectRootValidation = useProjectRootValidation(projectRoot);

  // Dynamic placeholders based on server config
  const reservedPlaceholder = defaultProjectRoot
    ? `${defaultProjectRoot}/scenarios/my-scenario`
    : "/home/user/myproject/scenarios/my-scenario";
  const projectRootPlaceholder = defaultProjectRoot || "/home/user/myproject";

  // Recent paths for quick selection
  const recentPaths = useMemo(() => getRecentPaths(recentSandboxes), [recentSandboxes]);

  const effectiveProjectRoot = projectRoot.trim() || defaultProjectRoot;
  const effectiveScopePath = mountScopePath.trim() || effectiveProjectRoot;

  const canSubmit =
    !!effectiveScopePath &&
    reservedPath.trim() &&
    reservedValidation.status !== "invalid" &&
    reservedValidation.status !== "conflict" &&
    reservedValidation.status !== "outside" &&
    projectRootValidation.status !== "invalid" &&
    !isCreating;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!canSubmit) return;

    const scopePath = effectiveScopePath;

    onCreate({
      scopePath,
      reservedPath: reservedPath.trim(),
      projectRoot: effectiveProjectRoot || undefined,
      owner: owner.trim() || undefined,
      ownerType,
    });
  };

  const handleClose = () => {
    if (!isCreating) {
      onOpenChange(false);
      // Reset form
      setReservedPath("");
      setProjectRoot("");
      setOwner("");
      setOwnerType("user");
      setShowAdvanced(false);
      setShowRecentPaths(false);
      setMountScopePath("");
    }
  };

  const handleSelectRecentPath = (path: string, field: "reserved" | "projectRoot") => {
    if (field === "reserved") {
      setReservedPath(path);
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
          {/* Reserved Directory - Required */}
          <div className="space-y-2">
            <Label htmlFor="reservedPath">
              Reserved Directory <span className="text-red-400">*</span>
            </Label>
            <div className="relative">
              <Input
                id="reservedPath"
                placeholder={reservedPlaceholder}
                value={reservedPath}
                onChange={(e) => setReservedPath(e.target.value)}
                required
                autoFocus
                className={
                  reservedValidation.status === "invalid" ||
                  reservedValidation.status === "conflict" ||
                  reservedValidation.status === "outside"
                    ? "border-red-500 focus:border-red-500"
                    : reservedValidation.status === "valid"
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
            <ValidationIndicator validation={reservedValidation} />
            <p className="text-xs text-slate-500">
              Prevents overlapping sandboxes by reserving this subtree. By default, approval applies only changes under this directory.
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
                    onClick={() => handleSelectRecentPath(path, "reserved")}
                  >
                    {path}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Project Root - Optional */}
          <div className="space-y-2">
            <Label htmlFor="projectRoot">Project Root (optional)</Label>
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
                ? `Defaults to ${defaultProjectRoot} if empty. Reserved Directory must be inside this root.`
                : "Reserved Directory must be inside this root."}
            </p>
          </div>

          {/* Path Preview */}
          {reservedPath.trim() && (
            <PathPreview
              reservedPath={reservedPath}
              projectRoot={projectRoot.trim() || defaultProjectRoot}
            />
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
              {/* Mount Scope (advanced) */}
              <div className="space-y-2">
                <Label htmlFor="mountScopePath">Mount Scope (advanced)</Label>
                <Input
                  id="mountScopePath"
                  placeholder={projectRootPlaceholder}
                  value={mountScopePath}
                  onChange={(e) => setMountScopePath(e.target.value)}
                />
                <p className="text-xs text-slate-500">
                  By default the sandbox mounts the full project root for easy reference. Set this to a subdirectory to sandbox less of the tree.
                </p>
              </div>

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

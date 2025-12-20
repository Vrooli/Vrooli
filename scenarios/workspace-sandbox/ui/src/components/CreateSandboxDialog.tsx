import { useState, useEffect, useMemo } from "react";
import {
  FolderOpen,
  Loader2,
  Plus,
  ChevronDown,
  ChevronRight,
  Check,
  X,
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

/** Paths that should never be allowed as project roots or reserved prefixes */
const DANGEROUS_PATHS = ["/", "/bin", "/sbin", "/usr", "/etc", "/var", "/tmp", "/root", "/home"];

/** Validates a reserved prefix path using server-side validation */
function useReservedPathValidation(
  reservedPath: string,
  projectRoot: string,
  defaultProjectRoot: string,
  existingReservedPaths: string[] = [],
  existingReservedPathsKey: string = ""
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
          message: "Cannot use system directories as reserved paths",
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
  }, [reservedPath, projectRoot, defaultProjectRoot, existingReservedPathsKey]);

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
      const reserved = s.reservedPaths?.length ? s.reservedPaths : [s.reservedPath || s.scopePath];
      reserved.forEach((p) => p && paths.add(p));
      if (s.scopePath) {
        paths.add(s.scopePath);
      }
      if (s.projectRoot && s.projectRoot !== (s.reservedPath || s.scopePath)) {
        paths.add(s.projectRoot);
      }
    });
  return Array.from(paths).slice(0, 5);
}

function getRecentProjectRoots(sandboxes: Sandbox[] = []): string[] {
  const roots = new Set<string>();
  sandboxes
    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    .slice(0, 20)
    .forEach((s) => {
      if (s.projectRoot) roots.add(s.projectRoot);
    });
  return Array.from(roots).slice(0, 5);
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
  const [reservedPathInput, setReservedPathInput] = useState("");
  const [reservedPaths, setReservedPaths] = useState<string[]>([]);
  const [projectRoot, setProjectRoot] = useState("");
  const [owner, setOwner] = useState("");
  const [ownerType, setOwnerType] = useState<OwnerType>("user");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [mountScopePath, setMountScopePath] = useState("");

  // Path validation - new reserved path must be within project root and not conflict with existing sandboxes
  const existingReservedPathsKey = useMemo(() => {
    if (existingReservedPaths.length === 0) return "";
    return existingReservedPaths
      .map((p) => p.trim())
      .filter(Boolean)
      .sort()
      .join("|");
  }, [existingReservedPaths]);
  const reservedInputValidation = useReservedPathValidation(
    reservedPathInput,
    projectRoot,
    defaultProjectRoot,
    [...existingReservedPaths, ...reservedPaths],
    existingReservedPathsKey
  );
  const projectRootValidation = useProjectRootValidation(projectRoot);

  // Dynamic placeholders based on server config
  const reservedPlaceholder = defaultProjectRoot
    ? `${defaultProjectRoot}/scenarios/my-scenario`
    : "/home/user/myproject/scenarios/my-scenario";
  const projectRootPlaceholder = defaultProjectRoot || "/home/user/myproject";

  // Recent paths for quick selection
  const recentPaths = useMemo(() => getRecentPaths(recentSandboxes), [recentSandboxes]);
  const recentProjectRoots = useMemo(
    () => getRecentProjectRoots(recentSandboxes),
    [recentSandboxes]
  );

  const effectiveProjectRoot = projectRoot.trim() || defaultProjectRoot;
  const effectiveScopePath = mountScopePath.trim() || effectiveProjectRoot;

  const reservedListValidation = useMemo<PathValidation>(() => {
    if (reservedPaths.length === 0) return { status: "idle" };

    // Basic client-side checks for the list; server will re-validate on create.
    for (const p of reservedPaths) {
      const trimmed = p.trim();
      if (!trimmed.startsWith("/")) {
        return { status: "invalid", message: "Reserved paths must be absolute (start with /)" };
      }
      const normalizedPath = trimmed.replace(/\/+$/, "") || "/";
      if (DANGEROUS_PATHS.includes(normalizedPath)) {
        return { status: "invalid", message: "Cannot reserve system directories" };
      }
      if (effectiveProjectRoot && !isPathWithin(trimmed, effectiveProjectRoot)) {
        return { status: "outside", message: `Reserved paths must be within ${effectiveProjectRoot}` };
      }
    }

    // Prevent overlaps within the reserved list itself.
    for (let i = 0; i < reservedPaths.length; i++) {
      for (let j = i + 1; j < reservedPaths.length; j++) {
        const a = reservedPaths[i].trim().replace(/\/+$/, "");
        const b = reservedPaths[j].trim().replace(/\/+$/, "");
        if (!a || !b) continue;
        if (a === b || a.startsWith(b + "/") || b.startsWith(a + "/")) {
          return { status: "conflict", message: "Reserved paths overlap each other" };
        }
      }
    }

    return { status: "valid", message: `${reservedPaths.length} reserved path(s)` };
  }, [reservedPaths, effectiveProjectRoot]);

  const effectiveReservedPaths = useMemo(() => {
    const result = [...reservedPaths];
    const pending = reservedPathInput.trim();
    if (pending && reservedInputValidation.status === "valid") {
      result.push(pending);
    }
    // De-dupe while preserving order
    return result.filter((p, idx) => result.indexOf(p) === idx);
  }, [reservedPaths, reservedPathInput, reservedInputValidation.status]);

  const canSubmit =
    !!effectiveScopePath &&
    effectiveReservedPaths.length > 0 &&
    reservedListValidation.status !== "invalid" &&
    reservedListValidation.status !== "conflict" &&
    reservedListValidation.status !== "outside" &&
    projectRootValidation.status !== "invalid" &&
    !isCreating;

  const handleAddReservedPath = () => {
    const trimmed = reservedPathInput.trim();
    if (!trimmed) return;
    if (reservedInputValidation.status !== "valid") return;
    setReservedPaths((prev) => (prev.includes(trimmed) ? prev : [...prev, trimmed]));
    setReservedPathInput("");
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!canSubmit) return;

    const scopePath = effectiveScopePath;

    onCreate({
      scopePath,
      reservedPath: effectiveReservedPaths[0],
      reservedPaths: effectiveReservedPaths,
      projectRoot: effectiveProjectRoot || undefined,
      owner: owner.trim() || undefined,
      ownerType,
    });
  };

  const handleClose = () => {
    if (!isCreating) {
      onOpenChange(false);
      // Reset form
      setReservedPathInput("");
      setReservedPaths([]);
      setProjectRoot("");
      setOwner("");
      setOwnerType("user");
      setShowAdvanced(false);
      setShowRecentPaths(false);
      setShowRecentProjectRoots(false);
      setMountScopePath("");
    }
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
              Reserved Path(s) <span className="text-red-400">*</span>
            </Label>
            <div className="relative">
              <div className="flex gap-2">
                <Input
                  id="reservedPath"
                  placeholder={reservedPlaceholder}
                  value={reservedPathInput}
                  onChange={(e) => setReservedPathInput(e.target.value)}
                  list={recentPaths.length > 0 ? "reservedPathSuggestions" : undefined}
                  autoFocus
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      handleAddReservedPath();
                    }
                  }}
                  className={
                    reservedInputValidation.status === "invalid" ||
                    reservedInputValidation.status === "conflict" ||
                    reservedInputValidation.status === "outside"
                      ? "border-red-500 focus:border-red-500"
                      : reservedInputValidation.status === "valid"
                      ? "border-emerald-500 focus:border-emerald-500"
                      : ""
                  }
                  data-testid={SELECTORS.scopePathInput}
                />
                <Button
                  type="button"
                  variant="secondary"
                  onClick={handleAddReservedPath}
                  disabled={!reservedPathInput.trim() || reservedInputValidation.status !== "valid"}
                >
                  Add
                </Button>
              </div>
            </div>
            {recentPaths.length > 0 && (
              <datalist id="reservedPathSuggestions">
                {recentPaths.map((path) => (
                  <option key={path} value={path} />
                ))}
              </datalist>
            )}
            {reservedPaths.length > 0 && <ValidationIndicator validation={reservedListValidation} />}
            <ValidationIndicator validation={reservedInputValidation} />

            {reservedPaths.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {reservedPaths.map((p) => (
                  <div
                    key={p}
                    className="inline-flex items-center gap-2 rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200"
                  >
                    <span className="font-mono truncate max-w-[360px]">{p}</span>
                    <button
                      type="button"
                      className="text-slate-400 hover:text-slate-200"
                      onClick={() => setReservedPaths((prev) => prev.filter((x) => x !== p))}
                      aria-label={`Remove ${p}`}
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
            )}
            <p className="text-xs text-slate-500">
              Reserves one or more subtrees to prevent overlapping sandboxes. Approval defaults to changes under these paths unless you explicitly approve more.
            </p>
          </div>

          {/* Project Root - Optional */}
          <div className="space-y-2">
            <Label htmlFor="projectRoot">Project Root (full repo mount)</Label>
            <div className="relative">
              <Input
                id="projectRoot"
                placeholder={projectRootPlaceholder}
                value={projectRoot}
                onChange={(e) => setProjectRoot(e.target.value)}
                list={recentProjectRoots.length > 0 ? "projectRootSuggestions" : undefined}
                className={
                  projectRootValidation.status === "invalid"
                    ? "border-red-500 focus:border-red-500"
                    : projectRootValidation.status === "valid"
                    ? "border-emerald-500 focus:border-emerald-500"
                    : ""
                }
                data-testid={SELECTORS.projectRootInput}
              />
            </div>
            {recentProjectRoots.length > 0 && (
              <datalist id="projectRootSuggestions">
                {recentProjectRoots.map((path) => (
                  <option key={path} value={path} />
                ))}
              </datalist>
            )}
            <ValidationIndicator validation={projectRootValidation} />
            <p className="text-xs text-slate-500">
              {defaultProjectRoot
                ? `Defaults to ${defaultProjectRoot} if empty. The sandbox mounts the full project root by default; reserved paths must be inside it.`
                : "The sandbox mounts the full project root by default; reserved paths must be inside it."}
            </p>
          </div>

          {/* Path Preview */}
          {effectiveReservedPaths[0] && (
            <PathPreview
              reservedPath={effectiveReservedPaths[0]}
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
                  By default the sandbox mounts the full project root. Set this to a subdirectory if you want a narrower copy-on-write mount.
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

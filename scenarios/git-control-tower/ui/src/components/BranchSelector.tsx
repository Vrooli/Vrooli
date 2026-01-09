import { useMemo, useRef, useState, useEffect } from "react";
import {
  GitBranch,
  ChevronDown,
  Plus,
  Upload,
  AlertTriangle,
  ArrowUp,
  ArrowDown
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { BottomSheet } from "./ui/bottom-sheet";
import type {
  RepoStatus,
  SyncStatusResponse,
  RepoBranchesResponse,
  CreateBranchRequest,
  BranchCreateResponse,
  SwitchBranchRequest,
  BranchSwitchResponse,
  PublishBranchRequest,
  BranchPublishResponse,
  BranchWarning
} from "../lib/api";

export interface BranchActions {
  branches?: RepoBranchesResponse;
  isLoading: boolean;
  createBranch: (request: CreateBranchRequest) => Promise<BranchCreateResponse>;
  switchBranch: (request: SwitchBranchRequest) => Promise<BranchSwitchResponse>;
  publishBranch: (request: PublishBranchRequest) => Promise<BranchPublishResponse>;
  isCreating: boolean;
  isSwitching: boolean;
  isPublishing: boolean;
}

interface BranchSelectorProps {
  status?: RepoStatus;
  syncStatus?: SyncStatusResponse;
  actions: BranchActions;
  variant?: "desktop" | "mobile";
}

type PendingAction =
  | { type: "switch"; request: SwitchBranchRequest }
  | { type: "create"; request: CreateBranchRequest }
  | { type: "publish"; request: PublishBranchRequest };

export function BranchSelector({
  status,
  syncStatus,
  actions,
  variant = "desktop"
}: BranchSelectorProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createFrom, setCreateFrom] = useState("");
  const [checkoutAfterCreate, setCheckoutAfterCreate] = useState(true);
  const [warning, setWarning] = useState<BranchWarning | null>(null);
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const dropdownRef = useRef<HTMLDivElement | null>(null);

  const isDirty = Boolean(
    status &&
      (status.summary.staged > 0 ||
        status.summary.unstaged > 0 ||
        status.summary.untracked > 0 ||
        status.summary.conflicts > 0)
  );

  const currentBranch =
    syncStatus?.branch || status?.branch.head || actions.branches?.current || "";
  const currentUpstream = syncStatus?.upstream ?? status?.branch.upstream ?? "";
  const ahead = syncStatus?.ahead ?? status?.branch.ahead ?? 0;
  const behind = syncStatus?.behind ?? status?.branch.behind ?? 0;
  const showPublish = currentBranch !== "" && currentUpstream === "";

  useEffect(() => {
    if (variant !== "desktop" || !open) return;

    const handleClickOutside = (event: MouseEvent) => {
      if (!dropdownRef.current) return;
      if (!dropdownRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    window.addEventListener("mousedown", handleClickOutside);
    return () => window.removeEventListener("mousedown", handleClickOutside);
  }, [open, variant]);

  const { locals, remotes } = useMemo(() => {
    const branches = actions.branches;
    if (!branches) return { locals: [], remotes: [] };

    const term = search.trim().toLowerCase();
    const filter = (name: string) => (term ? name.toLowerCase().includes(term) : true);

    return {
      locals: branches.locals.filter((branch) => filter(branch.name)),
      remotes: branches.remotes.filter((branch) => filter(branch.name))
    };
  }, [actions.branches, search]);

  const resetWarning = () => {
    setWarning(null);
    setPendingAction(null);
  };

  const handleSwitch = async (name: string) => {
    resetWarning();
    try {
      const result = await actions.switchBranch({ name });
      if (result.success) {
        setOpen(false);
        return;
      }
      if (result.warning) {
        setWarning(result.warning);
        setPendingAction({ type: "switch", request: { name } });
        return;
      }
      if (result.error) {
        setWarning({ message: result.error });
      }
    } catch (error) {
      setWarning({ message: error instanceof Error ? error.message : "Switch failed" });
    }
  };

  const handleCreate = async () => {
    resetWarning();
    if (!createName.trim()) {
      setWarning({ message: "Branch name is required" });
      return;
    }
    try {
      const result = await actions.createBranch({
        name: createName.trim(),
        from: createFrom.trim() || undefined,
        checkout: checkoutAfterCreate
      });
      if (result.success) {
        setCreateName("");
        setCreateFrom("");
        setShowCreate(false);
        setOpen(false);
        return;
      }
      if (result.warning) {
        setWarning(result.warning);
        setPendingAction({
          type: "create",
          request: {
            name: createName.trim(),
            from: createFrom.trim() || undefined,
            checkout: checkoutAfterCreate
          }
        });
        return;
      }
      if (result.validation_errors?.length) {
        setWarning({ message: result.validation_errors.join("; ") });
        return;
      }
      if (result.error) {
        setWarning({ message: result.error });
      }
    } catch (error) {
      setWarning({ message: error instanceof Error ? error.message : "Create failed" });
    }
  };

  const handlePublish = async (fetch = false) => {
    resetWarning();
    try {
      const result = await actions.publishBranch({ fetch });
      if (result.success) {
        setOpen(false);
        return;
      }
      if (result.warning) {
        setWarning(result.warning);
        setPendingAction({ type: "publish", request: { fetch } });
        return;
      }
      if (result.error) {
        setWarning({ message: result.error });
      }
    } catch (error) {
      setWarning({ message: error instanceof Error ? error.message : "Publish failed" });
    }
  };

  const handleConfirmWarning = async () => {
    if (!warning || !pendingAction) return;
    if (pendingAction.type === "switch") {
      const request = {
        ...pendingAction.request,
        allow_dirty: warning.requires_confirmation || undefined,
        track_remote: warning.requires_tracking || pendingAction.request.track_remote
      } as SwitchBranchRequest;
      setPendingAction(null);
      await handleSwitchWithRequest(request);
      return;
    }
    if (pendingAction.type === "create") {
      const request = {
        ...pendingAction.request,
        allow_dirty: warning.requires_confirmation || undefined
      } as CreateBranchRequest;
      setPendingAction(null);
      await handleCreateWithRequest(request);
      return;
    }
    if (pendingAction.type === "publish") {
      const request = {
        ...pendingAction.request,
        fetch: warning.requires_fetch || pendingAction.request.fetch
      } as PublishBranchRequest;
      setPendingAction(null);
      await handlePublishWithRequest(request);
    }
  };

  const handleSwitchWithRequest = async (request: SwitchBranchRequest) => {
    resetWarning();
    try {
      const result = await actions.switchBranch(request);
      if (result.success) {
        setOpen(false);
        return;
      }
      if (result.warning) {
        setWarning(result.warning);
        setPendingAction({ type: "switch", request });
        return;
      }
      if (result.error) {
        setWarning({ message: result.error });
      }
    } catch (error) {
      setWarning({ message: error instanceof Error ? error.message : "Switch failed" });
    }
  };

  const handleCreateWithRequest = async (request: CreateBranchRequest) => {
    resetWarning();
    try {
      const result = await actions.createBranch(request);
      if (result.success) {
        setCreateName("");
        setCreateFrom("");
        setShowCreate(false);
        setOpen(false);
        return;
      }
      if (result.warning) {
        setWarning(result.warning);
        setPendingAction({ type: "create", request });
        return;
      }
      if (result.validation_errors?.length) {
        setWarning({ message: result.validation_errors.join("; ") });
        return;
      }
      if (result.error) {
        setWarning({ message: result.error });
      }
    } catch (error) {
      setWarning({ message: error instanceof Error ? error.message : "Create failed" });
    }
  };

  const handlePublishWithRequest = async (request: PublishBranchRequest) => {
    resetWarning();
    try {
      const result = await actions.publishBranch(request);
      if (result.success) {
        setOpen(false);
        return;
      }
      if (result.warning) {
        setWarning(result.warning);
        setPendingAction({ type: "publish", request });
        return;
      }
      if (result.error) {
        setWarning({ message: result.error });
      }
    } catch (error) {
      setWarning({ message: error instanceof Error ? error.message : "Publish failed" });
    }
  };

  const content = (
    <div className="space-y-3" data-testid="branch-selector-panel">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <GitBranch className="h-4 w-4 text-slate-400" />
          <span className="text-sm font-mono text-slate-200">
            {currentBranch || "Detached"}
          </span>
          {ahead > 0 && (
            <Badge variant="info" className="gap-1">
              <ArrowUp className="h-3 w-3" />
              {ahead}
            </Badge>
          )}
          {behind > 0 && (
            <Badge variant="warning" className="gap-1">
              <ArrowDown className="h-3 w-3" />
              {behind}
            </Badge>
          )}
        </div>
        {showPublish && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => handlePublish(false)}
            disabled={actions.isPublishing}
            className="gap-1"
            data-testid="branch-publish-button"
          >
            <Upload className="h-3.5 w-3.5" />
            Publish
          </Button>
        )}
      </div>
      {currentUpstream && (
        <div className="text-[11px] text-slate-500 font-mono">→ {currentUpstream}</div>
      )}

      <input
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Search branches..."
        className="w-full px-3 py-2 text-xs bg-slate-900/60 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        data-testid="branch-search"
      />

      <div className="flex items-center justify-between">
        <span className="text-xs uppercase tracking-wide text-slate-500">Create</span>
        <button
          className="text-xs text-slate-300 hover:text-slate-100 flex items-center gap-1"
          onClick={() => setShowCreate((prev) => !prev)}
          data-testid="branch-create-toggle"
        >
          <Plus className="h-3 w-3" />
          {showCreate ? "Hide" : "New"}
        </button>
      </div>

      {showCreate && (
        <div className="space-y-2 rounded-md border border-slate-800 bg-slate-900/40 p-3">
          <input
            value={createName}
            onChange={(e) => setCreateName(e.target.value)}
            placeholder="Branch name"
            className="w-full px-3 py-2 text-xs bg-slate-900/60 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="branch-create-name"
          />
          <input
            value={createFrom}
            onChange={(e) => setCreateFrom(e.target.value)}
            placeholder="Base (optional)"
            className="w-full px-3 py-2 text-xs bg-slate-900/60 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="branch-create-from"
          />
          <label className="flex items-center gap-2 text-xs text-slate-400">
            <input
              type="checkbox"
              checked={checkoutAfterCreate}
              onChange={(e) => setCheckoutAfterCreate(e.target.checked)}
              className="h-3.5 w-3.5 rounded border-slate-600 bg-slate-800"
              data-testid="branch-create-checkout"
            />
            Checkout after create
          </label>
          <Button
            size="sm"
            onClick={handleCreate}
            disabled={actions.isCreating}
            data-testid="branch-create-submit"
          >
            Create branch
          </Button>
        </div>
      )}

      <div className="space-y-2">
        {actions.isLoading && (
          <div className="text-[11px] text-slate-500">Loading branches...</div>
        )}
        <div className="text-xs uppercase tracking-wide text-slate-500">Local branches</div>
        <div className="space-y-1">
          {locals.length === 0 && (
            <div className="text-xs text-slate-600">No local branches</div>
          )}
          {locals.map((branch) => (
            <button
              key={branch.name}
              className={`w-full flex items-center justify-between px-3 py-2 rounded-md text-left text-xs transition-colors ${
                branch.is_current
                  ? "bg-slate-800 text-slate-100"
                  : "hover:bg-slate-800/60 text-slate-300"
              }`}
              onClick={() => !branch.is_current && handleSwitch(branch.name)}
              disabled={actions.isSwitching || branch.is_current}
              data-testid={`branch-local-${branch.name}`}
            >
              <span className="font-mono truncate">{branch.name}</span>
              <span className="text-[10px] text-slate-500">
                {branch.upstream ? `→ ${branch.upstream}` : ""}
              </span>
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-2">
        <div className="text-xs uppercase tracking-wide text-slate-500">Remote branches</div>
        <div className="space-y-1">
          {remotes.length === 0 && (
            <div className="text-xs text-slate-600">No remote branches</div>
          )}
          {remotes.map((branch) => (
            <button
              key={branch.name}
              className="w-full flex items-center justify-between px-3 py-2 rounded-md text-left text-xs text-slate-300 hover:bg-slate-800/60"
              onClick={() => handleSwitch(branch.name)}
              disabled={actions.isSwitching}
              data-testid={`branch-remote-${branch.name}`}
            >
              <span className="font-mono truncate">{branch.name}</span>
              <span className="text-[10px] text-slate-500">remote</span>
            </button>
          ))}
        </div>
      </div>

      {warning && (
        <div className="rounded-md border border-amber-700/60 bg-amber-950/40 p-3 text-xs text-amber-200" data-testid="branch-warning">
          <div className="flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 mt-0.5 text-amber-400" />
            <div className="space-y-2">
              <p>{warning.message}</p>
              {warning.dirty_summary && (
                <p className="text-[11px] text-amber-300">
                  Dirty: {warning.dirty_summary.staged} staged, {warning.dirty_summary.unstaged} modified, {warning.dirty_summary.untracked} untracked, {warning.dirty_summary.conflicts} conflicts
                </p>
              )}
              {(warning.requires_confirmation || warning.requires_tracking || warning.requires_fetch) && pendingAction && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleConfirmWarning}
                  className="border-amber-600/70 text-amber-100 hover:bg-amber-900/40"
                  data-testid="branch-warning-confirm"
                >
                  {warning.requires_tracking
                    ? "Track and switch"
                    : warning.requires_fetch
                    ? "Fetch and recheck"
                    : "Proceed anyway"}
                </Button>
              )}
            </div>
          </div>
        </div>
      )}

      {isDirty && !warning && (
        <div className="text-[11px] text-slate-500">
          Uncommitted changes detected. Switching may be blocked.
        </div>
      )}
    </div>
  );

  if (variant === "mobile") {
    return (
      <>
        <button
          className="flex items-center gap-1 min-w-0"
          onClick={() => setOpen(true)}
          data-testid="branch-selector-trigger"
        >
          <GitBranch className="h-4 w-4 text-slate-400 flex-shrink-0" />
          <span className="font-mono text-sm text-slate-200 truncate">
            {currentBranch || "Detached"}
          </span>
          {ahead > 0 && (
            <Badge variant="info" className="gap-0.5 flex-shrink-0">
              <ArrowUp className="h-3 w-3" />
              {ahead}
            </Badge>
          )}
          {behind > 0 && (
            <Badge variant="warning" className="gap-0.5 flex-shrink-0">
              <ArrowDown className="h-3 w-3" />
              {behind}
            </Badge>
          )}
          <ChevronDown className="h-3.5 w-3.5 text-slate-500" />
        </button>
        <BottomSheet
          isOpen={open}
          onClose={() => setOpen(false)}
          title="Branches"
        >
          {content}
        </BottomSheet>
      </>
    );
  }

  return (
    <div className="relative" ref={dropdownRef} data-testid="branch-selector">
      <button
        className="flex items-center gap-1.5 px-2 py-1 rounded-md hover:bg-slate-800 transition-colors"
        onClick={() => setOpen((prev) => !prev)}
        data-testid="branch-selector-trigger"
      >
        <GitBranch className="h-4 w-4 text-slate-400" />
        <span className="font-mono text-sm text-slate-200">
          {currentBranch || "Detached"}
        </span>
        {ahead > 0 && (
          <Badge variant="info" className="gap-0.5">
            <ArrowUp className="h-3 w-3" />
            {ahead}
          </Badge>
        )}
        {behind > 0 && (
          <Badge variant="warning" className="gap-0.5">
            <ArrowDown className="h-3 w-3" />
            {behind}
          </Badge>
        )}
        <ChevronDown className="h-3.5 w-3.5 text-slate-500" />
      </button>

      {open && (
        <div className="absolute left-0 mt-2 w-[560px] max-w-[calc(100vw-2rem)] max-h-[calc(100vh-8rem)] overflow-y-auto rounded-lg border border-slate-800 bg-slate-950/95 p-4 shadow-xl z-50">
          {content}
        </div>
      )}
    </div>
  );
}

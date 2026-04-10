import { useMemo, useState } from "react";
import {
  ChevronLeft,
  FolderOpen,
  DownloadCloud,
  Trash2
} from "lucide-react";
import { Button } from "./ui/button";
import type {
  RepoListResponse,
  RepoOpenRequest,
  RepoCloneRequest,
  RepoActiveRequest,
  RepoMutationResponse,
  RepoRemoveResponse
} from "../lib/api";

export interface RepoActions {
  repos?: RepoListResponse;
  isLoading: boolean;
  openRepo: (request: RepoOpenRequest) => Promise<RepoMutationResponse>;
  cloneRepo: (request: RepoCloneRequest) => Promise<RepoMutationResponse>;
  setActiveRepo: (request: RepoActiveRequest) => Promise<RepoMutationResponse>;
  removeRepo: (id: number) => Promise<RepoRemoveResponse>;
  isOpening: boolean;
  isCloning: boolean;
  isSettingActive: boolean;
  isRemoving: boolean;
}

interface RepoSelectorProps {
  repoActions: RepoActions;
  onRepoChange?: (repoId: string | null) => void;
  onBack: () => void;
}

export function RepoSelector({ repoActions, onRepoChange, onBack }: RepoSelectorProps) {
  const [repoSearch, setRepoSearch] = useState("");
  const [openPath, setOpenPath] = useState("");
  const [cloneUrl, setCloneUrl] = useState("");
  const [cloneDestination, setCloneDestination] = useState("");
  const [repoError, setRepoError] = useState<string | null>(null);

  const repoListSource = repoActions.repos?.repos;
  const activeRepoId = repoActions.repos?.active_id;

  const repoList = useMemo(() => repoListSource ?? [], [repoListSource]);

  const filteredRepos = useMemo(() => {
    const term = repoSearch.trim().toLowerCase();
    if (!term) return repoList;
    return repoList.filter((repo) => {
      const name = repo.name?.toLowerCase() ?? "";
      const path = repo.path?.toLowerCase() ?? "";
      const remote = repo.remote_url?.toLowerCase() ?? "";
      return name.includes(term) || path.includes(term) || remote.includes(term);
    });
  }, [repoList, repoSearch]);

  const handleRepoSelect = async (id: number) => {
    setRepoError(null);
    try {
      const result = await repoActions.setActiveRepo({ id });
      if (result.repo?.id) {
        onRepoChange?.(String(result.repo.id));
      }
      onBack();
    } catch (error) {
      setRepoError(error instanceof Error ? error.message : "Unable to switch repository");
    }
  };

  const handleRepoOpen = async () => {
    setRepoError(null);
    if (!openPath.trim()) {
      setRepoError("Repository path is required");
      return;
    }
    try {
      const result = await repoActions.openRepo({ path: openPath.trim() });
      if (result.repo?.id) {
        onRepoChange?.(String(result.repo.id));
      }
      setOpenPath("");
      onBack();
    } catch (error) {
      setRepoError(error instanceof Error ? error.message : "Unable to open repository");
    }
  };

  const handleRepoClone = async () => {
    setRepoError(null);
    if (!cloneUrl.trim()) {
      setRepoError("Clone URL is required");
      return;
    }
    if (!cloneDestination.trim()) {
      setRepoError("Destination path is required");
      return;
    }
    try {
      const result = await repoActions.cloneRepo({
        url: cloneUrl.trim(),
        destination: cloneDestination.trim()
      });
      if (result.repo?.id) {
        onRepoChange?.(String(result.repo.id));
      }
      setCloneUrl("");
      setCloneDestination("");
      onBack();
    } catch (error) {
      setRepoError(error instanceof Error ? error.message : "Unable to clone repository");
    }
  };

  const handleRepoRemove = async (id: number) => {
    setRepoError(null);
    try {
      await repoActions.removeRepo(id);
    } catch (error) {
      setRepoError(error instanceof Error ? error.message : "Unable to remove repository");
    }
  };

  return (
    <div className="space-y-3" data-testid="repo-selector-panel">
      <div className="flex items-center justify-between">
        <button
          className="flex items-center gap-1 text-xs text-slate-300 hover:text-slate-100"
          onClick={onBack}
          data-testid="repo-back-button"
        >
          <ChevronLeft className="h-3 w-3" />
          Back to branches
        </button>
        <span className="text-[11px] uppercase tracking-wide text-slate-500">Repositories</span>
      </div>

      <input
        value={repoSearch}
        onChange={(e) => setRepoSearch(e.target.value)}
        placeholder="Search repositories..."
        className="w-full px-3 py-2 text-xs bg-slate-900/60 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        data-testid="repo-search"
      />

      <div className="space-y-2">
        {repoActions.isLoading && (
          <div className="text-[11px] text-slate-500">Loading repositories...</div>
        )}
        {filteredRepos.length === 0 && (
          <div className="text-xs text-slate-600">No repositories yet</div>
        )}
        {filteredRepos.map((repo) => {
          const isActive = repo.id === activeRepoId;
          return (
            <div
              key={repo.id}
              className={`flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-left text-xs transition-colors ${
                isActive
                  ? "border-slate-700 bg-slate-800 text-slate-100"
                  : "border-slate-800 bg-slate-900/40 text-slate-300 hover:bg-slate-800/60"
              }`}
            >
              <button
                className="flex-1 min-w-0 text-left"
                onClick={() => !isActive && handleRepoSelect(repo.id)}
                disabled={repoActions.isSettingActive || isActive}
                data-testid={`repo-item-${repo.id}`}
              >
                <div className="font-mono truncate">{repo.name}</div>
                <div className="text-[10px] text-slate-500 truncate" title={repo.path}>
                  {repo.path}
                </div>
                {repo.remote_url && (
                  <div className="text-[10px] text-slate-500 truncate">
                    {repo.remote_url}
                  </div>
                )}
              </button>
              <button
                className="text-slate-500 hover:text-rose-300 disabled:opacity-40"
                onClick={() => handleRepoRemove(repo.id)}
                disabled={repoActions.isRemoving || isActive}
                title={isActive ? "Active repository cannot be removed" : "Remove repository"}
                data-testid={`repo-remove-${repo.id}`}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          );
        })}
      </div>

      <div className="space-y-2 rounded-md border border-slate-800 bg-slate-900/40 p-3">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-slate-500">
          <FolderOpen className="h-3.5 w-3.5" />
          Open existing
        </div>
        <input
          value={openPath}
          onChange={(e) => setOpenPath(e.target.value)}
          placeholder="/path/to/repository"
          className="w-full px-3 py-2 text-xs bg-slate-900/60 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          data-testid="repo-open-path"
        />
        <Button
          size="sm"
          onClick={handleRepoOpen}
          disabled={repoActions.isOpening}
          data-testid="repo-open-submit"
        >
          Open repository
        </Button>
      </div>

      <div className="space-y-2 rounded-md border border-slate-800 bg-slate-900/40 p-3">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-slate-500">
          <DownloadCloud className="h-3.5 w-3.5" />
          Clone repository
        </div>
        <input
          value={cloneUrl}
          onChange={(e) => setCloneUrl(e.target.value)}
          placeholder="git@github.com:org/repo.git"
          className="w-full px-3 py-2 text-xs bg-slate-900/60 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          data-testid="repo-clone-url"
        />
        <input
          value={cloneDestination}
          onChange={(e) => setCloneDestination(e.target.value)}
          placeholder="/path/to/destination"
          className="w-full px-3 py-2 text-xs bg-slate-900/60 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          data-testid="repo-clone-destination"
        />
        <Button
          size="sm"
          onClick={handleRepoClone}
          disabled={repoActions.isCloning}
          data-testid="repo-clone-submit"
        >
          Clone repository
        </Button>
      </div>

      {repoError && (
        <div className="rounded-md border border-amber-700/60 bg-amber-950/40 p-3 text-xs text-amber-200">
          {repoError}
        </div>
      )}
    </div>
  );
}

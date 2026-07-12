// DOC: docs/concepts/ARCHITECTURE.md
// App orchestrates the 3-pane git-control-tower UI. See the Architecture
// doc for component boundaries and the operational targets (OT-P1-002 etc.)
// driving future work; performance characteristics are tracked in
// docs/perf/.
import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { emitShortcutIntent, HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER } from "@vrooli/iframe-bridge";
import { StatusHeader } from "./components/StatusHeader";
import { MobileHeader } from "./components/MobileHeader";
import { MobileNav } from "./components/MobileNav";
import { FileList } from "./components/FileList";
import { HistoryFileList } from "./components/HistoryFileList";
import { DiffViewer } from "./components/DiffViewer";
import { CommitPanel } from "./components/CommitPanel";
import { GitHistory } from "./components/GitHistory";
import { DiscardConfirmationModal, type DiscardFile } from "./components/DiscardConfirmationModal";
import { DeleteConfirmationModal } from "./components/DeleteConfirmationModal";
import { UpstreamInfoModal } from "./components/UpstreamInfoModal";
import { FileSearchModal } from "./components/FileSearchModal";
import { MobileFileSearch } from "./components/MobileFileSearch";
import { RelatedFilesPanel } from "./components/RelatedFilesPanel";
import { type LayoutPreset, type LayoutSection } from "./components/LayoutSettingsModal";
import { SettingsModal } from "./components/SettingsModal";
import { ScenarioReviewPanel } from "./components/ScenarioReviewPanel";
import { useGlobalKeydown, useIsMobile, useUrlState, parseUrlState, useScenarioReviewState } from "./hooks";
import type { UrlState, ReviewTab } from "./hooks";
import type { GroupingRule } from "./components/FileList";
import { fetchSyncStatus } from "./lib/api";
import type { RepoHistoryEntry, ViewMode, FileViewMode, GroupingRulesConfig, PrecommitRunResult, CommitRequest } from "./lib/api";
import { getFileTypeInfo } from "./lib/fileTypes";
import type { ViewingCommit } from "./components/HistoryModeHeader";
import { computeNextSelection, layoutOrder, type SelectionEntry } from "./AppSelection";
import {
  useHealth,
  useRepoStatus,
  useRepoHistory,
  useDiff,
  useSyncStatus,
  useApprovedChanges,
  useApprovedChangesPreview,
  useStageFiles,
  useUnstageFiles,
  useCommit,
  usePrecommitConfig,
  useSavePrecommitConfig,
  useRunPrecommit,
  useStreamPrecommit,
  useDiscardFiles,
  useIgnoreFile,
  usePush,
  usePull,
  useSaveFileContent,
  useBranches,
  useCreateBranch,
  useSwitchBranch,
  usePublishBranch,
  useDeletePath,
  useRepos,
  useOpenRepo,
  useCloneRepo,
  useSetActiveRepo,
  useRemoveRepo,
  useRepoSelection,
  useGroupingRules,
  useSaveGroupingRules,
  queryKeys
} from "./lib/hooks";

type GroupingRuleLike = Partial<GroupingRule> & { prefix?: string };

function isLayoutSection(value: string): value is LayoutSection {
  return value === "changes" || value === "diff" || value === "commit" || value === "history" || value === "review";
}

function isLayoutPreset(value: string): value is LayoutPreset {
  return value === "classic" || value === "split" || value === "bottom";
}

function isGroupingRuleLike(value: unknown): value is GroupingRuleLike {
  return typeof value === "object" && value !== null;
}

function isPresent<T>(value: T | null): value is T {
  return value !== null;
}

export default function App() {
  const queryClient = useQueryClient();
  const { repoId, setRepoId } = useRepoSelection();
  const isMobile = useIsMobile();
  const mainRef = useRef<HTMLDivElement | null>(null);
  const stackRef = useRef<HTMLDivElement | null>(null);
  // Refs for inner grid containers driven by changesHeight / historyHeight.
  // During panel drag we write style.gridTemplateRows imperatively on these
  // refs so App's React state isn't updated on every mousemove. The state is
  // committed once on mouseup so it still persists to localStorage.
  const sidebarGridRef = useRef<HTMLDivElement | null>(null);
  const topStackGridRef = useRef<HTMLDivElement | null>(null);

  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") return 320;
    const stored = Number(localStorage.getItem("gct.sidebarWidth"));
    return Number.isFinite(stored) && stored > 0 ? stored : 320;
  });
  const [changesHeight, setChangesHeight] = useState(() => {
    if (typeof window === "undefined") return 420;
    const stored = Number(localStorage.getItem("gct.changesHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 420;
  });
  const [historyHeight, setHistoryHeight] = useState(() => {
    if (typeof window === "undefined") return 200;
    const stored = Number(localStorage.getItem("gct.historyHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 200;
  });
  const [changesCollapsed, setChangesCollapsed] = useState(false);
  const [historyCollapsed, setHistoryCollapsed] = useState(false);
  const [commitCollapsed, setCommitCollapsed] = useState(false);
  const [isResizingStack, setIsResizingStack] = useState(false);
  const [isResizingSplit, setIsResizingSplit] = useState(false);
  const [isResizingHistory, setIsResizingHistory] = useState(false);
  const [historyLimit, setHistoryLimit] = useState(50);
  const historyMaxLimit = 200;
  const [historySearch, setHistorySearch] = useState("");
  const [historyScopeFilter, setHistoryScopeFilter] = useState<string | null>(null);
  const [historyWorkingSetOnly, setHistoryWorkingSetOnly] = useState(false);
  const [historyGrepPrefix, setHistoryGrepPrefix] = useState<string | null>(null);
  const [isHistoryFiltersOpen, setIsHistoryFiltersOpen] = useState(false);
  const stackResize = useRef<
    | { mode: "left" | "right"; start: number; max: number }
    | { mode: "bottom"; top: number; height: number }
    | null
  >(null);
  const splitResize = useRef<{ top: number; height: number } | null>(null);
  const historyResize = useRef<{ bottom: number } | null>(null);
  const sidebarMinWidth = 200;
  const diffMinWidth = 320;
  const [fileViewMode, setFileViewMode] = useState<FileViewMode>("flat");
  const [groupingRules, setGroupingRules] = useState<GroupingRule[]>([]);
  const [groupingLoadedKey, setGroupingLoadedKey] = useState<string | null>(null);
  const [groupingDefaultsPending, setGroupingDefaultsPending] = useState(false);
  const [layoutPreset, setLayoutPreset] = useState<LayoutPreset>("classic");
  const [primaryPanel, setPrimaryPanel] = useState<LayoutSection>("diff");
  const [layoutLoadedKey, setLayoutLoadedKey] = useState<string | null>(null);
  const [stackHeight, setStackHeight] = useState(() => {
    if (typeof window === "undefined") return 320;
    const stored = Number(localStorage.getItem("gct.stackHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 320;
  });
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [reviewScenarioSlug, setReviewScenarioSlug] = useState("");
  // URL overrides are captured once on mount so the hook can prioritize URL params
  const urlInitOverridesRef = useRef<{ activeTab?: ReviewTab; agentRunId?: string | null } | undefined>(undefined);
  const scenarioReview = useScenarioReviewState(reviewScenarioSlug, {
    urlOverrides: urlInitOverridesRef.current,
  });
  // Mobile-specific state: which panel is currently active on mobile
  const [mobileActivePanel, setMobileActivePanel] = useState<LayoutSection>(() => {
    if (typeof window === "undefined") return "changes";
    const stored = localStorage.getItem("gct.mobileActivePanel");
    if (stored && isLayoutSection(stored)) {
      return stored;
    }
    return "changes";
  });
  const [pushNotice, setPushNotice] = useState<{
    tone: "success" | "info" | "warning";
    message: string;
  } | null>(null);
  const [warningNotice, setWarningNotice] = useState<{
    message: string;
    details?: string;
  } | null>(null);
  const [isUpstreamInfoOpen, setIsUpstreamInfoOpen] = useState(false);
  // File search state
  const [isFileSearchOpen, setIsFileSearchOpen] = useState(false);
  // Track if viewing a non-changed file (any file from search)
  const [isViewingAnyFile, setIsViewingAnyFile] = useState(false);
  // Related files panel state
  const [showRelatedFiles, setShowRelatedFiles] = useState(false);
  const [relatedFilesForPath, setRelatedFilesForPath] = useState<string | undefined>();
  const [scrollToFile, setScrollToFile] = useState<string | undefined>();
  // File blame mode (viewing history for a specific file)
  const [viewingFileBlame, setViewingFileBlame] = useState<{
    path: string;
    filename: string;
  } | null>(null);
  // Pending file/folder delete confirmation
  const [pendingDeletePath, setPendingDeletePath] = useState<{
    path: string;
    isDir: boolean;
  } | null>(null);
  // Track whether URL initialization is complete (state variable, not ref, for proper batching)
  const [urlInitComplete, setUrlInitComplete] = useState(false);

  useEffect(() => {
    if (fileViewMode !== "grouped" || groupingRules.length === 0) {
      setHistoryScopeFilter(null);
    }
  }, [fileViewMode, groupingRules.length]);

  useEffect(() => {
    if (!pushNotice) return;
    const timeout = window.setTimeout(() => {
      setPushNotice(null);
    }, 4000);
    return () => window.clearTimeout(timeout);
  }, [pushNotice]);

  useEffect(() => {
    if (!warningNotice) return;
    const timeout = window.setTimeout(() => {
      setWarningNotice(null);
    }, 4000);
    return () => window.clearTimeout(timeout);
  }, [warningNotice]);

  // Selected file state
  const selectionKey = useCallback(
    (entry: { path: string; staged: boolean }) => `${entry.staged ? "1" : "0"}:${entry.path}`,
    []
  );
  const [selectedFile, setSelectedFile] = useState<string | undefined>();
  const [selectedIsStaged, setSelectedIsStaged] = useState(false);
  const [selectedIsUntracked, setSelectedIsUntracked] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<Array<{ path: string; staged: boolean }>>([]);
  const lastSelectedKeyRef = useRef<string | null>(null);
  const [confirmingDiscard, setConfirmingDiscard] = useState<string | null>(null);
  const [pendingDiscardFiles, setPendingDiscardFiles] = useState<DiscardFile[] | null>(null);
  const [confirmingIgnore, setConfirmingIgnore] = useState<string | null>(null);
  const [_lastCommitHash, setLastCommitHash] = useState<string | undefined>();
  const [commitError, setCommitError] = useState<string | undefined>();
  const [precommitFailure, setPrecommitFailure] = useState<PrecommitRunResult | null>(null);
  const [pendingPrecommitCommit, setPendingPrecommitCommit] = useState<CommitRequest | null>(null);
  // Set true right before an intentional pre-commit stream abort (Commit Anyway) so
  // handleCommit does not surface the resulting AbortError as a commit error.
  const commitAnywayRef = useRef(false);
  // Survives the mobile Changes-panel unmount so its scroll position is restored
  // when the operator switches away and back (desktop keeps panels mounted).
  const changesScrollTopRef = useRef(0);
  const [commitMessage, setCommitMessage] = useState(
    () => localStorage.getItem("gct.commitMessage") ?? ""
  );
  // History mode: when viewing a previous commit
  const [viewingCommit, setViewingCommit] = useState<ViewingCommit | null>(null);
  // View mode for diff viewer
  const [viewMode, setViewMode] = useState<ViewMode>("diff");

  // Track whether we've initialized from URL (prevents clearing URL on first render)
  const initializedFromUrlRef = useRef(false);
  // Track whether URL specified a primary panel (prevents localStorage from overriding it)
  const urlSetPrimaryRef = useRef(false);

  // URL state management - handle browser back/forward and initial state
  const handleUrlStateChange = useCallback((state: UrlState) => {
    if (state.file) {
      const isAnyFile = state.anyFile === true;
      setSelectedFile(state.file);
      setSelectedIsStaged(state.staged ?? false);
      setSelectedIsUntracked(false);
      setSelectedFiles([]);
      setIsViewingAnyFile(isAnyFile);
      setViewingCommit(null);
      // Default mode: "source" for anyFile, "diff" for changed files
      if (!state.mode) {
        setViewMode(isAnyFile ? "source" : "diff");
      }
    }
    if (state.mode) {
      setViewMode(state.mode);
    }
    if (state.panel === "related" && state.file) {
      setShowRelatedFiles(true);
      setRelatedFilesForPath(state.file);
    } else if (state.panel === "changes") {
      setShowRelatedFiles(false);
      setRelatedFilesForPath(undefined);
    }
    if (state.commit) {
      // Note: Full history mode restoration would require fetching commit details
      // For now, we just store the hash - user can click on the commit in history to fully restore
    }
    if (state.primary) {
      if (isLayoutSection(state.primary)) {
        setPrimaryPanel(state.primary);
      }
    } else {
      setPrimaryPanel("diff");
    }
    if (state.reviewScenario) {
      setReviewScenarioSlug(state.reviewScenario);
    }
    if (state.reviewTab) {
      scenarioReview.update({ activeTab: state.reviewTab });
    }
    if (state.agentRunId) {
      scenarioReview.update({ agentRunId: state.agentRunId });
    }
  }, [scenarioReview]);

  const { updateState: updateUrlState } = useUrlState({
    onStateChange: handleUrlStateChange
  });

  // Initialize state from URL on mount
  useEffect(() => {
    const initialState = parseUrlState(window.location.search);
    if (initialState.primary) {
      urlSetPrimaryRef.current = true;
    }
    // Capture URL overrides for the scenario review hook
    if (initialState.reviewTab || initialState.agentRunId) {
      urlInitOverridesRef.current = {
        activeTab: initialState.reviewTab,
        agentRunId: initialState.agentRunId ?? null,
      };
    }
    if (initialState.file || initialState.commit || initialState.primary || initialState.reviewScenario || initialState.agentRunId) {
      handleUrlStateChange(initialState);
    }
    // Mark as initialized so URL update effect can run
    initializedFromUrlRef.current = true;
    // Mark URL init complete (batched with other state updates from handleUrlStateChange)
    setUrlInitComplete(true);
    // Only run on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update URL when relevant state changes (skip until initialized from URL)
  useEffect(() => {
    // Don't update URL until we've processed initial URL state
    if (!initializedFromUrlRef.current) return;

    const urlState: UrlState = {};

    if (selectedFile) {
      urlState.file = selectedFile;
    }
    if (selectedIsStaged) {
      urlState.staged = true;
    }
    if (viewMode && viewMode !== "diff") {
      urlState.mode = viewMode;
    }
    if (showRelatedFiles) {
      urlState.panel = "related";
    }
    if (viewingCommit?.hash) {
      urlState.commit = viewingCommit.hash;
    }

    if (primaryPanel !== "diff") {
      urlState.primary = primaryPanel;
    }
    if (reviewScenarioSlug) {
      urlState.reviewScenario = reviewScenarioSlug;
    }
    if (scenarioReview.state.activeTab && scenarioReview.state.activeTab !== "overview") {
      urlState.reviewTab = scenarioReview.state.activeTab;
    }
    if (isViewingAnyFile) {
      urlState.anyFile = true;
    }
    if (scenarioReview.state.agentRunId) {
      urlState.agentRunId = scenarioReview.state.agentRunId;
    }

    updateUrlState(urlState);
  }, [selectedFile, selectedIsStaged, viewMode, showRelatedFiles, viewingCommit?.hash, primaryPanel, reviewScenarioSlug, scenarioReview.state.activeTab, isViewingAnyFile, scenarioReview.state.agentRunId, updateUrlState]);

  const stackPosition: "left" | "right" | "bottom" =
    layoutPreset === "bottom" ? "bottom" : layoutPreset === "split" ? "right" : "left";
  const stackPanels = useMemo(
    () => primaryPanel === "review" ? layoutOrder : layoutOrder.filter((section) => section !== primaryPanel),
    [primaryPanel]
  );
  const stackSlots = useMemo(() => {
    const remaining = new Set(stackPanels);
    const fallbackPanel: LayoutSection = "diff";
    const middle = remaining.has("history") ? "history" : stackPanels[0] ?? fallbackPanel;
    remaining.delete(middle);
    const bottom = remaining.has("commit")
      ? "commit"
      : stackPanels[1] ?? stackPanels[0] ?? fallbackPanel;
    remaining.delete(bottom);
    const top = Array.from(remaining)[0] ?? middle;
    return { top, middle, bottom };
  }, [stackPanels]);
  const collapsedBySection = useMemo(
    () => ({
      changes: changesCollapsed,
      history: historyCollapsed,
      commit: commitCollapsed,
      diff: false,
      review: false
    }),
    [changesCollapsed, historyCollapsed, commitCollapsed]
  );
  const topPanel = stackSlots.top;
  const middlePanel = stackSlots.middle;
  const bottomPanel = stackSlots.bottom;
  const topCollapsed = collapsedBySection[topPanel];
  const middleCollapsed = collapsedBySection[middlePanel];
  const bottomCollapsed = collapsedBySection[bottomPanel];
  const topStackCollapsed = topCollapsed && middleCollapsed;

  // Queries
  const healthQuery = useHealth();
  const statusQuery = useRepoStatus(repoId);
  // Always fetch entry details for commit viewing and blame mode filtering
  const historyNeedsDetails = true;
  const historyGrepPattern = historyGrepPrefix ? `${historyGrepPrefix} p` : undefined;
  const historyEffectiveLimit = historyGrepPrefix ? 1000 : historyLimit;
  const historyQuery = useRepoHistory(historyEffectiveLimit, historyNeedsDetails, repoId, historyGrepPattern, historyNeedsDetails);
  const syncStatusQuery = useSyncStatus(repoId);
  const approvedChangesQuery = useApprovedChanges(repoId);
  const diffQuery = useDiff(
    selectedFile,
    viewingCommit || isViewingAnyFile ? false : selectedIsStaged,
    viewingCommit || isViewingAnyFile ? false : selectedIsUntracked,
    viewingCommit?.hash,
    isViewingAnyFile || viewMode === "preview" ? "source" : viewMode,
    isViewingAnyFile,
    repoId
  );
  const pushTargetRef =
    syncStatusQuery.data?.upstream ?? statusQuery.data?.branch.upstream;
  const pushSourceBranch = statusQuery.data?.branch.head;
  const upstreamAhead = syncStatusQuery.data?.ahead ?? statusQuery.data?.branch.ahead ?? 0;
  const upstreamBehind = syncStatusQuery.data?.behind ?? statusQuery.data?.branch.behind ?? 0;
  const hasUpstream =
    syncStatusQuery.data?.has_upstream ?? Boolean(statusQuery.data?.branch.upstream);
  const canAmend = hasUpstream && upstreamAhead > 0;
  const amendDisabledReason = hasUpstream
    ? upstreamAhead > 0
      ? undefined
      : "Last commit already on upstream"
    : "Set upstream before amending";

  // Mutations
  const stageMutation = useStageFiles(repoId);
  const unstageMutation = useUnstageFiles(repoId);
  const commitMutation = useCommit(repoId);
  const precommitConfigQuery = usePrecommitConfig(repoId);
  const savePrecommitConfigMutation = useSavePrecommitConfig(repoId);
  const runPrecommitMutation = useRunPrecommit(repoId);
  const precommitStream = useStreamPrecommit(repoId);
  const discardMutation = useDiscardFiles(repoId);
  const ignoreMutation = useIgnoreFile(repoId);
  const pushMutation = usePush(repoId);
  const pullMutation = usePull(repoId);
  const saveFileContentMutation = useSaveFileContent(repoId);
  const approvedPreviewMutation = useApprovedChangesPreview(repoId);
  const branchesQuery = useBranches(repoId);
  const createBranchMutation = useCreateBranch(repoId);
  const switchBranchMutation = useSwitchBranch(repoId);
  const publishBranchMutation = usePublishBranch(repoId);
  const deletePathMutation = useDeletePath(repoId);
  const reposQuery = useRepos();
  const openRepoMutation = useOpenRepo();
  const cloneRepoMutation = useCloneRepo();
  const setActiveRepoMutation = useSetActiveRepo();
  const removeRepoMutation = useRemoveRepo();
  const groupingRulesQuery = useGroupingRules(repoId);
  const saveGroupingRulesMutation = useSaveGroupingRules(repoId);

  const isStaging = stageMutation.isPending || unstageMutation.isPending;
  // Per-path in-flight tracking so only touched rows spin (replaces the global
  // isStaging flag for row-level loading); bulk buttons still use isStaging.
  const pendingPaths = useMemo(() => {
    const set = new Set<string>();
    if (stageMutation.isPending) stageMutation.variables?.paths?.forEach((p) => set.add(p));
    if (unstageMutation.isPending) unstageMutation.variables?.paths?.forEach((p) => set.add(p));
    if (discardMutation.isPending) discardMutation.variables?.paths?.forEach((p) => set.add(p));
    return set;
  }, [
    stageMutation.isPending,
    stageMutation.variables,
    unstageMutation.isPending,
    unstageMutation.variables,
    discardMutation.isPending,
    discardMutation.variables,
  ]);
  const isDeleting = deletePathMutation.isPending;
  const isDiscarding = discardMutation.isPending;
  const isIgnoring = ignoreMutation.isPending;
  const repoDir = statusQuery.data?.repo_dir;
  const repoKey = useMemo(
    () => (repoDir ? encodeURIComponent(repoDir) : "unknown"),
    [repoDir]
  );

  // Handlers
  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.health });
    queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    queryClient.invalidateQueries({
      queryKey: queryKeys.repoHistory(historyLimit, historyNeedsDetails, repoId)
    });
    queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
    queryClient.invalidateQueries({ queryKey: queryKeys.approvedChanges(repoId) });
    queryClient.invalidateQueries({ queryKey: queryKeys.branches(repoId) });
    if (selectedFile) {
      queryClient.invalidateQueries({ queryKey: ["repo", "diff", repoId ?? "default"] });
    }
  }, [
    historyLimit,
    historyNeedsDetails,
    queryClient,
    repoId,
    selectedFile
  ]);

  const branchActions = useMemo(
    () => ({
      branches: branchesQuery.data,
      isLoading: branchesQuery.isLoading,
      createBranch: createBranchMutation.mutateAsync,
      switchBranch: switchBranchMutation.mutateAsync,
      publishBranch: publishBranchMutation.mutateAsync,
      isCreating: createBranchMutation.isPending,
      isSwitching: switchBranchMutation.isPending,
      isPublishing: publishBranchMutation.isPending
    }),
    [
      branchesQuery.data,
      branchesQuery.isLoading,
      createBranchMutation.isPending,
      createBranchMutation.mutateAsync,
      switchBranchMutation.isPending,
      switchBranchMutation.mutateAsync,
      publishBranchMutation.isPending,
      publishBranchMutation.mutateAsync
    ]
  );

  const repoActions = useMemo(
    () => ({
      repos: reposQuery.data,
      isLoading: reposQuery.isLoading,
      openRepo: openRepoMutation.mutateAsync,
      cloneRepo: cloneRepoMutation.mutateAsync,
      setActiveRepo: setActiveRepoMutation.mutateAsync,
      removeRepo: removeRepoMutation.mutateAsync,
      isOpening: openRepoMutation.isPending,
      isCloning: cloneRepoMutation.isPending,
      isSettingActive: setActiveRepoMutation.isPending,
      isRemoving: removeRepoMutation.isPending
    }),
    [
      reposQuery.data,
      reposQuery.isLoading,
      openRepoMutation.isPending,
      openRepoMutation.mutateAsync,
      cloneRepoMutation.isPending,
      cloneRepoMutation.mutateAsync,
      setActiveRepoMutation.isPending,
      setActiveRepoMutation.mutateAsync,
      removeRepoMutation.isPending,
      removeRepoMutation.mutateAsync
    ]
  );

  useEffect(() => {
    if (repoId || !reposQuery.data?.active_id) return;
    setRepoId(String(reposQuery.data.active_id));
  }, [repoId, reposQuery.data?.active_id, setRepoId]);

  useEffect(() => {
    setSelectedFile(undefined);
    setSelectedFiles([]);
    setSelectedIsStaged(false);
    setSelectedIsUntracked(false);
    setViewingCommit(null);
    setIsViewingAnyFile(false);
    setShowRelatedFiles(false);
  }, [repoId]);

  const handleRepoChange = useCallback(
    (nextRepoId: string | null) => {
      setRepoId(nextRepoId);
    },
    [setRepoId]
  );

  const orderedFiles = useMemo<Array<{ path: string; staged: boolean }>>(() => {
    const files = statusQuery.data?.files;
    if (!files) return [];

    return [
      ...(files.conflicts ?? []).map((path) => ({ path, staged: false })),
      ...(files.staged ?? []).map((path) => ({ path, staged: true })),
      ...(files.unstaged ?? []).map((path) => ({ path, staged: false })),
      ...(files.untracked ?? []).map((path) => ({ path, staged: false }))
    ];
  }, [statusQuery.data?.files]);
  const untrackedSet = useMemo(
    () => new Set(statusQuery.data?.files?.untracked ?? []),
    [statusQuery.data?.files?.untracked]
  );

  const workingSetPaths = useMemo<string[]>(() => {
    const files = statusQuery.data?.files;
    if (!files) return [];
    return [
      ...(files.staged ?? []),
      ...(files.unstaged ?? []),
      ...(files.untracked ?? []),
      ...(files.conflicts ?? [])
    ];
  }, [statusQuery.data?.files]);

  const approvedPendingPaths = useMemo(() => {
    const files = approvedChangesQuery.data?.files ?? [];
    return files
      .filter((file) => file.status === "pending" && file.relativePath)
      .map((file) => file.relativePath);
  }, [approvedChangesQuery.data?.files]);

  const approvedPendingSet = useMemo(
    () => new Set(approvedPendingPaths),
    [approvedPendingPaths]
  );

  const createGroupingRule = useCallback(
    (label: string, prefix: string, mode: GroupingRule["mode"] = "prefix"): GroupingRule => {
      return {
        id: `group-${prefix}`,
        label,
        prefixes: [prefix],
        mode: mode ?? "prefix"
      };
    },
    []
  );
  const normalizeGroupingRules = useCallback(
    (rawRules: GroupingRuleLike[]): GroupingRule[] => {
      return rawRules
        .map((rule, index) => {
          const rawPrefixes = Array.isArray(rule?.prefixes)
            ? rule.prefixes
            : typeof rule?.prefix === "string"
              ? [rule.prefix]
              : [];
          const prefixes = rawPrefixes.map((prefix) => prefix.trim()).filter((prefix) => prefix);
          if (prefixes.length === 0) return null;
          const label =
            typeof rule?.label === "string" && rule.label.trim()
              ? rule.label.trim()
              : prefixes[0] ?? `Group ${index + 1}`;
          const mode: GroupingRule["mode"] = rule?.mode === "segment" ? "segment" : "prefix";
          const id =
            typeof rule?.id === "string" && rule.id.trim()
              ? rule.id.trim()
              : `group-${prefixes[0] ?? index}`;
          return { id, label, prefixes, mode };
        })
        .filter(isPresent);
    },
    []
  );

  // Check if grouping rules are available (have valid prefixes)
  const groupingAvailable = useMemo(() => {
    return groupingRules.some((rule) => {
      const rawPrefixes = Array.isArray(rule?.prefixes)
        ? rule.prefixes
        : typeof rule?.prefix === "string"
          ? [rule.prefix]
          : [];
      return rawPrefixes.some((prefix) => prefix.trim());
    });
  }, [groupingRules]);

  const handleCycleViewMode = useCallback(() => {
    setFileViewMode((prev) => {
      if (prev === "flat") return groupingAvailable ? "grouped" : "tree";
      if (prev === "grouped") return "tree";
      return "flat";
    });
  }, [groupingAvailable]);

  const approvedStagedPaths = useMemo(() => {
    const staged = statusQuery.data?.files?.staged ?? [];
    return staged.filter((path) => approvedPendingSet.has(path));
  }, [approvedPendingSet, statusQuery.data?.files?.staged]);

  const canUseApprovedMessage =
    approvedStagedPaths.length > 0 &&
    approvedStagedPaths.length === (statusQuery.data?.files?.staged ?? []).length;

  const orderedKeys = useMemo(() => orderedFiles.map((entry) => selectionKey(entry)), [orderedFiles, selectionKey]);
  const orderedKeyToEntry = useMemo(
    () => new Map(orderedFiles.map((entry) => [selectionKey(entry), entry])),
    [orderedFiles, selectionKey]
  );
  const orderedIndexMap = useMemo(
    () => new Map(orderedKeys.map((key, index) => [key, index])),
    [orderedKeys]
  );
  const orderedKeySet = useMemo(() => new Set(orderedKeys), [orderedKeys]);
  const selectedKeySet = useMemo(
    () => new Set(selectedFiles.map((entry) => selectionKey(entry))),
    [selectedFiles, selectionKey]
  );

  /** Apply a computed selection and update primary-file state */
  const applySelection = useCallback(
    (nextSelection: SelectionEntry[], nextKey: string) => {
      setSelectedFiles(nextSelection);
      lastSelectedKeyRef.current = nextKey;

      if (nextSelection.length === 0) {
        setSelectedFile(undefined);
        setSelectedIsStaged(false);
        setSelectedIsUntracked(false);
        return;
      }

      const clickedEntry = orderedKeyToEntry.get(nextKey) ?? { path: nextKey.slice(2), staged: nextKey.startsWith("1:") };
      const clickedStillSelected = nextSelection.some((entry) => selectionKey(entry) === nextKey);
      const primary = clickedStillSelected
        ? clickedEntry
        : nextSelection[nextSelection.length - 1] ?? clickedEntry;
      setSelectedFile(primary.path);
      setSelectedIsStaged(primary.staged);
      setSelectedIsUntracked(!primary.staged && untrackedSet.has(primary.path));
      setIsViewingAnyFile(false);
      if (showRelatedFiles) {
        setRelatedFilesForPath(primary.path);
      } else {
        setRelatedFilesForPath(undefined);
      }
    },
    [orderedKeyToEntry, selectionKey, untrackedSet, showRelatedFiles],
  );

  const handleSelectFile = useCallback(
    (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => {
      const nextKey = selectionKey({ path, staged });
      const lastKey = lastSelectedKeyRef.current;
      const isToggle = event.metaKey || event.ctrlKey;
      const isRange = event.shiftKey && lastKey && orderedIndexMap.has(lastKey);
      const mode: "single" | "toggle" | "range" = isRange ? "range" : isToggle ? "toggle" : "single";

      const nextSelection = computeNextSelection(
        nextKey, lastKey, mode, selectedFiles,
        orderedIndexMap, orderedKeys, orderedKeyToEntry, selectionKey,
      );
      applySelection(nextSelection, nextKey);
    },
    [
      orderedIndexMap,
      orderedKeyToEntry,
      orderedKeys,
      selectionKey,
      selectedFiles,
      applySelection,
    ]
  );

  // --- Mobile multi-select ---
  const [mobileSelectionMode, setMobileSelectionMode] = useState(false);

  const handleMobileSelectFile = useCallback(
    (path: string, staged: boolean, mode: "toggle" | "range") => {
      const nextKey = selectionKey({ path, staged });
      const lastKey = lastSelectedKeyRef.current;
      const nextSelection = computeNextSelection(
        nextKey, lastKey, mode, selectedFiles,
        orderedIndexMap, orderedKeys, orderedKeyToEntry, selectionKey,
      );
      applySelection(nextSelection, nextKey);
    },
    [orderedIndexMap, orderedKeyToEntry, orderedKeys, selectionKey, selectedFiles, applySelection],
  );

  const handleEnterSelectionMode = useCallback(
    (path: string, staged: boolean) => {
      setMobileSelectionMode(true);
      const nextKey = selectionKey({ path, staged });
      const entry = orderedKeyToEntry.get(nextKey) ?? { path, staged };
      setSelectedFiles([entry]);
      lastSelectedKeyRef.current = nextKey;
      setSelectedFile(path);
      setSelectedIsStaged(staged);
      setSelectedIsUntracked(!staged && untrackedSet.has(path));
    },
    [selectionKey, orderedKeyToEntry, untrackedSet],
  );

  const handleExitSelectionMode = useCallback(() => {
    setMobileSelectionMode(false);
    setSelectedFiles([]);
    setSelectedFile(undefined);
    setSelectedIsStaged(false);
    setSelectedIsUntracked(false);
    lastSelectedKeyRef.current = null;
  }, []);

  // Auto-exit mobile selection mode when entering history or tree view
  useEffect(() => {
    if (mobileSelectionMode && (viewingCommit || fileViewMode === "tree")) {
      setMobileSelectionMode(false);
    }
  }, [mobileSelectionMode, viewingCommit, fileViewMode]);

  const handleStageFile = useCallback(
    (path: string) => {
      const selectedUnstaged = selectedFiles.filter((entry) => !entry.staged).map((entry) => entry.path);
      const shouldStageSelection =
        selectedUnstaged.length > 1 &&
        selectedUnstaged.some((selectedPath) => selectedPath === path);
      const pathsToStage = shouldStageSelection ? selectedUnstaged : [path];

      stageMutation.mutate(
        { paths: pathsToStage },
        {
          onSuccess: (data) => {
            // If we were viewing this file's unstaged diff, switch to staged
            if (selectedFile === path && !selectedIsStaged) {
              setSelectedIsStaged(true);
              setSelectedIsUntracked(false);
            }
            // On mobile, return to Changes tab so user isn't stranded on empty diff
            if (isMobile) {
              setMobileActivePanel("changes");
            }
            pathsToStage.forEach((stagedPath) => {
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(stagedPath, false, false, undefined, "diff", false, repoId)
              });
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(stagedPath, false, true, undefined, "diff", false, repoId)
              });
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(stagedPath, true, false, undefined, "diff", false, repoId)
              });
            });
            // Show warning notice if there were warnings (e.g., ignored files)
            if (data.warnings && data.warnings.length > 0) {
              setWarningNotice({
                message: "Some files were skipped",
                details: data.warnings.join("\n")
              });
            }
          }
        }
      );
    },
    [stageMutation, queryClient, selectedFile, selectedIsStaged, selectedFiles, repoId, isMobile, setMobileActivePanel]
  );

  const handleUnstageFile = useCallback(
    (path: string) => {
      const selectedStaged = selectedFiles.filter((entry) => entry.staged).map((entry) => entry.path);
      const shouldUnstageSelection =
        selectedStaged.length > 1 &&
        selectedStaged.some((selectedPath) => selectedPath === path);
      const pathsToUnstage = shouldUnstageSelection ? selectedStaged : [path];

      unstageMutation.mutate(
        { paths: pathsToUnstage },
        {
          onSuccess: () => {
            // If we were viewing this file's staged diff, switch to unstaged
            if (selectedFile === path && selectedIsStaged) {
              setSelectedIsStaged(false);
              setSelectedIsUntracked(false);
            }
            // On mobile, return to Changes tab so user isn't stranded on empty diff
            if (isMobile) {
              setMobileActivePanel("changes");
            }
            pathsToUnstage.forEach((unstagedPath) => {
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(unstagedPath, false, false, undefined, "diff", false, repoId)
              });
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(unstagedPath, false, true, undefined, "diff", false, repoId)
              });
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(unstagedPath, true, false, undefined, "diff", false, repoId)
              });
            });
          }
        }
      );
    },
    [unstageMutation, queryClient, selectedFile, selectedIsStaged, selectedFiles, repoId, isMobile, setMobileActivePanel]
  );

  const handleStageAll = useCallback(() => {
    const files = statusQuery.data?.files;
    if (!files) return;

    const allUnstaged = [
      ...(files.unstaged ?? []),
      ...(files.untracked ?? []),
      ...(files.conflicts ?? [])
    ];
    if (allUnstaged.length === 0) return;

    stageMutation.mutate(
      { paths: allUnstaged },
      {
        onSuccess: (data) => {
          if (data.warnings && data.warnings.length > 0) {
            setWarningNotice({
              message: "Some files were skipped",
              details: data.warnings.join("\n")
            });
          }
        }
      }
    );
  }, [stageMutation, statusQuery.data]);

  const handleStagePaths = useCallback(
    (paths: string[]) => {
      if (paths.length === 0) return;
      stageMutation.mutate(
        { paths },
        {
          onSuccess: (data) => {
            if (data.warnings && data.warnings.length > 0) {
              setWarningNotice({
                message: "Some files were skipped",
                details: data.warnings.join("\n")
              });
            }
          }
        }
      );
    },
    [stageMutation]
  );

  const handleStageApproved = useCallback(() => {
    const suggestedMessage = approvedChangesQuery.data?.suggestedMessage ?? "";
    if (approvedPendingPaths.length === 0) return;

    stageMutation.mutate(
      { paths: approvedPendingPaths },
      {
        onSuccess: (data) => {
          if (suggestedMessage) {
            setCommitMessage(suggestedMessage);
          }
          if (data.warnings && data.warnings.length > 0) {
            setWarningNotice({
              message: "Some files were skipped",
              details: data.warnings.join("\n")
            });
          }
        }
      }
    );
  }, [approvedChangesQuery.data?.suggestedMessage, approvedPendingPaths, stageMutation]);

  const handleUnstageAll = useCallback(() => {
    const files = statusQuery.data?.files;
    if (!files || (files.staged?.length ?? 0) === 0) return;

    unstageMutation.mutate({ paths: files.staged ?? [] });
  }, [unstageMutation, statusQuery.data]);

  const handleDiscardPaths = useCallback(
    (paths: string[], untracked: boolean) => {
      if (paths.length === 0) return;
      discardMutation.mutate(
        { paths, untracked },
        {
          onSuccess: () => {
            if (selectedFile && paths.includes(selectedFile)) {
              setSelectedFile(undefined);
            }
            setSelectedFiles((prev) =>
              prev.filter((entry) => entry.staged || !paths.includes(entry.path))
            );
            queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
          }
        }
      );
    },
    [discardMutation, queryClient, selectedFile, repoId]
  );

  const handleDiscardFile = useCallback(
    (path: string, untracked: boolean) => {
      discardMutation.mutate(
        { paths: [path], untracked },
        {
          onSuccess: () => {
            // If we were viewing this file's diff, clear selection
            if (selectedFile === path) {
              setSelectedFile(undefined);
            }
            setSelectedFiles((prev) =>
              prev.filter((entry) => !(entry.path === path && !entry.staged))
            );
            queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
          }
        }
      );
    },
    [discardMutation, queryClient, selectedFile, repoId]
  );

  const handleDiscardMultiple = useCallback(() => {
    if (!pendingDiscardFiles || pendingDiscardFiles.length === 0) return;

    const trackedPaths = pendingDiscardFiles.filter((f) => !f.untracked).map((f) => f.path);
    const untrackedPaths = pendingDiscardFiles.filter((f) => f.untracked).map((f) => f.path);
    const allPaths = pendingDiscardFiles.map((f) => f.path);

    const cleanup = () => {
      // Clear selection for discarded files
      if (selectedFile && allPaths.includes(selectedFile)) {
        setSelectedFile(undefined);
      }
      setSelectedFiles((prev) => prev.filter((entry) => entry.staged || !allPaths.includes(entry.path)));
      setPendingDiscardFiles(null);
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    };

    // Handle tracked and untracked separately since API requires specific untracked flag
    if (trackedPaths.length > 0 && untrackedPaths.length > 0) {
      // Both types - chain the mutations
      discardMutation.mutate(
        { paths: trackedPaths, untracked: false },
        {
          onSuccess: () => {
            discardMutation.mutate(
              { paths: untrackedPaths, untracked: true },
              { onSuccess: cleanup }
            );
          }
        }
      );
    } else if (trackedPaths.length > 0) {
      discardMutation.mutate({ paths: trackedPaths, untracked: false }, { onSuccess: cleanup });
    } else if (untrackedPaths.length > 0) {
      discardMutation.mutate({ paths: untrackedPaths, untracked: true }, { onSuccess: cleanup });
    }
  }, [pendingDiscardFiles, discardMutation, queryClient, selectedFile, repoId]);

  const handleIgnoreFile = useCallback(
    (path: string, level?: "project" | "group", groupDir?: string) => {
      ignoreMutation.mutate(
        { path, level, group_dir: groupDir },
        {
          onSuccess: () => {
            if (selectedFile === path) {
              setSelectedFile(undefined);
            }
            setSelectedFiles((prev) => prev.filter((entry) => entry.path !== path));
            queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
          }
        }
      );
    },
    [ignoreMutation, queryClient, selectedFile, repoId]
  );

  const handleConfirmDiscard = useCallback(
    (path: string | null) => {
      if (!path) {
        setConfirmingDiscard(null);
        return;
      }

      // Check if this file is part of a multi-selection
      const selectedDiscardable = selectedFiles
        .filter((entry) => !entry.staged)
        .map((entry) => entry.path);
      const isInMultiSelection =
        selectedDiscardable.length > 1 && selectedDiscardable.includes(path);

      if (isInMultiSelection) {
        // Show modal with all selected files
        const filesToDiscard: DiscardFile[] = selectedDiscardable.map((p) => ({
          path: p,
          untracked: untrackedSet.has(p)
        }));
        setPendingDiscardFiles(filesToDiscard);
        setConfirmingIgnore(null);
      } else {
        // Single file - use inline confirmation
        setConfirmingDiscard(path);
        setConfirmingIgnore(null);
      }
    },
    [selectedFiles, untrackedSet]
  );

  const handleConfirmIgnore = useCallback((path: string | null) => {
    setConfirmingIgnore(path);
    if (path) {
      setConfirmingDiscard(null);
    }
  }, []);

  const handleCommit = useCallback(
    async (
      message: string,
      options: { conventional: boolean; amend: boolean; skipHooks?: boolean; authorName?: string; authorEmail?: string }
    ) => {
      commitAnywayRef.current = false;
      setCommitError(undefined);
      setLastCommitHash(undefined);
      const request: CommitRequest = {
        message,
        validate_conventional: options.conventional,
        amend: options.amend,
        author_name: options.authorName,
        author_email: options.authorEmail
      };

      const precommitCfg = precommitConfigQuery.data;
      const skipHooks = Boolean(options.skipHooks);
      // Skip-hooks bypasses the streamed pre-commit entirely and commits with --no-verify.
      const shouldStream = Boolean(precommitCfg?.enabled && precommitCfg.run_before_commit) && !skipHooks;
      if (shouldStream) {
        // Remember the request so a mid-stream "Commit Anyway" (or a post-pass
        // lock-failure retry) can reuse it without re-running pre-commit.
        setPendingPrecommitCommit(request);
        try {
          const finished = await precommitStream.run({});
          if (finished.type === "error") {
            // An intentional abort (Commit Anyway) is handled by that path; don't surface it.
            if (commitAnywayRef.current) return;
            setCommitError(finished.error || "precommit stream failed");
            return;
          }
          const result = finished.result;
          if (!result || result.status !== "passed") {
            if (result) {
              setPrecommitFailure(result);
              setPendingPrecommitCommit(request);
            }
            setCommitError(undefined);
            return;
          }
        } catch (err) {
          if (commitAnywayRef.current) {
            commitAnywayRef.current = false;
            return;
          }
          setCommitError(err instanceof Error ? err.message : String(err));
          return;
        }
      }

      commitMutation.mutate(
        { ...request, skip_precommit_once: shouldStream || skipHooks || undefined },
        {
          onSuccess: (result) => {
            if (result.success && result.hash) {
              setLastCommitHash(result.hash);
              setCommitMessage("");
              setPrecommitFailure(null);
              setPendingPrecommitCommit(null);
              // Clear selection if viewing staged diff
              if (selectedIsStaged) {
                setSelectedFile(undefined);
              }
            } else if (result.precommit) {
              setPrecommitFailure(result.precommit);
              setPendingPrecommitCommit(request);
              setCommitError(undefined);
            } else {
              // Post-pass failure (e.g. index lock): keep the passed pre-commit reusable
              // so the retry affordance commits without re-streaming.
              if (shouldStream) setPendingPrecommitCommit(request);
              setCommitError(
                result.error ||
                  result.validation_errors?.join("; ") ||
                  "Commit failed"
              );
            }
          },
          onError: (error) => {
            if (shouldStream) setPendingPrecommitCommit(request);
            setCommitError(error.message);
          }
        }
      );
    },
    [commitMutation, precommitConfigQuery.data, precommitStream, selectedIsStaged]
  );

  const handleRunPrecommitAgain = useCallback(async () => {
    setPrecommitFailure(null);
    try {
      const finished = await precommitStream.run({});
      if (finished.type === "error") {
        setCommitError(finished.error || "precommit stream failed");
        return;
      }
      const result = finished.result;
      if (result && result.status !== "passed") {
        setPrecommitFailure(result);
      }
    } catch (err) {
      setCommitError(err instanceof Error ? err.message : String(err));
    }
  }, [precommitStream]);

  const handleCommitSkipPrecommit = useCallback(() => {
    const request = pendingPrecommitCommit ?? { message: commitMessage.trim() };
    setPrecommitFailure(null);
    commitMutation.mutate(
      {
        ...request,
        skip_precommit_once: true,
      },
      {
        onSuccess: (result) => {
          if (result.success && result.hash) {
            setLastCommitHash(result.hash);
            setCommitMessage("");
            setPendingPrecommitCommit(null);
            if (selectedIsStaged) {
              setSelectedFile(undefined);
            }
          } else {
            setCommitError(result.error || result.validation_errors?.join("; ") || "Commit failed");
          }
        },
        onError: (error) => setCommitError(error.message),
      },
    );
  }, [commitMessage, commitMutation, pendingPrecommitCommit, selectedIsStaged]);

  // Commit-anyway trigger reachable both while pre-commit is streaming and from the
  // failure box: abort any in-flight stream first, then commit with skip_precommit_once.
  const handleCommitAnyway = useCallback(() => {
    commitAnywayRef.current = true;
    precommitStream.cancel();
    handleCommitSkipPrecommit();
  }, [precommitStream, handleCommitSkipPrecommit]);

  const handleDisablePrecommit = useCallback(() => {
    const config = precommitConfigQuery.data;
    if (!config) return;
    savePrecommitConfigMutation.mutate(
      { ...config, enabled: false },
      {
        onSuccess: () => {
          setPrecommitFailure(null);
        },
      },
    );
  }, [precommitConfigQuery.data, savePrecommitConfigMutation]);

  const dismissPrecommitFailure = useCallback(() => {
    setPrecommitFailure(null);
    setPendingPrecommitCommit(null);
  }, []);

  const precommitProgressProps = useMemo(() => ({
    running: precommitStream.state.running,
    command: precommitStream.state.command,
    elapsedMs: precommitStream.state.elapsedMs,
    tail: precommitStream.state.tail,
    onCancel: precommitStream.cancel,
    failedResult: precommitFailure,
    onDismissFailure: dismissPrecommitFailure,
    onCommitAnyway: handleCommitAnyway,
    onRunAgain: handleRunPrecommitAgain,
    onDisable: handleDisablePrecommit,
    isCommittingAnyway: commitMutation.isPending,
    isRunningAgain: runPrecommitMutation.isPending || precommitStream.state.running,
    isDisablingChecks: savePrecommitConfigMutation.isPending,
  }), [
    precommitStream.state.running,
    precommitStream.state.command,
    precommitStream.state.elapsedMs,
    precommitStream.state.tail,
    precommitStream.cancel,
    precommitFailure,
    dismissPrecommitFailure,
    handleCommitAnyway,
    handleRunPrecommitAgain,
    handleDisablePrecommit,
    commitMutation.isPending,
    runPrecommitMutation.isPending,
    savePrecommitConfigMutation.isPending,
  ]);

  const handleUseApprovedMessage = useCallback(() => {
    if (!canUseApprovedMessage) return;

    approvedPreviewMutation.mutate(
      { paths: approvedStagedPaths },
      {
        onSuccess: (result) => {
          if (result.available && result.suggestedMessage) {
            setCommitMessage(result.suggestedMessage);
          }
        }
      }
    );
  }, [approvedPreviewMutation, approvedStagedPaths, canUseApprovedMessage]);

  const handlePush = useCallback(() => {
    setPushNotice(null);
    fetchSyncStatus(true, repoId ?? undefined)
      .then((freshStatus) => {
        queryClient.setQueryData(queryKeys.syncStatus(repoId), freshStatus);
        if (freshStatus.fetch_error) {
          setPushNotice({
            tone: "warning",
            message: `Push preflight could not refresh remote: ${freshStatus.fetch_error}`
          });
          return;
        }
        if (freshStatus.behind > 0) {
          setPushNotice({
            tone: "warning",
            message: `Remote has ${freshStatus.behind} new commit${
              freshStatus.behind !== 1 ? "s" : ""
            }. Pull before pushing.`
          });
          return;
        }
        pushMutation.mutate(
          {},
          {
            onSuccess: (result) => {
              const localBranch = statusQuery.data?.branch.head;
              const targetRef = `${result.remote}/${result.branch}`;
              const sourceSuffix =
                localBranch && localBranch !== result.branch ? ` (from ${localBranch})` : "";
              if (result.verification_error) {
                setPushNotice({
                  tone: "warning",
                  message: `Push to ${targetRef}${sourceSuffix} reported success, but verification failed: ${result.verification_error}`
                });
                return;
              }
              if (result.up_to_date) {
                setPushNotice({
                  tone: "info",
                  message: `Already up to date with ${targetRef}${sourceSuffix}`
                });
                return;
              }
              if (result.pushed) {
                setPushNotice({
                  tone: "success",
                  message: `Pushed to ${targetRef}${sourceSuffix}`
                });
                return;
              }
              setPushNotice({
                tone: "warning",
                message: `Push to ${targetRef}${sourceSuffix} completed but could not be verified.`
              });
            },
            onError: () => {
              setPushNotice(null);
            }
          }
        );
      })
      .catch(() => {
        pushMutation.mutate({});
      });
  }, [pushMutation, queryClient, statusQuery.data?.branch.head, repoId]);

  const handlePull = useCallback(() => {
    pullMutation.mutate({});
  }, [pullMutation]);

  const handleSaveFileContent = useCallback(
    async (path: string, content: string, expectedHash?: string) => {
      return saveFileContentMutation.mutateAsync({
        path,
        content,
        expected_hash: expectedHash
      });
    },
    [saveFileContentMutation]
  );

  const handleLoadMoreHistory = useCallback(() => {
    setHistoryLimit((prev) => Math.min(historyMaxLimit, prev + 50));
  }, []);

  // Handle clicking a commit in the history panel to enter history mode
  const handleSelectCommit = useCallback(
    (entry: RepoHistoryEntry | null) => {
      if (!entry) {
        // Exit history mode
        setViewingCommit(null);
        setSelectedFile(undefined);
        setSelectedFiles([]);
        return;
      }

      // Enter history mode
      setViewingCommit({
        hash: entry.hash,
        subject: entry.subject,
        files: entry.files,
        author: entry.author,
        date: entry.date,
        checks: entry.checks
      });

      // Clear current file selection - user will select from commit files
      setSelectedFile(undefined);
      setSelectedFiles([]);
      setSelectedIsStaged(false);
      setSelectedIsUntracked(false);
    },
    []
  );

  // Handle "continue" action: pre-fill commit message with incremented pN
  const handleContinueCommit = useCallback((message: string) => {
    setCommitMessage(message);
    setViewingCommit(null);
    setSelectedFile(undefined);
    setSelectedFiles([]);
    if (isMobile) setMobileActivePanel("commit");
  }, [isMobile]);

  // Computed group filter info for the active grep prefix
  const activeGroupFilter = useMemo(() => {
    if (!historyGrepPrefix) return null;
    const count = historyQuery.data?.entries?.length ?? historyQuery.data?.lines?.length ?? 0;
    return { prefix: historyGrepPrefix, count };
  }, [historyGrepPrefix, historyQuery.data]);

  // Handle selecting a file when in history mode
  const handleSelectHistoryFile = useCallback(
    (path: string) => {
      setSelectedFile(path);
      setSelectedIsStaged(false);
      setSelectedIsUntracked(false);
      setShowRelatedFiles(false);
      setRelatedFilesForPath(undefined);
    },
    []
  );

  // Exit history mode and return to working directory
  const handleExitHistoryMode = useCallback(() => {
    setViewingCommit(null);
    setSelectedFile(undefined);
    setSelectedFiles([]);
    setSelectedIsStaged(false);
    setSelectedIsUntracked(false);
    setIsViewingAnyFile(false);
    setShowRelatedFiles(false);
    setRelatedFilesForPath(undefined);
  }, []);

  // Handle selecting a file from file search (view any file in source mode)
  // Optional lineNumber for content search results - can be used to scroll to that line
  const handleSelectAnyFile = useCallback((path: string, _lineNumber?: number) => {
    setSelectedFile(path);
    setSelectedIsStaged(false);
    setSelectedIsUntracked(false);
    setSelectedFiles([]);
    setIsViewingAnyFile(true);
    setViewingCommit(null);
    setViewMode("source"); // Force source mode for any file
    // If related files panel is open, update it to show relations for the new file
    if (showRelatedFiles) {
      setRelatedFilesForPath(path);
    } else {
      setRelatedFilesForPath(undefined);
    }
    if (isMobile) {
      setMobileActivePanel("diff");
    } else {
      setPrimaryPanel("diff");
    }
    // Scrolling to a specific lineNumber would require adding state and passing it to DiffViewer
  }, [isMobile, showRelatedFiles]);

  // Handle showing related files panel
  const handleShowRelatedFiles = useCallback((path: string) => {
    setRelatedFilesForPath(path);
    setShowRelatedFiles(true);
  }, []);

  // Handle back from related files panel
  const handleBackFromRelatedFiles = useCallback(() => {
    // Set scrollToFile before hiding related files so we scroll to the current selection
    if (selectedFile) {
      setScrollToFile(selectedFile);
    }
    setShowRelatedFiles(false);
    setRelatedFilesForPath(undefined);
  }, [selectedFile]);

  // Clear scrollToFile after scroll completes
  const handleScrollComplete = useCallback(() => {
    setScrollToFile(undefined);
  }, []);

  // Handle selecting a file from the related files panel
  const handleSelectRelatedFile = useCallback((path: string) => {
    setSelectedFile(path);
    setSelectedIsStaged(false);
    setSelectedIsUntracked(false);
    setSelectedFiles([]);
    setIsViewingAnyFile(true);
    setViewingCommit(null);
    setViewMode("source");
    // Keep related files panel open so user can explore related files of the new selection
    setRelatedFilesForPath(path);
    if (isMobile) {
      setMobileActivePanel("diff");
    }
  }, [isMobile]);

  // Handle request to delete file/folder (shows confirmation modal)
  const handleRequestDeletePath = useCallback((path: string, isDir: boolean) => {
    setPendingDeletePath({ path, isDir });
  }, []);

  // Handle confirmed delete
  const handleConfirmDelete = useCallback(() => {
    if (!pendingDeletePath) return;
    deletePathMutation.mutate(
      { path: pendingDeletePath.path },
      {
        onSuccess: () => {
          setPendingDeletePath(null);
          // Clear selection if we deleted the selected file
          if (selectedFile === pendingDeletePath.path) {
            setSelectedFile(undefined);
          }
        }
      }
    );
  }, [pendingDeletePath, deletePathMutation, selectedFile]);

  // Handle cancel delete
  const handleCancelDelete = useCallback(() => {
    setPendingDeletePath(null);
  }, []);

  // Handle blame file (view file history)
  const handleBlameFile = useCallback((path: string) => {
    const filename = path.split("/").pop() || path;
    setViewingFileBlame({ path, filename });
    if (isMobile) {
      setMobileActivePanel("history");
    }
  }, [isMobile]);

  // Handle exit blame mode
  const handleExitBlameMode = useCallback(() => {
    setViewingFileBlame(null);
  }, []);

  // Keyboard shortcut for file search (Cmd+K / Ctrl+K)
  useGlobalKeydown((event) => {
    if ((event.metaKey || event.ctrlKey) && event.key === "k") {
      event.preventDefault();
      if (isFileSearchOpen) {
        emitShortcutIntent({
          action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
          outcome: "noop",
          chord: "mod+k",
          source: "keyboard",
        });
        return;
      }
        setIsFileSearchOpen(true);
      }
  });

  // View mode fallback: when selectedFile changes, ensure viewMode is valid for the new file
  useEffect(() => {
    if (!selectedFile) return;

    const fileType = getFileTypeInfo(selectedFile);
    const availableModes: ViewMode[] = ["diff", "full_diff", "source"];
    if (fileType.canPreview) {
      availableModes.push("preview");
    }

    // Check if this file has any git changes
    const hasGitChanges = orderedFiles.some((entry) => entry.path === selectedFile);

    // If current mode is not available for this file, fallback
    if (!availableModes.includes(viewMode)) {
      // For files from search (isViewingAnyFile) or without changes, prefer "source"
      // For git changes, prefer "diff"
      const fallbackMode = isViewingAnyFile || !hasGitChanges ? "source" : "diff";
      setViewMode(fallbackMode);
    }
    // If viewing a file without changes in diff mode, switch to source
    else if (!hasGitChanges && (viewMode === "diff" || viewMode === "full_diff")) {
      setViewMode("source");
    }
  }, [selectedFile, viewMode, isViewingAnyFile, orderedFiles]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.sidebarWidth", String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.changesHeight", String(changesHeight));
  }, [changesHeight]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.historyHeight", String(historyHeight));
  }, [historyHeight]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.mobileActivePanel", mobileActivePanel);
  }, [mobileActivePanel]);

  useEffect(() => {
    if (commitMessage) {
      localStorage.setItem("gct.commitMessage", commitMessage);
    } else {
      localStorage.removeItem("gct.commitMessage");
    }
  }, [commitMessage]);

  useEffect(() => {
    if (!repoDir) return;
    if (groupingLoadedKey === repoKey) return;

    // View mode still from localStorage (it's a UI-only preference)
    const viewModeKey = `gct.viewMode.${repoKey}`;
    const storedViewMode = localStorage.getItem(viewModeKey);
    const legacyEnabledKey = `gct.grouping.${repoKey}.enabled`;
    const legacyEnabled = localStorage.getItem(legacyEnabledKey);
    if (storedViewMode === "flat" || storedViewMode === "grouped" || storedViewMode === "tree") {
      setFileViewMode(storedViewMode);
    } else if (legacyEnabled === "true") {
      setFileViewMode("grouped");
    } else {
      setFileViewMode("flat");
    }

    // Load grouping rules from API
    if (groupingRulesQuery.data) {
      const apiRules = groupingRulesQuery.data.rules ?? [];
      // Convert API format to UI format
      const uiRules: GroupingRule[] = apiRules.map(r => ({
        id: r.id,
        label: r.label,
        prefixes: r.prefixes,
        mode: r.mode as "prefix" | "segment",
      }));
      setGroupingRules(normalizeGroupingRules(uiRules));
      setGroupingDefaultsPending(apiRules.length === 0);
      setGroupingLoadedKey(repoKey);
    } else if (!groupingRulesQuery.isLoading) {
      // API returned no data and isn't loading - check localStorage for migration
      const rulesKey = `gct.grouping.${repoKey}.rules`;
      const storedRules = localStorage.getItem(rulesKey);
      if (storedRules) {
        try {
          const parsed: unknown = JSON.parse(storedRules);
          const normalized = Array.isArray(parsed)
            ? normalizeGroupingRules(parsed.filter(isGroupingRuleLike))
            : [];
          setGroupingRules(normalized);
          setGroupingDefaultsPending(false);
          // Migrate to API
          if (normalized.length > 0) {
            const apiConfig: GroupingRulesConfig = {
              enabled: true,
              rules: normalized.map(r => ({
                id: r.id,
                label: r.label,
                prefixes: r.prefixes ?? (r.prefix ? [r.prefix] : []),
                mode: r.mode ?? "prefix",
              })),
            };
            saveGroupingRulesMutation.mutate(apiConfig, {
              onSuccess: () => {
                // Clear localStorage after successful migration
                localStorage.removeItem(rulesKey);
              },
            });
          }
        } catch {
          setGroupingRules([]);
          setGroupingDefaultsPending(true);
        }
      } else {
        setGroupingRules([]);
        setGroupingDefaultsPending(true);
      }
      setGroupingLoadedKey(repoKey);
    }
    // When groupingRulesQuery is still loading (data=undefined, isLoading=true),
    // we intentionally do NOT set groupingLoadedKey. This allows the effect to
    // re-run when the data arrives, preventing a race condition where slow API
    // responses (e.g. cold proxy cache) would cause grouping rules to be missed.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoDir, repoKey, groupingLoadedKey, normalizeGroupingRules, groupingRulesQuery.data, groupingRulesQuery.isLoading]);

  useEffect(() => {
    if (!repoDir) return;
    if (layoutLoadedKey === repoKey) return;
    const presetKey = `gct.layout.${repoKey}.preset`;
    const primaryKey = `gct.layout.${repoKey}.primary`;
    const stackKey = `gct.layout.${repoKey}.stackHeight`;
    const storedPreset = localStorage.getItem(presetKey);
    const storedPrimary = localStorage.getItem(primaryKey);
    const storedStackHeight = Number(localStorage.getItem(stackKey));
    setLayoutPreset(
      storedPreset && isLayoutPreset(storedPreset)
        ? storedPreset
        : "classic"
    );
    if (!urlSetPrimaryRef.current) {
      setPrimaryPanel(
        storedPrimary === "changes" ||
          storedPrimary === "history" ||
          storedPrimary === "commit" ||
          storedPrimary === "diff" ||
          storedPrimary === "review"
          ? storedPrimary
          : "diff"
      );
    }
    // Clear the URL override flag — localStorage can take over for future repo switches
    urlSetPrimaryRef.current = false;
    if (Number.isFinite(storedStackHeight) && storedStackHeight > 0) {
      setStackHeight(storedStackHeight);
    }
    setLayoutLoadedKey(repoKey);
  }, [repoDir, repoKey, layoutLoadedKey]);

  useEffect(() => {
    if (!repoDir || groupingLoadedKey !== repoKey) return;
    const viewModeKey = `gct.viewMode.${repoKey}`;
    localStorage.setItem(viewModeKey, fileViewMode);
    // Save grouping rules to API (instead of localStorage)
    saveGroupingRulesMutation.mutate({
      enabled: groupingRules.length > 0,
      rules: groupingRules.map(r => ({
        id: r.id,
        label: r.label,
        prefixes: r.prefixes ?? (r.prefix ? [r.prefix] : []),
        mode: r.mode ?? "prefix",
      })),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoDir, repoKey, groupingLoadedKey, fileViewMode, groupingRules]);

  useEffect(() => {
    if (!repoDir || layoutLoadedKey !== repoKey) return;
    const presetKey = `gct.layout.${repoKey}.preset`;
    const primaryKey = `gct.layout.${repoKey}.primary`;
    const stackKey = `gct.layout.${repoKey}.stackHeight`;
    localStorage.setItem(presetKey, layoutPreset);
    localStorage.setItem(primaryKey, primaryPanel);
    localStorage.setItem(stackKey, String(stackHeight));
  }, [repoDir, repoKey, layoutLoadedKey, layoutPreset, primaryPanel, stackHeight]);

  useEffect(() => {
    if (!groupingDefaultsPending || !repoDir) return;
    const files = statusQuery.data?.files;
    if (!files) return;
    const allFiles = [
      ...(files.staged ?? []),
      ...(files.unstaged ?? []),
      ...(files.untracked ?? []),
      ...(files.conflicts ?? [])
    ];
    const hasScenarios = allFiles.some((path) => path.startsWith("scenarios/"));
    const hasResources = allFiles.some((path) => path.startsWith("resources/"));
    if (hasScenarios || hasResources) {
      const defaults: GroupingRule[] = [];
      if (hasScenarios) defaults.push(createGroupingRule("Scenarios", "scenarios/", "segment"));
      if (hasResources) defaults.push(createGroupingRule("Resources", "resources/", "segment"));
      setGroupingRules(defaults);
    }
    setGroupingDefaultsPending(false);
  }, [groupingDefaultsPending, repoDir, statusQuery.data?.files, createGroupingRule]);

  useEffect(() => {
    // If in grouped mode but no rules available, fall back to flat
    if (groupingRules.length === 0 && fileViewMode === "grouped") {
      setFileViewMode("flat");
    }
  }, [groupingRules, fileViewMode]);

  useEffect(() => {
    // Skip until URL initialization is complete (state batched together)
    if (!urlInitComplete) return;
    // Skip working directory cleanup when in history mode or viewing any file
    if (viewingCommit || isViewingAnyFile) return;

    if (!orderedKeySet.size) {
      setSelectedFiles([]);
      setSelectedFile(undefined);
      setSelectedIsStaged(false);
      setSelectedIsUntracked(false);
      lastSelectedKeyRef.current = null;
      return;
    }

    setSelectedFiles((prev) => prev.filter((entry) => orderedKeySet.has(selectionKey(entry))));
  }, [urlInitComplete, orderedKeySet, selectionKey, viewingCommit, isViewingAnyFile]);

  useEffect(() => {
    // Skip until URL initialization is complete (state batched together)
    if (!urlInitComplete) return;
    // Skip working directory cleanup when in history mode or viewing any file
    if (viewingCommit || isViewingAnyFile) return;

    if (!selectedFile) return;
    const activeKey = selectionKey({ path: selectedFile, staged: selectedIsStaged });
    if (orderedKeySet.has(activeKey)) return;

    if (selectedFiles.length > 0) {
      const fallback = selectedFiles[selectedFiles.length - 1];
      if (!fallback) {
        setSelectedFile(undefined);
        setSelectedIsStaged(false);
        setSelectedIsUntracked(false);
        return;
      }
      setSelectedFile(fallback.path);
      setSelectedIsStaged(fallback.staged);
      setSelectedIsUntracked(!fallback.staged && untrackedSet.has(fallback.path));
    } else {
      setSelectedFile(undefined);
      setSelectedIsStaged(false);
      setSelectedIsUntracked(false);
    }
  }, [
    urlInitComplete,
    orderedKeySet,
    selectedFile,
    selectedFiles,
    selectedIsStaged,
    selectionKey,
    untrackedSet,
    viewingCommit,
    isViewingAnyFile
  ]);

  useEffect(() => {
    if (!stackRef.current || typeof ResizeObserver === "undefined") return;

    const minChanges = 200;
    const minHistory = 140;
    const dividerHeight = 6;
    const minBottom = 180;
    const clamp = () => {
      if (!stackRef.current || bottomCollapsed) return;
      const height = stackRef.current.clientHeight;
      const minTop =
        topCollapsed && middleCollapsed
          ? minChanges
          : topCollapsed
            ? minHistory
            : middleCollapsed
              ? minChanges
              : minChanges + minHistory + dividerHeight;
      const maxHeight = Math.max(minTop, height - minBottom);
      if (changesHeight > maxHeight) {
        setChangesHeight(maxHeight);
      } else if (changesHeight < minTop) {
        setChangesHeight(Math.min(minTop, maxHeight));
      }
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(stackRef.current);
    return () => observer.disconnect();
  }, [changesHeight, topCollapsed, middleCollapsed, bottomCollapsed]);

  useEffect(() => {
    if (topCollapsed || middleCollapsed) return;
    const minChanges = 200;
    const minHistory = 140;
    const maxHistory = Math.max(minHistory, changesHeight - minChanges);
    if (historyHeight > maxHistory) {
      setHistoryHeight(maxHistory);
    } else if (historyHeight < minHistory) {
      setHistoryHeight(Math.min(minHistory, maxHistory));
    }
  }, [changesHeight, historyHeight, topCollapsed, middleCollapsed]);

  useEffect(() => {
    if (stackPosition !== "bottom" || !mainRef.current || typeof ResizeObserver === "undefined") {
      return;
    }
    const minStack = 220;
    const minMain = 240;
    const clamp = () => {
      if (!mainRef.current) return;
      const height = mainRef.current.clientHeight;
      const maxStack = height - minMain;
      if (stackHeight > maxStack) {
        setStackHeight(Math.max(minStack, maxStack));
      }
    };
    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(mainRef.current);
    return () => observer.disconnect();
  }, [stackHeight, stackPosition]);

  useEffect(() => {
    if (!isResizingStack) return;

    // Track the latest clamped value during drag so handleUp can commit it
    // to React state once. handleMove writes to the DOM imperatively to
    // avoid one App re-render per mouse event.
    let latestHeight: number | null = null;
    let latestWidth: number | null = null;

    const handleMove = (event: MouseEvent) => {
      if (!stackResize.current) return;
      if (stackResize.current.mode === "bottom") {
        const minStack = 220;
        const minMain = 240;
        const nextHeight =
          stackResize.current.height - (event.clientY - stackResize.current.top);
        const maxHeight = stackResize.current.height - minMain;
        const clampedHeight = Math.max(minStack, Math.min(maxHeight, nextHeight));
        latestHeight = clampedHeight;
        if (stackRef.current) stackRef.current.style.height = `${clampedHeight}px`;
        return;
      }

      const minWidth = sidebarMinWidth;
      const nextWidth =
        stackResize.current.mode === "left"
          ? event.clientX - stackResize.current.start
          : stackResize.current.start - event.clientX;
      const clampedWidth = Math.max(minWidth, Math.min(stackResize.current.max, nextWidth));
      latestWidth = clampedWidth;
      if (stackRef.current) stackRef.current.style.width = `${clampedWidth}px`;
    };

    const handleUp = () => {
      if (latestHeight !== null) setStackHeight(latestHeight);
      if (latestWidth !== null) setSidebarWidth(latestWidth);
      setIsResizingStack(false);
      stackResize.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = stackPosition === "bottom" ? "row-resize" : "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizingStack, stackPosition]);

  useEffect(() => {
    if (!isResizingSplit) return;

    let latestHeight: number | null = null;

    const handleMove = (event: MouseEvent) => {
      if (!splitResize.current) return;
      const minTop = 200;
      const minBottom = 180;
      const nextHeight = event.clientY - splitResize.current.top;
      const maxHeight = splitResize.current.height - minBottom;
      const clampedHeight = Math.max(minTop, Math.min(maxHeight, nextHeight));
      latestHeight = clampedHeight;
      // Imperatively rewrite the inner grid's row template; the React render
      // path computes the same string from sidebarRows on commit (handleUp).
      if (sidebarGridRef.current) {
        sidebarGridRef.current.style.gridTemplateRows = `minmax(0, ${clampedHeight}px) 6px minmax(0, 1fr)`;
      }
    };

    const handleUp = () => {
      if (latestHeight !== null) setChangesHeight(latestHeight);
      setIsResizingSplit(false);
      splitResize.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "row-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizingSplit]);

  useEffect(() => {
    if (!isResizingHistory) return;

    let latestHeight: number | null = null;

    const handleMove = (event: MouseEvent) => {
      if (!historyResize.current) return;
      const minHistory = 140;
      const minChanges = 200;
      const nextHeight = historyResize.current.bottom - event.clientY;
      const maxHeight = Math.max(minHistory, changesHeight - minChanges);
      const clampedHeight = Math.max(minHistory, Math.min(maxHeight, nextHeight));
      latestHeight = clampedHeight;
      if (topStackGridRef.current) {
        topStackGridRef.current.style.gridTemplateRows = `minmax(0, 1fr) 6px minmax(0, ${clampedHeight}px)`;
      }
    };

    const handleUp = () => {
      if (latestHeight !== null) setHistoryHeight(latestHeight);
      setIsResizingHistory(false);
      historyResize.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "row-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizingHistory, changesHeight]);

  const handleStackResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!mainRef.current) return;
    const rect = mainRef.current.getBoundingClientRect();
    if (stackPosition === "bottom") {
      stackResize.current = { mode: "bottom", top: rect.top, height: rect.height };
    } else {
      const minWidth = sidebarMinWidth;
      stackResize.current = {
        mode: stackPosition,
        start: stackPosition === "left" ? rect.left : rect.right,
        max: Math.max(minWidth, rect.width - diffMinWidth)
      };
    }
    setIsResizingStack(true);
  };

  const handleSplitResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!stackRef.current || bottomCollapsed || topStackCollapsed) return;
    const rect = stackRef.current.getBoundingClientRect();
    splitResize.current = { top: rect.top, height: rect.height };
    setIsResizingSplit(true);
  };

  const handleHistoryResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!stackRef.current || topCollapsed || middleCollapsed) return;
    const rect = stackRef.current.getBoundingClientRect();
    historyResize.current = { bottom: rect.top + changesHeight };
    setIsResizingHistory(true);
  };

  const renderPanel = (panel: LayoutSection, slot: "top" | "middle" | "bottom" | "main") => {
    const isMain = slot === "main";
    const isHistoryMode = Boolean(viewingCommit);

    switch (panel) {
      case "changes":
        // Show related files panel when active
        if (showRelatedFiles && relatedFilesForPath) {
          return (
            <RelatedFilesPanel
              forPath={relatedFilesForPath}
              onBack={handleBackFromRelatedFiles}
              onSelectFile={handleSelectRelatedFile}
              repoId={repoId}
            />
          );
        }
        // In history mode, show HistoryFileList instead of FileList
        if (isHistoryMode && viewingCommit) {
          return (
            <HistoryFileList
              viewingCommit={viewingCommit}
              selectedFile={selectedFile}
              onSelectFile={handleSelectHistoryFile}
              collapsed={changesCollapsed}
              onToggleCollapse={() => setChangesCollapsed((prev) => !prev)}
              fillHeight={isMain || !changesCollapsed}
              onDeletePath={handleRequestDeletePath}
            />
          );
        }
        return (
          <FileList
            files={statusQuery.data?.files}
            fileStats={statusQuery.data?.file_stats}
            selectedFiles={selectedFiles}
            selectedKeySet={selectedKeySet}
            selectionKey={selectionKey}
            approvedChanges={
              approvedChangesQuery.data
                ? {
                    available: approvedChangesQuery.data.available,
                    committableFiles: approvedChangesQuery.data.committableFiles,
                    warning: approvedChangesQuery.data.warning
                  }
                : undefined
            }
            approvedPaths={approvedPendingSet}
            onStageApproved={handleStageApproved}
            isStagingApproved={isStaging}
            onSelectFile={(path, staged, event) => {
              handleSelectFile(path, staged, event);
              if (primaryPanel === "review") setPrimaryPanel("diff");
            }}
            onStageFile={handleStageFile}
            onUnstageFile={handleUnstageFile}
            onDiscardFile={handleDiscardFile}
            onIgnoreFile={handleIgnoreFile}
            onStageAll={handleStageAll}
            onUnstageAll={handleUnstageAll}
            isStaging={isStaging}
            pendingPaths={pendingPaths}
            isDiscarding={isDiscarding}
            isIgnoring={isIgnoring}
            confirmingDiscard={confirmingDiscard}
            onConfirmDiscard={handleConfirmDiscard}
            confirmingIgnore={confirmingIgnore}
            onConfirmIgnore={handleConfirmIgnore}
            collapsed={changesCollapsed}
            onToggleCollapse={() => setChangesCollapsed((prev) => !prev)}
            fillHeight={isMain || !changesCollapsed}
            fileViewMode={fileViewMode}
            groupingRules={groupingRules}
            groupingAvailable={groupingAvailable}
            onCycleViewMode={handleCycleViewMode}

            onStagePaths={handleStagePaths}
            onDiscardPaths={handleDiscardPaths}
            scrollToFile={scrollToFile}
            onScrollComplete={handleScrollComplete}
            onDeletePath={handleRequestDeletePath}
            onBlameFile={handleBlameFile}
            repoId={repoId}
            onOpenReview={(slug) => { if (reviewScenarioSlug && slug !== reviewScenarioSlug) { scenarioReview.switchScenario(reviewScenarioSlug, slug); } setReviewScenarioSlug(slug); setPrimaryPanel("review"); }}
            mobileSelectionMode={mobileSelectionMode}
            onEnterSelectionMode={handleEnterSelectionMode}
            onExitSelectionMode={handleExitSelectionMode}
            onMobileSelectFile={handleMobileSelectFile}
            fileHotspots={statusQuery.data?.file_hotspots}
          />
        );
      case "history":
        return (
          <GitHistory
            lines={historyQuery.data?.lines}
            entries={historyQuery.data?.entries}
            isLoading={historyQuery.isLoading}
            error={historyQuery.error}
            collapsed={historyCollapsed}
            onToggleCollapse={() => setHistoryCollapsed((prev) => !prev)}
            height={slot === "middle" ? historyHeight : undefined}
            fillHeight={isMain}
            onLoadMore={handleLoadMoreHistory}
            isFetching={historyQuery.isFetching}
            hasMore={
              !historyGrepPrefix &&
              (historyQuery.data?.lines?.length ?? 0) >= historyLimit &&
              historyLimit < historyMaxLimit
            }
            searchQuery={historySearch}
            onSearchQueryChange={setHistorySearch}
            scopeFilter={historyScopeFilter}
            onScopeFilterChange={setHistoryScopeFilter}
            groupingEnabled={fileViewMode === "grouped"}
            groupingRules={groupingRules}
            workingSetPaths={workingSetPaths}
            workingSetOnly={historyWorkingSetOnly}
            onWorkingSetOnlyChange={setHistoryWorkingSetOnly}
            filtersOpen={isHistoryFiltersOpen}
            onOpenFilters={() => setIsHistoryFiltersOpen(true)}
            onCloseFilters={() => setIsHistoryFiltersOpen(false)}
            selectedCommitHash={viewingCommit?.hash}
            onSelectCommit={(entry) => {
              handleSelectCommit(entry);
              if (entry && primaryPanel === "review") setPrimaryPanel("diff");
            }}
            blameFilePath={viewingFileBlame?.path}
            blameFileName={viewingFileBlame?.filename}
            onExitBlameMode={handleExitBlameMode}
            onContinueCommit={handleContinueCommit}
            activeGroupFilter={activeGroupFilter}
            onFilterGroup={setHistoryGrepPrefix}
            onClearGroupFilter={() => setHistoryGrepPrefix(null)}
          />
        );
      case "commit":
        return (
          <CommitPanel
            stagedCount={statusQuery.data?.summary.staged ?? 0}
            commitMessage={commitMessage}
            onCommitMessageChange={setCommitMessage}
            canUseApprovedMessage={canUseApprovedMessage}
            onUseApprovedMessage={handleUseApprovedMessage}
            isUsingApprovedMessage={approvedPreviewMutation.isPending}
            onCommit={handleCommit}
            isCommitting={commitMutation.isPending || precommitStream.state.running}
            precommitProgress={precommitProgressProps}
            commitError={commitError}
            onRetryWithoutPrecommit={handleCommitSkipPrecommit}
            canRetryWithoutPrecommit={Boolean(commitError && pendingPrecommitCommit)}
            defaultAuthorName={statusQuery.data?.author?.name}
            defaultAuthorEmail={statusQuery.data?.author?.email}
            canAmend={canAmend}
            amendDisabledReason={amendDisabledReason}
            collapsed={commitCollapsed}
            onToggleCollapse={() => setCommitCollapsed((prev) => !prev)}
            fillHeight={isMain || !commitCollapsed}
            onPush={handlePush}
            isPushing={pushMutation.isPending}
            canPush={syncStatusQuery.data?.can_push ?? false}
            aheadCount={syncStatusQuery.data?.ahead ?? 0}
            pushTarget={pushTargetRef}
            sourceBranch={pushSourceBranch}
            isHistoryMode={isHistoryMode}
            historyCommit={viewingCommit}
          />
        );
      case "review":
        return (
          <ScenarioReviewPanel
            scenarioSlug={reviewScenarioSlug}
            repoId={repoId}
            fileStats={statusQuery.data?.file_stats}
            onChangeScenario={(slug) => { if (reviewScenarioSlug && slug !== reviewScenarioSlug) { scenarioReview.switchScenario(reviewScenarioSlug, slug); } setReviewScenarioSlug(slug); }}
            activeTab={scenarioReview.state.activeTab}
            onActiveTabChange={(tab) => scenarioReview.update({ activeTab: tab })}
            agentRunId={scenarioReview.state.agentRunId}
            onAgentRunIdChange={(id) => scenarioReview.update({ agentRunId: id })}
            scenarioState={scenarioReview.state}
            onScenarioStateChange={scenarioReview.update}
          />
        );
      case "diff":
      default:
        return (
          <div className="h-full min-h-0">
            <DiffViewer
              diff={diffQuery.data}
              selectedFile={selectedFile}
              isStaged={selectedIsStaged}
              isUntracked={selectedIsUntracked}
              isLoading={diffQuery.isLoading}
              error={diffQuery.error}
              repoDir={statusQuery.data?.repo_dir}
              viewMode={viewMode}
              onViewModeChange={setViewMode}
              isHistoryMode={isHistoryMode}
              commitHash={viewingCommit?.hash}
              onShowRelatedFiles={handleShowRelatedFiles}
              onOpenSearch={() => setIsFileSearchOpen(true)}
              onOpenReview={() => setPrimaryPanel("review")}
              isReadOnly={isViewingAnyFile}
              onSaveFileContent={handleSaveFileContent}
              isSavingFile={saveFileContentMutation.isPending}
              onDeletePath={handleRequestDeletePath}
              isDeleting={isDeleting}
            />
          </div>
        );
    }
  };

  const showSplitHandle = !bottomCollapsed && !topStackCollapsed;
  const sidebarRows = (() => {
    if (topStackCollapsed && bottomCollapsed) return "auto 0px auto";
    if (topStackCollapsed) return "auto 0px minmax(0, 1fr)";
    if (bottomCollapsed) return "minmax(0, 1fr) 0px auto";
    return `minmax(0, ${changesHeight}px) 6px minmax(0, 1fr)`;
  })();
  const topStackRows =
    !topCollapsed && !middleCollapsed
      ? `minmax(0, 1fr) 6px minmax(0, ${historyHeight}px)`
      : "minmax(0, 1fr)";
  const isBottomLayout = stackPosition === "bottom";
  const stackBorderClass =
    stackPosition === "bottom"
      ? "border-t"
      : stackPosition === "left"
        ? "border-r"
        : "border-l";
  const stackPanel = (
    <div
      className={`flex-shrink-0 overflow-hidden min-w-0 ${stackBorderClass} border-slate-800`}
      style={
        isBottomLayout
          ? { height: stackHeight }
          : { width: sidebarWidth, minWidth: sidebarMinWidth }
      }
      ref={stackRef}
    >
      <div ref={sidebarGridRef} className="h-full min-h-0 min-w-0 grid overflow-hidden" style={{ gridTemplateRows: sidebarRows }}>
        <div className="min-h-0 min-w-0 overflow-hidden">
          <div ref={topStackGridRef} className="h-full min-h-0 min-w-0 grid" style={{ gridTemplateRows: topStackRows }}>
            <div className="min-h-0 min-w-0">{renderPanel(topPanel, "top")}</div>
            <div
              className={`${
                !topCollapsed && !middleCollapsed
                  ? "cursor-row-resize bg-slate-900 hover:bg-slate-800"
                  : "bg-transparent"
              }`}
              onMouseDown={!topCollapsed && !middleCollapsed ? handleHistoryResizeStart : undefined}
              aria-hidden="true"
            />
            <div className="min-h-0 min-w-0">{renderPanel(middlePanel, "middle")}</div>
          </div>
        </div>
        <div
          className={`${
            showSplitHandle ? "cursor-row-resize bg-slate-900 hover:bg-slate-800" : "bg-transparent"
          }`}
          onMouseDown={showSplitHandle ? handleSplitResizeStart : undefined}
          aria-hidden="true"
        />
        <div className="min-h-0 min-w-0 border-t border-slate-800 overflow-hidden">
          {renderPanel(bottomPanel, "bottom")}
        </div>
      </div>
    </div>
  );

  // Helper to render panel content for mobile (simplified, fill height)
  const renderMobilePanel = (panel: LayoutSection) => {
    const isHistoryMode = Boolean(viewingCommit);

    switch (panel) {
      case "changes":
        // Show related files panel when active
        if (showRelatedFiles && relatedFilesForPath) {
          return (
            <RelatedFilesPanel
              forPath={relatedFilesForPath}
              onBack={handleBackFromRelatedFiles}
              onSelectFile={(path) => {
                handleSelectRelatedFile(path);
                // On mobile, switch to diff view after selecting a file
                setMobileActivePanel("diff");
              }}
              repoId={repoId}
            />
          );
        }
        // In history mode, show HistoryFileList instead of FileList
        if (isHistoryMode && viewingCommit) {
          return (
            <HistoryFileList
              viewingCommit={viewingCommit}
              selectedFile={selectedFile}
              onSelectFile={(path) => {
                handleSelectHistoryFile(path);
                // On mobile, switch to diff view after selecting a file
                setMobileActivePanel("diff");
              }}
              collapsed={false}
              fillHeight={true}
              onDeletePath={handleRequestDeletePath}
            />
          );
        }
        return (
          <FileList
            files={statusQuery.data?.files}
            fileStats={statusQuery.data?.file_stats}
            selectedFiles={selectedFiles}
            selectedKeySet={selectedKeySet}
            selectionKey={selectionKey}
            approvedChanges={
              approvedChangesQuery.data
                ? {
                    available: approvedChangesQuery.data.available,
                    committableFiles: approvedChangesQuery.data.committableFiles,
                    warning: approvedChangesQuery.data.warning
                  }
                : undefined
            }
            approvedPaths={approvedPendingSet}
            onStageApproved={handleStageApproved}
            isStagingApproved={isStaging}
            onSelectFile={(path, staged, event) => {
              handleSelectFile(path, staged, event);
              // On mobile, switch to diff view after selecting a file
              setMobileActivePanel("diff");
            }}
            onStageFile={handleStageFile}
            onUnstageFile={handleUnstageFile}
            onDiscardFile={handleDiscardFile}
            onIgnoreFile={handleIgnoreFile}
            onStageAll={handleStageAll}
            onUnstageAll={handleUnstageAll}
            isStaging={isStaging}
            pendingPaths={pendingPaths}
            isDiscarding={isDiscarding}
            isIgnoring={isIgnoring}
            confirmingDiscard={confirmingDiscard}
            onConfirmDiscard={handleConfirmDiscard}
            confirmingIgnore={confirmingIgnore}
            onConfirmIgnore={handleConfirmIgnore}
            collapsed={false}
            fillHeight={true}
            fileViewMode={fileViewMode}
            groupingRules={groupingRules}
            groupingAvailable={groupingAvailable}
            onCycleViewMode={handleCycleViewMode}

            onStagePaths={handleStagePaths}
            onDiscardPaths={handleDiscardPaths}
            scrollToFile={scrollToFile}
            onScrollComplete={handleScrollComplete}
            scrollTopStore={changesScrollTopRef}
            onDeletePath={handleRequestDeletePath}
            onBlameFile={handleBlameFile}
            repoId={repoId}
            onOpenReview={(slug) => { setReviewScenarioSlug(slug); setMobileActivePanel("review"); }}
            mobileSelectionMode={mobileSelectionMode}
            onEnterSelectionMode={handleEnterSelectionMode}
            onExitSelectionMode={handleExitSelectionMode}
            onMobileSelectFile={handleMobileSelectFile}
            fileHotspots={statusQuery.data?.file_hotspots}
          />
        );
      case "diff":
        return (
          <DiffViewer
            diff={diffQuery.data}
            selectedFile={selectedFile}
            isStaged={selectedIsStaged}
            isUntracked={selectedIsUntracked}
            isLoading={diffQuery.isLoading}
            error={diffQuery.error}
            repoDir={statusQuery.data?.repo_dir}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            onStage={handleStageFile}
            onUnstage={handleUnstageFile}
            onDiscard={handleDiscardFile}
            isStaging={isStaging}
            isDiscarding={isDiscarding}
            isHistoryMode={isHistoryMode}
            commitHash={viewingCommit?.hash}
            onShowRelatedFiles={(path) => {
              handleShowRelatedFiles(path);
              // On mobile, switch to changes view to see the related files panel
              setMobileActivePanel("changes");
            }}
            onOpenSearch={() => setIsFileSearchOpen(true)}
            onOpenReview={() => setMobileActivePanel("review")}
            isReadOnly={isViewingAnyFile}
            onSaveFileContent={handleSaveFileContent}
            isSavingFile={saveFileContentMutation.isPending}
            onDeletePath={handleRequestDeletePath}
            isDeleting={isDeleting}
          />
        );
      case "commit":
        return (
          <CommitPanel
            stagedCount={statusQuery.data?.summary.staged ?? 0}
            commitMessage={commitMessage}
            onCommitMessageChange={setCommitMessage}
            canUseApprovedMessage={canUseApprovedMessage}
            onUseApprovedMessage={handleUseApprovedMessage}
            isUsingApprovedMessage={approvedPreviewMutation.isPending}
            onCommit={handleCommit}
            isCommitting={commitMutation.isPending || precommitStream.state.running}
            precommitProgress={precommitProgressProps}
            commitError={commitError}
            onRetryWithoutPrecommit={handleCommitSkipPrecommit}
            canRetryWithoutPrecommit={Boolean(commitError && pendingPrecommitCommit)}
            defaultAuthorName={statusQuery.data?.author?.name}
            defaultAuthorEmail={statusQuery.data?.author?.email}
            canAmend={canAmend}
            amendDisabledReason={amendDisabledReason}
            collapsed={false}
            fillHeight={true}
            onPush={handlePush}
            isPushing={pushMutation.isPending}
            canPush={syncStatusQuery.data?.can_push ?? false}
            aheadCount={syncStatusQuery.data?.ahead ?? 0}
            pushTarget={pushTargetRef}
            sourceBranch={pushSourceBranch}
            isHistoryMode={isHistoryMode}
            historyCommit={viewingCommit}
          />
        );
      case "history":
        return (
          <GitHistory
            lines={historyQuery.data?.lines}
            entries={historyQuery.data?.entries}
            isLoading={historyQuery.isLoading}
            error={historyQuery.error}
            collapsed={false}
            fillHeight={true}
            onLoadMore={handleLoadMoreHistory}
            isFetching={historyQuery.isFetching}
            hasMore={
              !historyGrepPrefix &&
              (historyQuery.data?.lines?.length ?? 0) >= historyLimit &&
              historyLimit < historyMaxLimit
            }
            searchQuery={historySearch}
            onSearchQueryChange={setHistorySearch}
            scopeFilter={historyScopeFilter}
            onScopeFilterChange={setHistoryScopeFilter}
            groupingEnabled={fileViewMode === "grouped"}
            groupingRules={groupingRules}
            workingSetPaths={workingSetPaths}
            workingSetOnly={historyWorkingSetOnly}
            onWorkingSetOnlyChange={setHistoryWorkingSetOnly}
            filtersOpen={isHistoryFiltersOpen}
            onOpenFilters={() => setIsHistoryFiltersOpen(true)}
            onCloseFilters={() => setIsHistoryFiltersOpen(false)}
            selectedCommitHash={viewingCommit?.hash}
            onSelectCommit={(entry) => {
              handleSelectCommit(entry);
              // On mobile, switch to changes view after selecting a commit to see files
              if (entry) {
                setMobileActivePanel("changes");
              }
            }}
            blameFilePath={viewingFileBlame?.path}
            blameFileName={viewingFileBlame?.filename}
            onExitBlameMode={handleExitBlameMode}
            onContinueCommit={handleContinueCommit}
            activeGroupFilter={activeGroupFilter}
            onFilterGroup={setHistoryGrepPrefix}
            onClearGroupFilter={() => setHistoryGrepPrefix(null)}
          />
        );
      case "review":
        return (
          <ScenarioReviewPanel
            scenarioSlug={reviewScenarioSlug}
            repoId={repoId}
            fileStats={statusQuery.data?.file_stats}
            onChangeScenario={(slug) => { if (reviewScenarioSlug && slug !== reviewScenarioSlug) { scenarioReview.switchScenario(reviewScenarioSlug, slug); } setReviewScenarioSlug(slug); }}
            activeTab={scenarioReview.state.activeTab}
            onActiveTabChange={(tab) => scenarioReview.update({ activeTab: tab })}
            agentRunId={scenarioReview.state.agentRunId}
            onAgentRunIdChange={(id) => scenarioReview.update({ agentRunId: id })}
            scenarioState={scenarioReview.state}
            onScenarioStateChange={scenarioReview.update}
            isMobile
          />
        );
    }
  };

  const pushNoticeTone =
    pushNotice?.tone === "warning"
      ? "bg-amber-950 border-amber-800 text-amber-200"
      : pushNotice?.tone === "info"
        ? "bg-sky-950 border-sky-800 text-sky-200"
        : "bg-emerald-950 border-emerald-800 text-emerald-200";
  const pushNoticeTitle =
    pushNotice?.tone === "warning" ? "Push verification warning" : "Push status";

  // Mobile Layout
  if (isMobile) {
    const stagedCount = statusQuery.data?.summary.staged ?? 0;
    const unstagedCount =
      (statusQuery.data?.summary.unstaged ?? 0) +
      (statusQuery.data?.summary.untracked ?? 0) +
      (statusQuery.data?.summary.conflicts ?? 0);

    return (
      <div
        className="gct-mobile-shell text-slate-50"
        data-testid="git-control-tower"
        data-mobile-shell="true"
        role="application"
      >
        {/* Mobile Header */}
        <MobileHeader
          status={statusQuery.data}
          health={healthQuery.data}
          syncStatus={syncStatusQuery.data}
          branchActions={branchActions}
          repoActions={repoActions}
          onRepoChange={handleRepoChange}
          isLoading={statusQuery.isLoading || healthQuery.isLoading}
          onRefresh={handleRefresh}
          onOpenSettings={() => setIsSettingsOpen(true)}

          onOpenUpstreamInfo={() => setIsUpstreamInfoOpen(true)}
          onOpenFileSearch={() => setIsFileSearchOpen(true)}
          onOpenReview={() => setMobileActivePanel("review")}
          viewingCommit={viewingCommit}
          onExitHistoryMode={handleExitHistoryMode}
          viewingFileBlame={viewingFileBlame}
          onExitBlameMode={handleExitBlameMode}
          onPush={handlePush}
          onPull={handlePull}
          isPushing={pushMutation.isPending}
          isPulling={pullMutation.isPending}
        />

        {/* Main Content - Single Panel at a time */}
        <div className="gct-mobile-content" data-testid="gct-mobile-content">
          {renderMobilePanel(mobileActivePanel)}
        </div>

        {/* Mobile Navigation */}
        <MobileNav
          activePanel={mobileActivePanel}
          onPanelChange={setMobileActivePanel}
          stagedCount={stagedCount}
          unstagedCount={unstagedCount}
        />

        {/* Error Toast for Mutations - positioned above bottom nav */}
        {(stageMutation.error ||
          unstageMutation.error ||
          discardMutation.error ||
          ignoreMutation.error ||
          pushMutation.error ||
          pullMutation.error ||
          createBranchMutation.error ||
          switchBranchMutation.error ||
          publishBranchMutation.error) && (
          <div
            className="fixed bottom-20 left-4 right-4 px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm shadow-lg"
            data-testid="error-toast"
          >
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="font-medium">Operation failed</p>
                <p className="text-xs mt-1 text-red-300">
                  {(
                    stageMutation.error ||
                    unstageMutation.error ||
                    discardMutation.error ||
                    ignoreMutation.error ||
                    pushMutation.error ||
                    pullMutation.error ||
                    createBranchMutation.error ||
                    switchBranchMutation.error ||
                    publishBranchMutation.error
                  )?.message}
                </p>
              </div>
              <button
                type="button"
                onClick={() => {
                  stageMutation.reset();
                  unstageMutation.reset();
                  discardMutation.reset();
                  ignoreMutation.reset();
                  pushMutation.reset();
                  pullMutation.reset();
                  createBranchMutation.reset();
                  switchBranchMutation.reset();
                  publishBranchMutation.reset();
                }}
                className="text-red-400 hover:text-red-200 p-1"
                aria-label="Dismiss"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>
        )}
        {/* Warning Toast - positioned above bottom nav */}
        {warningNotice && (
          <div
            className="fixed bottom-20 left-4 right-4 px-4 py-3 rounded-lg bg-amber-950 border border-amber-800 text-amber-200 text-sm shadow-lg"
            data-testid="warning-toast"
          >
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="font-medium">{warningNotice.message}</p>
                {warningNotice.details && (
                  <p className="text-xs mt-1 text-amber-300 whitespace-pre-wrap max-h-32 overflow-y-auto">
                    {warningNotice.details}
                  </p>
                )}
              </div>
              <button
                type="button"
                onClick={() => setWarningNotice(null)}
                className="text-amber-400 hover:text-amber-200 p-1"
                aria-label="Dismiss"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>
        )}
        {pushNotice && (
          <div
            className={`fixed bottom-20 left-4 right-4 px-4 py-3 rounded-lg border text-sm ${pushNoticeTone}`}
            data-testid="push-toast"
          >
            <p className="font-medium">{pushNoticeTitle}</p>
            <p className="text-xs mt-1">{pushNotice.message}</p>
          </div>
        )}

        {/* Modals */}
        <SettingsModal
          isOpen={isSettingsOpen}
          repoDir={repoDir}
          repoId={repoId}
          syncStatus={syncStatusQuery.data}
          preset={layoutPreset}
          primaryPanel={primaryPanel}
          onChangePreset={setLayoutPreset}
          onChangePrimary={setPrimaryPanel}
          onResetLayout={() => {
            setLayoutPreset("classic");
            setPrimaryPanel("diff");
            setStackHeight(320);
          }}
          groupingEnabled={fileViewMode === "grouped"}
          onToggleGrouping={() => setFileViewMode((prev) => prev === "grouped" ? "flat" : "grouped")}
          groupingRules={groupingRules}
          onChangeGroupingRules={setGroupingRules}
          onClose={() => setIsSettingsOpen(false)}
        />
        <UpstreamInfoModal
          isOpen={isUpstreamInfoOpen}
          localBranch={pushSourceBranch}
          upstreamRef={pushTargetRef}
          ahead={upstreamAhead}
          behind={upstreamBehind}
          repoId={repoId}
          onClose={() => setIsUpstreamInfoOpen(false)}
        />
        <DiscardConfirmationModal
          isOpen={pendingDiscardFiles !== null}
          files={pendingDiscardFiles ?? []}
          isLoading={discardMutation.isPending}
          onConfirm={handleDiscardMultiple}
          onCancel={() => setPendingDiscardFiles(null)}
        />
        <DeleteConfirmationModal
          isOpen={pendingDeletePath !== null}
          path={pendingDeletePath?.path ?? ""}
          isDirectory={pendingDeletePath?.isDir ?? false}
          isLoading={isDeleting}
          onConfirm={handleConfirmDelete}
          onCancel={handleCancelDelete}
        />
        <MobileFileSearch
          isOpen={isFileSearchOpen}
          onClose={() => setIsFileSearchOpen(false)}
          onSelectFile={handleSelectAnyFile}
          repoId={repoId}
        />
      </div>
    );
  }

  // Desktop Layout (original)
  return (
    <div
      className="h-full flex flex-col bg-slate-950 text-slate-50"
      data-testid="git-control-tower"
      role="application"
    >
      {/* Status Header */}
      <StatusHeader
        status={statusQuery.data}
        health={healthQuery.data}
        syncStatus={syncStatusQuery.data}
        branchActions={branchActions}
        repoActions={repoActions}
        onRepoChange={handleRepoChange}
        isLoading={statusQuery.isLoading || healthQuery.isLoading}
        onRefresh={handleRefresh}
        onOpenSettings={() => setIsSettingsOpen(true)}
        onOpenUpstreamInfo={() => setIsUpstreamInfoOpen(true)}
        onOpenFileSearch={() => setIsFileSearchOpen(true)}
        onOpenReview={() => setPrimaryPanel("review")}
        viewingCommit={viewingCommit}
        onExitHistoryMode={handleExitHistoryMode}
        viewingFileBlame={viewingFileBlame}
        onExitBlameMode={handleExitBlameMode}
        onPush={handlePush}
        onPull={handlePull}
        isPushing={pushMutation.isPending}
        isPulling={pullMutation.isPending}
      />

      {/* Main Content - Layout */}
      <div
        className={`flex-1 overflow-hidden ${isBottomLayout ? "flex flex-col" : "flex"}`}
        ref={mainRef}
      >
        {!isBottomLayout && stackPosition === "left" && (
          <>
            {stackPanel}
            <div
              className="w-1 bg-slate-900 hover:bg-slate-800 cursor-col-resize"
              onMouseDown={handleStackResizeStart}
              aria-hidden="true"
            />
          </>
        )}

        <div className="flex-1 min-w-0 min-h-0 overflow-hidden">
          {renderPanel(primaryPanel, "main")}
        </div>

        {!isBottomLayout && stackPosition === "right" && (
          <>
            <div
              className="w-1 bg-slate-900 hover:bg-slate-800 cursor-col-resize"
              onMouseDown={handleStackResizeStart}
              aria-hidden="true"
            />
            {stackPanel}
          </>
        )}

        {isBottomLayout && (
          <>
            <div
              className="h-1 bg-slate-900 hover:bg-slate-800 cursor-row-resize"
              onMouseDown={handleStackResizeStart}
              aria-hidden="true"
            />
            {stackPanel}
          </>
        )}
      </div>

      {/* Error Toast for Mutations */}
      {(stageMutation.error ||
        unstageMutation.error ||
        discardMutation.error ||
        ignoreMutation.error ||
        pushMutation.error ||
        pullMutation.error ||
        createBranchMutation.error ||
        switchBranchMutation.error ||
        publishBranchMutation.error) && (
        <div
          className="fixed bottom-4 right-4 px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm max-w-md shadow-lg"
          data-testid="error-toast"
        >
          <div className="flex items-start justify-between gap-2">
            <div>
              <p className="font-medium">Operation failed</p>
              <p className="text-xs mt-1 text-red-300">
                {(
                  stageMutation.error ||
                  unstageMutation.error ||
                  discardMutation.error ||
                  ignoreMutation.error ||
                  pushMutation.error ||
                  pullMutation.error ||
                  createBranchMutation.error ||
                  switchBranchMutation.error ||
                  publishBranchMutation.error
                )?.message}
              </p>
            </div>
            <button
              type="button"
              onClick={() => {
                stageMutation.reset();
                unstageMutation.reset();
                discardMutation.reset();
                ignoreMutation.reset();
                pushMutation.reset();
                pullMutation.reset();
                createBranchMutation.reset();
                switchBranchMutation.reset();
                publishBranchMutation.reset();
              }}
              className="text-red-400 hover:text-red-200 p-1"
              aria-label="Dismiss"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
      {/* Warning Toast */}
      {warningNotice && (
        <div
          className="fixed bottom-4 right-4 max-w-md px-4 py-3 rounded-lg bg-amber-950 border border-amber-800 text-amber-200 text-sm shadow-lg"
          data-testid="warning-toast"
        >
          <div className="flex items-start justify-between gap-2">
            <div>
              <p className="font-medium">{warningNotice.message}</p>
              {warningNotice.details && (
                <p className="text-xs mt-1 text-amber-300 whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {warningNotice.details}
                </p>
              )}
            </div>
            <button
              type="button"
              onClick={() => setWarningNotice(null)}
              className="text-amber-400 hover:text-amber-200 p-1"
              aria-label="Dismiss"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
      {pushNotice && (
        <div
          className={`fixed bottom-4 right-4 px-4 py-3 rounded-lg border text-sm max-w-md ${pushNoticeTone}`}
          data-testid="push-toast"
        >
          <p className="font-medium">{pushNoticeTitle}</p>
          <p className="text-xs mt-1">{pushNotice.message}</p>
        </div>
      )}
      <SettingsModal
        isOpen={isSettingsOpen}
        repoDir={repoDir}
        repoId={repoId}
        syncStatus={syncStatusQuery.data}
        preset={layoutPreset}
        primaryPanel={primaryPanel}
        onChangePreset={setLayoutPreset}
        onChangePrimary={setPrimaryPanel}
        onResetLayout={() => {
          setLayoutPreset("classic");
          setPrimaryPanel("diff");
          setStackHeight(320);
        }}
        groupingEnabled={fileViewMode === "grouped"}
        onToggleGrouping={() => setFileViewMode((prev) => prev === "grouped" ? "flat" : "grouped")}
        groupingRules={groupingRules}
        onChangeGroupingRules={setGroupingRules}
        onClose={() => setIsSettingsOpen(false)}
      />
      <UpstreamInfoModal
        isOpen={isUpstreamInfoOpen}
        localBranch={pushSourceBranch}
        upstreamRef={pushTargetRef}
        ahead={upstreamAhead}
        behind={upstreamBehind}
        repoId={repoId}
        onClose={() => setIsUpstreamInfoOpen(false)}
      />
      <DiscardConfirmationModal
        isOpen={pendingDiscardFiles !== null}
        files={pendingDiscardFiles ?? []}
        isLoading={discardMutation.isPending}
        onConfirm={handleDiscardMultiple}
        onCancel={() => setPendingDiscardFiles(null)}
      />
      <DeleteConfirmationModal
        isOpen={pendingDeletePath !== null}
        path={pendingDeletePath?.path ?? ""}
        isDirectory={pendingDeletePath?.isDir ?? false}
        isLoading={isDeleting}
        onConfirm={handleConfirmDelete}
        onCancel={handleCancelDelete}
      />
      <FileSearchModal
        isOpen={isFileSearchOpen}
        onClose={() => setIsFileSearchOpen(false)}
        onSelectFile={handleSelectAnyFile}
        repoId={repoId}
      />
    </div>
  );
}

import { useCallback, useState } from "react";
import type { ContentSearchRequest, ViewMode } from "./api";

export const queryKeys = {
  health: ["health"] as const,
  repoStatus: (repoId?: string | null) => ["repo", "status", repoId ?? "default"] as const,
  repoHistory: (limit?: number, includeFiles?: boolean, repoId?: string | null, grep?: string, includeChecks?: boolean) =>
    ["repo", "history", repoId ?? "default", limit, includeFiles, grep, includeChecks] as const,
  syncStatus: (repoId?: string | null) => ["repo", "sync-status", repoId ?? "default"] as const,
  branches: (repoId?: string | null) => ["repo", "branches", repoId ?? "default"] as const,
  diff: (
    path?: string,
    staged?: boolean,
    untracked?: boolean,
    commit?: string,
    mode?: ViewMode,
    any?: boolean,
    repoId?: string | null
  ) => ["repo", "diff", repoId ?? "default", path, staged, untracked, commit, mode, any] as const,
  approvedChanges: (repoId?: string | null) =>
    ["repo", "approved-changes", repoId ?? "default"] as const,
  provenance: (repoId?: string | null) =>
    ["repo", "provenance", repoId ?? "default"] as const,
  files: (pattern?: string, deep?: boolean, repoId?: string | null) =>
    ["repo", "files", repoId ?? "default", pattern, deep] as const,
  relatedFiles: (path: string, repoId?: string | null) =>
    ["repo", "related", repoId ?? "default", path] as const,
  directoryContents: (path: string, repoId?: string | null) =>
    ["repo", "dir", repoId ?? "default", path] as const,
  contentSearch: (query: string, opts?: Partial<ContentSearchRequest>, repoId?: string | null) =>
    ["repo", "search", "content", repoId ?? "default", query, opts] as const,
  credentials: (repoId?: string | null) => ["credentials", repoId ?? "default"] as const,
  groupingRules: (repoId?: string | null) => ["repo", "grouping-rules", repoId ?? "default"] as const,
  gitignoreHealth: (repoId?: string | null) => ["repo", "gitignore", "health", repoId ?? "default"] as const,
  capabilities: ["capabilities"] as const,
  sshKeys: ["ssh", "keys"] as const,
  repos: ["repos"] as const,
  activeRepo: ["repos", "active"] as const,
  visualCaptures: (slug: string, repoId?: string | null) =>
    ["repo", "visual-captures", repoId ?? "default", slug] as const,
  visualCaptureDetail: (id: string, slug: string, repoId?: string | null) =>
    ["repo", "visual-captures", repoId ?? "default", "detail", id, slug] as const,
  captureStorage: (repoId?: string | null) =>
    ["repo", "visual-capture-storage", repoId ?? "default"] as const,
  workflowCaptures: (slug: string, repoId?: string | null) =>
    ["repo", "workflow-captures", repoId ?? "default", slug] as const,
  testExecutions: (scenarioName: string, repoId?: string | null) =>
    ["repo", "test-executions", repoId ?? "default", scenarioName] as const,
  testExecution: (id: string, repoId?: string | null) =>
    ["repo", "test-executions", repoId ?? "default", "detail", id] as const,
  tidinessScore: (scenarioName: string, repoId?: string | null) =>
    ["repo", "tidiness-score", repoId ?? "default", scenarioName] as const,
  tidinessIssues: (scenarioName: string, file?: string, repoId?: string | null, category?: string, severity?: string, limit?: number) =>
    ["repo", "tidiness-issues", repoId ?? "default", scenarioName, file, category, severity, limit] as const,
  tidinessStaleness: (scenarioName: string, repoId?: string | null) =>
    ["repo", "tidiness-staleness", repoId ?? "default", scenarioName] as const,
  tidinessScenarioDetail: (scenarioName: string, repoId?: string | null) =>
    ["repo", "tidiness-scenario", repoId ?? "default", scenarioName] as const,
  scenarios: ["scenarios"] as const,
  agentProfiles: ["agent", "profiles"] as const,
  agentRuns: (slug: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", slug] as const,
  agentRun: (runId: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", "detail", runId] as const,
  agentRunEvents: (runId: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", "events", runId] as const,
  agentRunDiff: (runId: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", "diff", runId] as const,
  rulesRun: (scenarioName: string, repoId?: string | null) =>
    ["repo", "rules-run", repoId ?? "default", scenarioName] as const,
  rulesJob: (jobId: string, repoId?: string | null) =>
    ["repo", "rules-job", repoId ?? "default", jobId] as const,
  rulesList: (repoId?: string | null) =>
    ["repo", "rules-list", repoId ?? "default"] as const,
  rulesViolations: (scenarioName: string, repoId?: string | null) =>
    ["repo", "rules-violations", repoId ?? "default", scenarioName] as const,
  reviewSummary: (scenarioName: string, repoId?: string | null) =>
    ["review", "summary", repoId ?? "default", scenarioName] as const,
  reviewJob: (jobId: string, repoId?: string | null) =>
    ["review", "job", repoId ?? "default", jobId] as const,
};

const REPO_STORAGE_KEY = "gct.activeRepoId";

function readStoredRepoId(): string | null {
  if (typeof window === "undefined") return null;
  try {
    const value = window.localStorage.getItem(REPO_STORAGE_KEY);
    return value && value.trim().length > 0 ? value : null;
  } catch {
    return null;
  }
}

function persistRepoId(repoId: string | null) {
  if (typeof window === "undefined") return;
  try {
    if (repoId && repoId.trim().length > 0) {
      window.localStorage.setItem(REPO_STORAGE_KEY, repoId);
    } else {
      window.localStorage.removeItem(REPO_STORAGE_KEY);
    }
  } catch {
    // Ignore storage errors; repo selection still works in-memory.
  }
}

export function useRepoSelection() {
  const [repoId, setRepoIdState] = useState<string | null>(() => readStoredRepoId());

  const setRepoId = useCallback((next: string | null) => {
    const normalized = next && next.trim().length > 0 ? next : null;
    setRepoIdState(normalized);
    persistRepoId(normalized);
  }, []);

  return { repoId, setRepoId };
}

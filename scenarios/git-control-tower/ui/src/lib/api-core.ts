// ============================================================================
// Core Git/Repo API Functions
// ============================================================================

import { API_BASE, buildRepoHeaders, handleResponse, buildApiUrl, extractErrorMessage } from "./api-internals";
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  ConfirmMutationRequestSchema,
  GetAuthorityStatusRequestSchema,
  PrepareMutationRequestSchema,
} from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import {
	CreateBranchRequestSchema,
  ListBranchesRequestSchema,
  PublishBranchRequestSchema,
  SwitchBranchRequestSchema,
  type BranchInfo as ProtoBranchInfo,
  type BranchWarning as ProtoBranchWarning,
} from "@vrooli/proto-types/git-control-tower/v1/branch/branch_pb";
import { branchClient, humanControlClient, repoClient } from "./connect";
import { CreateCommitRequestSchema, DeletePathRequestSchema, DiscardFilesRequestSchema, GetApprovedChangesRequestSchema, GetDirectoryContentsRequestSchema, GetFilesRequestSchema, GetPrecommitConfigRequestSchema, GetProvenanceRequestSchema, GetRelatedFilesRequestSchema, GetRepoDiffRequestSchema, GetRepoHistoryRequestSchema, GetRepoStatusRequestSchema, GetSyncStatusRequestSchema, IgnorePathRequestSchema, PullFromRemoteRequestSchema, PushToRemoteRequestSchema, RunPrecommitRequestSchema, RunUpstreamActionRequestSchema, SaveFileContentRequestSchema, SavePrecommitConfigRequestSchema, SearchContentRequestSchema, SearchProvenanceRequestSchema, StageFilesRequestSchema, UnstageFilesRequestSchema } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import type { HealthResponse, RepoStatus, RepoHistoryResponse, RepoBranchesResponse, BranchInfo, BranchWarning, BranchCreateResponse, BranchSwitchResponse, BranchPublishResponse, CreateBranchRequest, SwitchBranchRequest, PublishBranchRequest, CommitCheckKind, CommitCheckStatus } from "./api-types-repo";
import {
  FileContentConflictError,
  RemoteOperationError,
  type ViewMode,
  type DiffResponse,
  type StageRequest,
  type StageResponse,
  type UnstageRequest,
  type UnstageResponse,
  type CommitRequest,
  type CommitResponse,
  type AuthorityStatus,
  type MutationPreviewRequest,
  type MutationPreviewResponse,
  type MutationIntentRequest,
  type MutationIntentResponse,
  type PrecommitConfig,
  type PrecommitRunResult,
  type PrecommitRunRequest,
  type PrecommitRunResponse,
  type PrecommitStreamEvent,
  type DiscardRequest,
  type DiscardResponse,
  type IgnoreRequest,
  type IgnoreResponse,
  type PushRequest,
  type PushResponse,
  type PullRequest,
  type PullResponse,
  type UpstreamActionRequest,
  type UpstreamActionResponse,
  type SyncStatusResponse,
  type ApprovedChangesResponse,
  type ApprovedChangesPreviewRequest,
  type ProvenanceResponse,
  type BlameResponse,
  type FileTreeResponse,
  type RelatedFilesResponse,
  type ContentSearchRequest,
  type ContentSearchResponse,
  type DirListResponse,
  type DeletePathRequest,
  type DeletePathResponse,
  type SaveFileContentRequest,
  type SaveFileContentResponse,
} from "./api-types-operations";

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<HealthResponse>(res);
}

export async function fetchRepoStatus(repoId?: string, hotspots = false): Promise<RepoStatus> {
  const response = await repoClient.getRepoStatus(create(GetRepoStatusRequestSchema, {
    repositoryId: repoId ?? "",
    includeHotspots: hotspots,
  }));
  const branch = response.branchStatus;
  const files = response.files;
  const summary = response.summary;
  const mapStats = (value?: Record<string, { additions: number; deletions: number; files: number; netLines: number; hunkCount: number; largestHunk: number; density: number; isBinary: boolean; isRename: boolean; oldPath: string; commentAdditions: number; commentDeletions: number; isNewFile: boolean; isDeletedFile: boolean }>) => value ? Object.fromEntries(Object.entries(value).map(([path, stats]) => [path, {
    additions: stats.additions,
    deletions: stats.deletions,
    files: stats.files,
    net_lines: stats.netLines,
    hunk_count: stats.hunkCount,
    largest_hunk: stats.largestHunk,
    density: stats.density,
    is_binary: stats.isBinary,
    is_rename: stats.isRename,
    old_path: stats.oldPath,
    comment_additions: stats.commentAdditions,
    comment_deletions: stats.commentDeletions,
    is_new_file: stats.isNewFile,
    is_deleted_file: stats.isDeletedFile,
  }])) : undefined;
  return {
    repo_dir: response.repoDir,
    branch: {
      head: branch?.head ?? response.branch,
      upstream: branch?.upstream ?? "",
      ahead: branch?.ahead ?? 0,
      behind: branch?.behind ?? 0,
      oid: branch?.oid ?? "",
    },
    files: {
      staged: files?.staged ?? [],
      unstaged: files?.unstaged ?? [],
      untracked: files?.untracked ?? [],
      conflicts: files?.conflicts ?? [],
      binary: files?.binary ?? [],
      ignored: files?.ignored ?? [],
      statuses: files?.statuses ?? {},
      renames: files?.renames ?? {},
    },
    file_stats: response.fileStats ? {
      staged: mapStats(response.fileStats.staged),
      unstaged: mapStats(response.fileStats.unstaged),
      untracked: mapStats(response.fileStats.untracked),
    } : undefined,
    file_hotspots: response.fileHotspots,
    scopes: Object.fromEntries(Object.entries(response.scopes).map(([key, value]) => [key, value.values])),
    summary: {
      staged: summary?.staged ?? 0,
      unstaged: summary?.unstaged ?? 0,
      untracked: summary?.untracked ?? 0,
      conflicts: summary?.conflicts ?? 0,
      ignored: summary?.ignored ?? 0,
    },
    author: {
      name: response.author?.name ?? "",
      email: response.author?.email ?? "",
    },
    timestamp: response.timestamp,
  };
}

export async function fetchRepoHistory(
  limit = 30,
  includeFiles = false,
  repoId?: string,
  grep?: string,
  includeChecks = false
): Promise<RepoHistoryResponse> {
  const response = await repoClient.getRepoHistory(create(GetRepoHistoryRequestSchema, {
    repositoryId: repoId ?? "",
    limit,
    includeFiles,
    includeChecks,
    grepPattern: grep ?? "",
  }));
  return {
    repo_dir: response.repoDir,
    lines: response.lines,
    entries: response.entries.map((entry) => ({
      hash: entry.hash,
      author: entry.author,
      date: entry.date,
      subject: entry.subject,
      files: entry.files,
      checks: entry.checks.map((check) => ({
        kind: check.kind as CommitCheckKind,
        status: check.status as CommitCheckStatus,
        command: check.command,
        exit_code: check.exitCode,
        summary: check.summary,
        stdout: check.stdout,
        stderr: check.stderr,
        duration_ms: Number(check.durationMs),
        timestamp: check.timestamp,
      })),
    })),
    limit: response.limit,
    grep_pattern: response.grepPattern,
    timestamp: response.timestamp,
  };
}

export async function fetchDiff(
  path?: string,
  staged = false,
  untracked = false,
  commit?: string,
  mode: ViewMode = "diff",
  any = false,
  repoId?: string
): Promise<DiffResponse> {
  const response = await repoClient.getRepoDiff(create(GetRepoDiffRequestSchema, {
    repositoryId: repoId ?? "",
    path: path ?? "",
    staged,
    untracked,
    commit: commit ?? "",
    mode,
    any,
  }));
  const stats = response.stats;
  return {
    repo_dir: response.repoDir,
    path: response.path,
    staged: response.staged,
    untracked: response.untracked,
    base: response.base,
    has_diff: response.hasDiff,
    hunks: response.hunks.map((hunk) => ({
      header: hunk.header,
      old_start: hunk.oldStart,
      old_lines: hunk.oldCount,
      new_start: hunk.newStart,
      new_lines: hunk.newCount,
      lines: hunk.lines,
    })),
    stats: {
      additions: stats?.additions ?? 0,
      deletions: stats?.deletions ?? 0,
      files: stats?.files ?? 0,
      net_lines: stats?.netLines ?? 0,
      hunk_count: stats?.hunkCount ?? 0,
      largest_hunk: stats?.largestHunk ?? 0,
      density: stats?.density ?? 0,
      is_binary: stats?.isBinary ?? false,
      is_rename: stats?.isRename ?? false,
      old_path: stats?.oldPath ?? "",
      comment_additions: stats?.commentAdditions ?? 0,
      comment_deletions: stats?.commentDeletions ?? 0,
      is_new_file: stats?.isNewFile ?? false,
      is_deleted_file: stats?.isDeletedFile ?? false,
    },
    raw: response.raw,
    full_content: response.fullContent,
    content_hash: response.contentHash,
    annotated_lines: response.annotatedLines.map((line) => ({
      number: line.number,
      content: line.content,
      change: line.change as "" | "added" | "deleted" | "modified",
      old_number: line.oldNumber,
    })),
    mode: response.mode as ViewMode,
    timestamp: response.timestamp,
  };
}

export async function stageFiles(request: StageRequest, repoId?: string): Promise<StageResponse> {
	const intent = await issueMutationIntentForOperation("repo.stage", repoId);
	const response = await repoClient.stageFiles(create(StageFilesRequestSchema, {
		repositoryId: intent.repositoryId, intentId: intent.intentId, paths: request.paths, scope: request.scope || "",
	}));
	return {
		success: response.success, staged: response.staged, failed: response.failed, errors: response.errors,
		warnings: response.warnings, timestamp: response.timestamp,
	};
}

export async function unstageFiles(request: UnstageRequest, repoId?: string): Promise<UnstageResponse> {
	const intent = await issueMutationIntentForOperation("repo.unstage", repoId);
	const response = await repoClient.unstageFiles(create(UnstageFilesRequestSchema, {
		repositoryId: intent.repositoryId, intentId: intent.intentId, paths: request.paths, scope: request.scope || "",
	}));
	return {
		success: response.success, unstaged: response.unstaged, failed: response.failed, errors: response.errors,
		timestamp: response.timestamp,
	};
}

export async function createCommit(request: CommitRequest, repoId?: string): Promise<CommitResponse> {
  return repoClient.createCommit(create(CreateCommitRequestSchema, {
    repositoryId: repoId || "",
    intentId: request.intent_id || "",
    message: request.message || "",
    validateConventional: request.validate_conventional || false,
    amend: request.amend || false,
    authorName: request.author_name || "",
    authorEmail: request.author_email || "",
    skipPrecommitOnce: request.skip_precommit_once || false,
  }));
}

export async function fetchAuthorityStatus(): Promise<AuthorityStatus> {
  return humanControlClient.getAuthorityStatus(create(GetAuthorityStatusRequestSchema, {}));
}

export async function fetchMutationPreview(request: MutationPreviewRequest, repoId?: string): Promise<MutationPreviewResponse> {
	return humanControlClient.prepareMutation(create(PrepareMutationRequestSchema, {
		repositoryId: request.repositoryId || repoId || "",
		operation: request.operation || "repo.commit",
		subjectContext: request.subjectContext || "",
	}));
}

export async function issueMutationIntent(request: MutationIntentRequest, repoId?: string): Promise<MutationIntentResponse> {
  return humanControlClient.confirmMutation(create(ConfirmMutationRequestSchema, {
    repositoryId: request.repositoryId || repoId || "",
    operation: request.operation,
		expectedRevision: request.expectedRevision,
		subjectDigest: request.subjectDigest,
		subjectContext: request.subjectContext || "",
	}));
}

export async function fetchPrecommitConfig(repoId?: string): Promise<PrecommitConfig> {
  const response = await repoClient.getPrecommitConfig(create(GetPrecommitConfigRequestSchema, { repositoryId: repoId ?? "" }));
  return precommitConfigFromProto(response);
}

export async function savePrecommitConfig(config: PrecommitConfig, repoId?: string): Promise<PrecommitConfig> {
  const intent = await issueMutationIntentForOperation("repo.precommit", repoId);
  const response = await repoClient.savePrecommitConfig(create(SavePrecommitConfigRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId,
    enabled: config.enabled, command: config.command, workingDirectory: config.working_directory,
    timeoutSeconds: config.timeout_seconds, runBeforeCommit: config.run_before_commit, allowOverride: config.allow_override,
  }));
  return precommitConfigFromProto(response);
}

export async function runPrecommit(request: PrecommitRunRequest = {}, repoId?: string): Promise<PrecommitRunResponse> {
  const response = await repoClient.runPrecommit(create(RunPrecommitRequestSchema, {
    repositoryId: repoId ?? "", command: request.command ?? "", workingDirectory: request.working_directory ?? "", timeoutSeconds: request.timeout_seconds ?? 0,
  }));
  return { success: response.success, result: precommitRunResultFromProto(response.result) };
}

export interface PrecommitStreamHandlers {
  onEvent?: (event: PrecommitStreamEvent) => void;
  signal?: AbortSignal;
}

export async function runPrecommitStream(
  request: PrecommitRunRequest = {},
  repoId: string | undefined,
  handlers: PrecommitStreamHandlers = {}
): Promise<PrecommitStreamEvent> {
  if (handlers.signal?.aborted) throw new Error("precommit run aborted");
  const response = await runPrecommit(request, repoId);
  const event: PrecommitStreamEvent = {
    type: "finished", elapsed_ms: response.result.duration_ms, result: response.result,
  };
  handlers.onEvent?.(event);
  return event;
}

function precommitConfigFromProto(response: {
  enabled: boolean;
  command: string;
  workingDirectory: string;
  timeoutSeconds: number;
  runBeforeCommit: boolean;
  allowOverride: boolean;
  lastResult?: { status: string; command: string; exitCode: number; summary: string; stdout: string; stderr: string; durationMs: bigint | number; overrideAllowed: boolean; timestamp: string };
  hook?: { status: string; reason: string; existingKind: string; existingHookPreview: string; path: string; hooksPath: string; installedAt: string };
}): PrecommitConfig {
  return {
    enabled: response.enabled,
    command: response.command,
    working_directory: response.workingDirectory,
    timeout_seconds: response.timeoutSeconds,
    run_before_commit: response.runBeforeCommit,
    allow_override: response.allowOverride,
    last_result: response.lastResult ? precommitRunResultFromProto(response.lastResult) : undefined,
    hook: response.hook ? {
      status: response.hook.status,
      reason: response.hook.reason,
      existing_kind: response.hook.existingKind,
      existing_hook_preview: response.hook.existingHookPreview,
      path: response.hook.path,
      hooks_path: response.hook.hooksPath,
      installed_at: response.hook.installedAt,
    } : undefined,
  };
}

function precommitRunResultFromProto(result?: { status: string; command: string; exitCode: number; summary: string; stdout: string; stderr: string; durationMs: bigint | number; overrideAllowed: boolean; timestamp: string }): PrecommitRunResult {
  return {
    status: result?.status ?? "error",
    command: result?.command ?? "",
    exit_code: result?.exitCode ?? -1,
    summary: result?.summary ?? "",
    stdout: result?.stdout ?? "",
    stderr: result?.stderr ?? "",
    duration_ms: Number(result?.durationMs ?? 0),
    override_allowed: result?.overrideAllowed ?? false,
    timestamp: result?.timestamp ?? "",
  };
}

function parseSSEEvent(block: string): PrecommitStreamEvent | null {
  let data = "";
  for (const rawLine of block.split("\n")) {
    const line = rawLine.replace(/\r$/, "");
    if (line.startsWith("data:")) {
      data += line.slice(5).replace(/^ /, "");
    }
  }
  if (!data) return null;
  try {
    return JSON.parse(data) as PrecommitStreamEvent;
  } catch {
    return null;
  }
}

export async function discardFiles(request: DiscardRequest, repoId?: string): Promise<DiscardResponse> {
  const intent = await issueMutationIntentForOperation("repo.discard", repoId);
  const response = await repoClient.discardFiles(create(DiscardFilesRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId,
    paths: request.paths, untracked: request.untracked ?? false,
  }));
  return {
    success: response.success, discarded: response.discarded,
    failed: response.failed, errors: response.errors, timestamp: response.timestamp,
  };
}

export async function ignoreFile(request: IgnoreRequest, repoId?: string): Promise<IgnoreResponse> {
  const intent = await issueMutationIntentForOperation("repo.ignore", repoId);
  const response = await repoClient.ignorePath(create(IgnorePathRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId,
    path: request.path, level: request.level ?? "", groupDir: request.group_dir ?? "",
  }));
  return {
    success: response.success, ignored: response.ignored,
    failed: response.failed, errors: response.errors,
    gitignore_path: response.gitignorePath, timestamp: response.timestamp,
  };
}

export async function pushToRemote(
  request: PushRequest = {},
  repoId?: string
): Promise<PushResponse> {
  const intent = await issueMutationIntentForOperation("repo.push", repoId);
  const response = await repoClient.pushToRemote(create(PushToRemoteRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId,
    remote: request.remote ?? "", branch: request.branch ?? "", setUpstream: request.set_upstream ?? false,
  }));
  const result: PushResponse = {
    success: response.success, remote: response.remote, branch: response.branch,
    pushed: response.pushed, up_to_date: response.upToDate, verified: response.verified,
    verification_error: response.verificationError, error: response.error, timestamp: response.timestamp,
  };
  // The server answers 200 with success=false when git itself failed. Returning that as
  // a resolved promise makes a failed push indistinguishable from a successful one at
  // every call site, so the failure is raised here, once, at the boundary.
  if (!result.success) {
    throw new RemoteOperationError("push", result.error || result.verification_error, result);
  }
  return result;
}

export async function pullFromRemote(
  request: PullRequest = {},
  repoId?: string
): Promise<PullResponse> {
  const intent = await issueMutationIntentForOperation("repo.pull", repoId);
  const response = await repoClient.pullFromRemote(create(PullFromRemoteRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId,
    remote: request.remote ?? "", branch: request.branch ?? "",
  }));
  const result: PullResponse = {
    success: response.success, remote: response.remote, branch: response.branch,
    error: response.error, has_conflicts: response.hasConflicts, timestamp: response.timestamp,
  };
  if (!result.success) {
    throw new RemoteOperationError("pull", result.error, result);
  }
  return result;
}

export async function runUpstreamAction(
  request: UpstreamActionRequest,
  repoId?: string
): Promise<UpstreamActionResponse> {
  const intent = await issueMutationIntentForOperation("repo.push", repoId);
  const response = await repoClient.runUpstreamAction(create(RunUpstreamActionRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId,
    action: request.action, remote: request.remote ?? "", branch: request.branch ?? "", upstream: request.upstream ?? "",
  }));
  return {
    success: response.success, action: response.action, remote: response.remote,
    branch: response.branch, upstream: response.upstream, error: response.error, timestamp: response.timestamp,
  };
}

export async function fetchSyncStatus(
  doFetch = false,
  repoId?: string
): Promise<SyncStatusResponse> {
  const response = await repoClient.getSyncStatus(create(GetSyncStatusRequestSchema, {
    repositoryId: repoId ?? "",
    fetch: doFetch,
  }));
  return {
    branch: response.branch,
    upstream: response.upstream,
    remote_url: response.remoteUrl,
    ahead: response.ahead,
    behind: response.behind,
    has_upstream: response.hasUpstream,
    can_push: response.canPush,
    can_pull: response.canPull,
    needs_pull: response.needsPull,
    needs_push: response.needsPush,
    has_uncommitted_changes: response.hasUncommittedChanges,
    safety_warnings: response.safetyWarnings,
    recommendations: response.recommendations,
    fetched: response.fetched,
    fetch_error: response.fetchError,
    timestamp: response.timestamp,
  };
}

export async function fetchApprovedChanges(repoId?: string): Promise<ApprovedChangesResponse> {
  const response = await repoClient.getApprovedChanges(create(GetApprovedChangesRequestSchema, {
    repositoryId: repoId ?? "",
  }));
  return approvedChangesFromProto(response);
}

export async function fetchApprovedChangesPreview(
  request: ApprovedChangesPreviewRequest,
  repoId?: string
): Promise<ApprovedChangesResponse> {
  const response = await repoClient.getApprovedChanges(create(GetApprovedChangesRequestSchema, {
    repositoryId: repoId ?? "",
    paths: request.paths,
  }));
  return approvedChangesFromProto(response);
}

export async function fetchProvenance(repoId?: string): Promise<ProvenanceResponse> {
  const response = await repoClient.getProvenance(create(GetProvenanceRequestSchema, {
    repositoryId: repoId ?? "",
  }));
  return {
    available: response.available,
    warning: response.warning || undefined,
    runGroups: response.runGroups.map((group) => ({
      runId: group.runId,
      sandboxId: group.sandboxId,
      sandboxOwner: group.sandboxOwner,
      latestAppliedAt: group.latestAppliedAt,
      files: group.files.map((file) => ({
        filePath: file.filePath,
        relativePath: file.relativePath,
        changeType: file.changeType,
        appliedAt: file.appliedAt,
        visibility: file.visibility || undefined,
      })),
    })),
  };
}

export async function fetchBlame(paths: string[], repoId?: string): Promise<BlameResponse> {
  const params = new URLSearchParams();
  paths.forEach((path) => params.append("path", path));
  params.set("max_paths", "1");
  params.set("max_lines", "300");
  params.set("max_bytes", String(1 << 20));
  params.set("enrich", "true");
  const response = await handleResponse<{
    revision: string;
    truncated: boolean;
    warnings?: string[];
    files?: Array<Record<string, unknown>>;
    change_bundles?: Array<Record<string, unknown>>;
  }>(await fetch(buildApiUrl(`/repo/blame?${params.toString()}`, { baseUrl: API_BASE }), { headers: buildRepoHeaders(repoId), cache: "no-store" }));
  return {
    revision: response.revision,
    truncated: response.truncated,
    warnings: response.warnings ?? [],
    files: (response.files ?? []).map((rawFile) => {
      const file = rawFile as { path?: string; status?: string; content_digest?: string; reason?: string; standing?: string; downgrade_reasons?: string[]; lines?: Array<{ line: number; content: string; commit?: string; author?: string; author_time?: string; subject?: string }>; evidence?: Array<Record<string, unknown>> };
      return {
        path: file.path ?? "",
        status: file.status ?? "unavailable",
        contentDigest: file.content_digest || undefined,
        reason: file.reason || undefined,
        standing: file.standing || "unknown",
        downgradeReasons: file.downgrade_reasons ?? [],
        lines: (file.lines ?? []).map((line) => ({
          line: line.line, content: line.content, commit: line.commit || undefined, author: line.author || undefined, authorTime: line.author_time || undefined, subject: line.subject || undefined,
        })),
        evidence: (file.evidence ?? []).map((rawEvidence) => {
          const evidence = rawEvidence as { run_id?: string; runId?: string; sandbox_id?: string; sandboxId?: string; content_digest?: string; contentDigest?: string; commit_id?: string; commitId?: string; visibility?: string; commit_state?: string; commitState?: string; run_outcome?: string; runOutcome?: string; conversation_id?: string; conversationId?: string; cost_usd?: number; costUsd?: number; committed_at?: string; committedAt?: string; unavailable?: string[]; work_references?: Array<Record<string, unknown>>; workReferences?: Array<Record<string, unknown>> };
          return {
            runId: evidence.run_id || evidence.runId || undefined, sandboxId: evidence.sandbox_id || evidence.sandboxId || undefined, contentDigest: evidence.content_digest || evidence.contentDigest || undefined, commitId: evidence.commit_id || evidence.commitId || undefined, visibility: evidence.visibility || undefined, commitState: evidence.commit_state || evidence.commitState || undefined, runOutcome: evidence.run_outcome || evidence.runOutcome || undefined, conversationId: evidence.conversation_id || evidence.conversationId || undefined, costUsd: evidence.cost_usd ?? evidence.costUsd, committedAt: evidence.committed_at || evidence.committedAt || undefined, unavailable: evidence.unavailable,
            workReferences: (evidence.work_references ?? evidence.workReferences ?? []).map((rawRef) => { const ref = rawRef as { kind?: string; id?: string; revision?: string; relationship?: string; verified?: boolean; visibility?: string; state?: string; unavailable_reason?: string; unavailableReason?: string }; return { kind: ref.kind ?? "unknown", id: ref.id ?? "", revision: ref.revision || undefined, relationship: ref.relationship || undefined, verified: ref.verified, visibility: ref.visibility || undefined, state: ref.state || undefined, unavailableReason: ref.unavailable_reason || ref.unavailableReason || undefined }; }),
          };
        }),
      };
    }),
    changeBundles: (response.change_bundles ?? []).map((rawBundle) => {
      const bundle = rawBundle as { run_id?: string; runId?: string; sandbox_id?: string; sandboxId?: string; files?: string[]; run_outcome?: string; runOutcome?: string; conversation_id?: string; conversationId?: string; cost_usd?: number; costUsd?: number; gaps?: string[]; work_references?: Array<Record<string, unknown>>; workReferences?: Array<Record<string, unknown>> };
      return {
        runId: bundle.run_id || bundle.runId || undefined, sandboxId: bundle.sandbox_id || bundle.sandboxId || undefined, files: bundle.files ?? [], runOutcome: bundle.run_outcome || bundle.runOutcome || undefined, conversationId: bundle.conversation_id || bundle.conversationId || undefined, costUsd: bundle.cost_usd ?? bundle.costUsd, gaps: bundle.gaps ?? [],
        workReferences: (bundle.work_references ?? bundle.workReferences ?? []).map((rawRef) => { const ref = rawRef as { kind?: string; id?: string; revision?: string; relationship?: string; verified?: boolean; visibility?: string; state?: string; unavailable_reason?: string; unavailableReason?: string }; return { kind: ref.kind ?? "unknown", id: ref.id ?? "", revision: ref.revision || undefined, relationship: ref.relationship || undefined, verified: ref.verified, visibility: ref.visibility || undefined, state: ref.state || undefined, unavailableReason: ref.unavailable_reason || ref.unavailableReason || undefined }; }),
      };
    }),
  };
}

function approvedChangesFromProto(response: Awaited<ReturnType<typeof repoClient.getApprovedChanges>>): ApprovedChangesResponse {
  return {
    available: response.available,
    committableFiles: response.committableFiles,
    suggestedMessage: response.suggestedMessage || undefined,
    warning: response.warning || undefined,
    files: response.files.map((file) => ({
      relativePath: file.relativePath,
      status: file.status,
      sandboxId: file.sandboxId || undefined,
      sandboxOwner: file.sandboxOwner || undefined,
      changeType: file.changeType || undefined,
      agentManagerRunId: file.agentManagerRunId || undefined,
    })),
  };
}

export async function fetchBranches(repoId?: string): Promise<RepoBranchesResponse> {
	const response = await branchClient.listBranches(create(ListBranchesRequestSchema, {
		repositoryId: repoId || "",
	}));
	return {
		current: response.current,
		locals: response.locals.map(branchInfoFromProto),
		remotes: response.remotes.map(branchInfoFromProto),
		timestamp: timestampFromProto(response.timestamp),
	};
}

export async function createBranch(
	request: CreateBranchRequest,
	repoId?: string
): Promise<BranchCreateResponse> {
	const intent = await issueMutationIntentForOperation("repo.branch.create", repoId);
	const response = await branchClient.createBranch(create(CreateBranchRequestSchema, {
		repositoryId: intent.repositoryId,
		intentId: intent.intentId,
		name: request.name || "",
		from: request.from || "",
		checkout: request.checkout || false,
		allowDirty: request.allow_dirty || false,
	}));
	return {
		success: response.success,
		branch: response.branch ? branchInfoFromProto(response.branch) : undefined,
		warning: response.warning ? branchWarningFromProto(response.warning) : undefined,
		error: response.error,
		validation_errors: response.validationErrors,
		timestamp: timestampFromProto(response.timestamp),
	};
}

export async function switchBranch(
	request: SwitchBranchRequest,
	repoId?: string
): Promise<BranchSwitchResponse> {
	const intent = await issueMutationIntentForOperation("repo.branch.switch", repoId);
	const response = await branchClient.switchBranch(create(SwitchBranchRequestSchema, {
		repositoryId: intent.repositoryId,
		intentId: intent.intentId,
		name: request.name || "",
		allowDirty: request.allow_dirty || false,
		trackRemote: request.track_remote || false,
	}));
	return {
		success: response.success,
		branch: response.branch ? branchInfoFromProto(response.branch) : undefined,
		warning: response.warning ? branchWarningFromProto(response.warning) : undefined,
		error: response.error,
		timestamp: timestampFromProto(response.timestamp),
	};
}

export async function publishBranch(
	request: PublishBranchRequest = {},
	repoId?: string
): Promise<BranchPublishResponse> {
	const intent = await issueMutationIntentForOperation("repo.branch.publish", repoId);
	const response = await branchClient.publishBranch(create(PublishBranchRequestSchema, {
		repositoryId: intent.repositoryId,
		intentId: intent.intentId,
		remote: request.remote || "",
		branch: request.branch || "",
		setUpstream: request.set_upstream || false,
		fetch: request.fetch || false,
	}));
	return {
		success: response.success,
		remote: response.remote,
		branch: response.branch,
		warning: response.warning ? branchWarningFromProto(response.warning) : undefined,
		error: response.error,
		timestamp: timestampFromProto(response.timestamp),
	};
}

export async function issueMutationIntentForOperation(operation: string, repoId?: string, subjectContext = ""): Promise<{ repositoryId: string; intentId: string; expectedRevision: string; subjectDigest: string }> {
	const authority = await humanControlClient.getAuthorityStatus(create(GetAuthorityStatusRequestSchema, {}));
	if (!authority.canMutate) {
		throw new Error(authority.reason || "mutation unavailable: authenticate through the configured provider");
	}
	const preview = await humanControlClient.prepareMutation(create(PrepareMutationRequestSchema, {
		repositoryId: repoId || "",
		operation,
		subjectContext,
	}));
	const intent = await humanControlClient.confirmMutation(create(ConfirmMutationRequestSchema, {
		repositoryId: preview.repositoryId,
		operation: preview.operation,
		expectedRevision: preview.expectedRevision,
		subjectDigest: preview.subjectDigest,
		subjectContext,
	}));
	if (!intent.intentId) {
		throw new Error("mutation intent response did not contain an intent id");
	}
	return { repositoryId: preview.repositoryId, intentId: intent.intentId, expectedRevision: preview.expectedRevision, subjectDigest: preview.subjectDigest };
}

function timestampFromProto(value?: { seconds: bigint | number; nanos: number }): string {
	if (!value) return "";
	return new Date(Number(value.seconds) * 1000 + value.nanos / 1_000_000).toISOString();
}

function branchInfoFromProto(value: ProtoBranchInfo): BranchInfo {
	return {
		name: value.name,
		upstream: value.upstream,
		oid: value.oid,
		last_commit_at: timestampFromProto(value.lastCommitAt),
		ahead: value.ahead,
		behind: value.behind,
		is_current: value.isCurrent,
	};
}

function branchWarningFromProto(value: ProtoBranchWarning): BranchWarning {
	return {
		message: value.message,
		requires_confirmation: value.requiresConfirmation,
		requires_tracking: value.requiresTracking,
		requires_fetch: value.requiresFetch,
		dirty_summary: value.dirtySummary ? {
			staged: value.dirtySummary.staged,
			unstaged: value.dirtySummary.unstaged,
			untracked: value.dirtySummary.untracked,
			conflicts: value.dirtySummary.conflicts,
		} : undefined,
	};
}

export async function fetchFiles(
  pattern?: string,
  limit = 1000,
  deep = false,
  timeout = 5000,
  repoId?: string
): Promise<FileTreeResponse> {
  const response = await repoClient.getFiles(create(GetFilesRequestSchema, {
    repositoryId: repoId ?? "", pattern: pattern ?? "", limit, deep, timeoutMs: timeout,
  }));
  return {
    files: response.files.map((file) => ({ path: file.path, language: file.language, status: file.status as FileTreeResponse["files"][number]["status"] })),
    truncated: response.truncated,
    cancelled: response.cancelled,
    search_mode: response.searchMode as FileTreeResponse["search_mode"],
    timestamp: response.timestamp,
  };
}

export async function fetchRelatedFiles(
  path: string,
  repoId?: string
): Promise<RelatedFilesResponse> {
  const response = await repoClient.getRelatedFiles(create(GetRelatedFilesRequestSchema, {
    repositoryId: repoId ?? "", path,
  }));
  return {
    path: response.path,
    related: response.related.map((file) => ({ path: file.path, relation_type: file.relationType as RelatedFilesResponse["related"][number]["relation_type"] })),
    timestamp: response.timestamp,
  };
}

export async function searchContent(
  request: ContentSearchRequest,
  repoId?: string
): Promise<ContentSearchResponse> {
  const response = await repoClient.searchContent(create(SearchContentRequestSchema, {
    repositoryId: repoId ?? "",
    query: request.query,
    caseSensitive: request.case_sensitive ?? false,
    wholeWord: request.whole_word ?? false,
    regex: request.regex ?? false,
    include: request.include ?? "",
    exclude: request.exclude ?? "",
    contextLines: request.context_lines ?? 0,
    limit: request.limit ?? 0,
    timeoutMs: request.timeout ?? 0,
  }));
  return {
    matches: response.matches.map((match) => ({
      path: match.path, line_number: match.lineNumber, content: match.content,
      context_before: match.contextBefore, context_after: match.contextAfter,
    })),
    total: response.total,
    truncated: response.truncated,
    cancelled: response.cancelled,
    query: response.query,
    timestamp: response.timestamp,
  };
}

export async function fetchDirectoryContents(path = "", repoId?: string): Promise<DirListResponse> {
  const response = await repoClient.getDirectoryContents(create(GetDirectoryContentsRequestSchema, {
    repositoryId: repoId ?? "", path,
  }));
  return {
    path: response.path,
    entries: response.entries.map((entry) => ({
      name: entry.name, path: entry.path, is_dir: entry.isDir,
      language: entry.language, tracked: entry.tracked,
    })),
    timestamp: response.timestamp,
  };
}

export async function deletePath(
  request: DeletePathRequest,
  repoId?: string
): Promise<DeletePathResponse> {
  const intent = await issueMutationIntentForOperation("repo.files.delete", repoId);
  const response = await repoClient.deletePath(create(DeletePathRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId, path: request.path,
  }));
  return {
    success: response.success, path: response.path, is_dir: response.isDir,
    error: response.error, timestamp: response.timestamp,
  };
}

export async function saveFileContent(
  request: SaveFileContentRequest,
  repoId?: string
): Promise<SaveFileContentResponse> {
  const intent = await issueMutationIntentForOperation("repo.files.content", repoId);
  try {
    const response = await repoClient.saveFileContent(create(SaveFileContentRequestSchema, {
      repositoryId: intent.repositoryId, intentId: intent.intentId, path: request.path,
      content: request.content, expectedHash: request.expected_hash ?? "",
    }));
    return {
      success: response.success, path: response.path, content_hash: response.contentHash,
      bytes_written: response.bytesWritten, timestamp: response.timestamp,
    };
  } catch (error) {
    const connectError = ConnectError.from(error);
    if (connectError.code === Code.Aborted) {
      throw new FileContentConflictError(
        connectError.message || "File changed on disk",
        request.path,
        connectError.metadata.get("X-GCT-Current-Hash") ?? "",
      );
    }
    throw error;
  }
}

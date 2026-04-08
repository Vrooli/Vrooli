// ============================================================================
// Agent Manager, Auditor & Review Hooks
// ============================================================================

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "./hooks-query-keys";
import {
  fetchScenarios,
  fetchScenarioEnvelope,
  fetchAgentProfiles,
  createAgentRun,
  fetchAgentRuns,
  fetchAgentRun,
  fetchAgentRunEvents,
  fetchAgentRunDiff,
  continueAgentRun,
  approveAgentRun,
  rejectAgentRun,
  stopAgentRun,
  startAuditorCheck,
  pollAuditorJob,
  fetchAuditorRules,
  applyAuditorFix,
  fetchAuditorViolations,
  fetchReviewSummary,
  triggerReviewRun,
  fetchReviewJobStatus,
  ACTIVE_STATUSES,
  RUN_STATUS,
} from "./api";
import type {
  ScenarioInfo,
  ScenarioEnvelopeData,
  AgentProfileListResponse,
  AgentRunRequest,
  AgentRunCreateResponse,
  AgentRun,
  AgentRunListResponse,
  AgentRunEventsResponse,
  AgentRunDiffResponse,
  AgentContinueRequest,
  AgentContinueResponse,
  AgentApproveRequest,
  AgentApproveResponse,
  AgentRejectRequest,
  AgentRejectResponse,
  AgentStopResponse,
  AuditorCheckJobResponse,
  AuditorJobStatus,
  AuditorRulesListResponse,
  AuditorFixRequest,
  AuditorFixResponse,
  AuditorViolation,
  ReviewSummaryResponse,
  ReviewJobStatus,
} from "./api";

// ── Agent Manager ──────────────────────────────────────────────────────

function agentPollingInterval(status?: string): number | false {
  if (!status) return false;
  if ((ACTIVE_STATUSES as readonly string[]).includes(status)) return 2_000;
  if (status === RUN_STATUS.NEEDS_REVIEW) return 5_000;
  return false;
}

export function useScenarios(enabled = true) {
  return useQuery<ScenarioInfo[], Error>({
    queryKey: queryKeys.scenarios,
    queryFn: fetchScenarios,
    enabled,
    staleTime: 30_000,
  });
}

export function useScenarioEnvelope(slug: string, enabled = true) {
  return useQuery<ScenarioEnvelopeData, Error>({
    queryKey: ["scenario-envelope", slug],
    queryFn: () => fetchScenarioEnvelope(slug),
    enabled,
    staleTime: 60_000,
  });
}

export function useAgentProfiles(enabled = true) {
  return useQuery<AgentProfileListResponse, Error>({
    queryKey: queryKeys.agentProfiles,
    queryFn: () => fetchAgentProfiles(),
    enabled,
    staleTime: 60_000,
  });
}

export function useAgentRuns(slug: string, enabled = true, repoId?: string | null) {
  return useQuery<AgentRunListResponse, Error>({
    queryKey: queryKeys.agentRuns(slug, repoId),
    queryFn: () => fetchAgentRuns(slug, 5, repoId ?? undefined),
    enabled: enabled && Boolean(slug),
    refetchInterval: 15_000,
  });
}

export function useAgentRun(runId: string | null, enabled = true, repoId?: string | null) {
  return useQuery<AgentRun, Error>({
    queryKey: queryKeys.agentRun(runId ?? "", repoId),
    queryFn: () => fetchAgentRun(runId as string, repoId ?? undefined),
    enabled: enabled && Boolean(runId),
    refetchInterval: (query) => agentPollingInterval(query.state.data?.status),
  });
}

export function useAgentRunEvents(
  runId: string | null,
  afterSequence: number,
  enabled = true,
  repoId?: string | null,
  status?: string,
) {
  return useQuery<AgentRunEventsResponse, Error>({
    queryKey: [...queryKeys.agentRunEvents(runId ?? "", repoId), afterSequence],
    queryFn: () => fetchAgentRunEvents(runId as string, afterSequence, repoId ?? undefined),
    enabled: enabled && Boolean(runId),
    refetchInterval: agentPollingInterval(status),
  });
}

export function useAgentRunDiff(runId: string | null, enabled = true, repoId?: string | null) {
  return useQuery<AgentRunDiffResponse, Error>({
    queryKey: queryKeys.agentRunDiff(runId ?? "", repoId),
    queryFn: () => fetchAgentRunDiff(runId as string, repoId ?? undefined),
    enabled: enabled && Boolean(runId),
    staleTime: 10_000,
  });
}

export function useCreateAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentRunCreateResponse, Error, AgentRunRequest>({
    mutationFn: (request) => createAgentRun(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

export function useContinueAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentContinueResponse, Error, { runId: string; request: AgentContinueRequest }>({
    mutationFn: ({ runId, request }) => continueAgentRun(runId, request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

export function useApproveAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentApproveResponse, Error, { runId: string; request: AgentApproveRequest }>({
    mutationFn: ({ runId, request }) => approveAgentRun(runId, request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "status"] });
    },
  });
}

export function useRejectAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentRejectResponse, Error, { runId: string; request: AgentRejectRequest }>({
    mutationFn: ({ runId, request }) => rejectAgentRun(runId, request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

export function useStopAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentStopResponse, Error, string>({
    mutationFn: (runId) => stopAgentRun(runId, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

// ── Auditor ────────────────────────────────────────────────────────────

export function useAuditorRules(enabled = true, repoId?: string | null) {
  return useQuery<AuditorRulesListResponse, Error>({
    queryKey: queryKeys.rulesList(repoId),
    queryFn: () => fetchAuditorRules(repoId ?? undefined),
    enabled,
    staleTime: 60_000,
  });
}

export function useStartAuditorCheck(repoId?: string | null) {
  return useMutation<AuditorCheckJobResponse, Error, { scenarioName: string; checkType?: string }>({
    mutationFn: ({ scenarioName, checkType }) =>
      startAuditorCheck(scenarioName, checkType, repoId ?? undefined),
  });
}

export function useAuditorJobStatus(jobId: string | null, repoId?: string | null) {
  return useQuery<AuditorJobStatus, Error>({
    queryKey: queryKeys.rulesJob(jobId ?? "", repoId),
    queryFn: () => pollAuditorJob(jobId as string, repoId ?? undefined),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!status || status === "completed" || status === "failed") return false;
      return 2_000;
    },
  });
}

export function useApplyAuditorFix(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AuditorFixResponse, Error, AuditorFixRequest>({
    mutationFn: (request) => applyAuditorFix(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repo", "rules-run"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "rules-violations"] });
    },
  });
}

export function useAuditorViolations(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<AuditorViolation[], Error>({
    queryKey: queryKeys.rulesViolations(scenarioName, repoId),
    queryFn: () => fetchAuditorViolations(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    staleTime: 30_000,
  });
}

// ── Review ─────────────────────────────────────────────────────────────

export function useReviewSummary(scenarioName: string, repoId?: string | null) {
  return useQuery<ReviewSummaryResponse, Error>({
    queryKey: queryKeys.reviewSummary(scenarioName, repoId),
    queryFn: () => fetchReviewSummary(scenarioName, repoId ?? undefined),
    enabled: Boolean(scenarioName),
    refetchInterval: 30_000,
  });
}

export function useTriggerReviewRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<{ jobId: string }, Error, { scenarioName: string; checks?: string[] }>({
    mutationFn: (req) => triggerReviewRun(req, repoId ?? undefined),
    onSuccess: (_data, req) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.reviewSummary(req.scenarioName, repoId),
      });
    },
  });
}

export function useReviewJobStatus(jobId: string | null, repoId?: string | null) {
  return useQuery<ReviewJobStatus, Error>({
    queryKey: queryKeys.reviewJob(jobId ?? "", repoId),
    queryFn: () => fetchReviewJobStatus(jobId as string, repoId ?? undefined),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!status || status === "completed" || status === "failed") return false;
      return 2_000;
    },
  });
}

import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  useAgentRun,
  useAgentRunEvents,
  useApplyAuditorFix,
  useApproveAgentRun,
  useCreateAgentRun,
  useReviewJobStatus,
  useTriggerReviewRun,
} from "./hooks-agent";
import { queryKeys } from "./hooks-query-keys";
import { renderHookWithQueryClient } from "../test-utils";

const mockFetchScenarios = vi.fn();
const mockFetchScenarioEnvelope = vi.fn();
const mockFetchAgentProfiles = vi.fn();
const mockCreateAgentRun = vi.fn();
const mockFetchAgentRuns = vi.fn();
const mockFetchAgentRun = vi.fn();
const mockFetchAgentRunEvents = vi.fn();
const mockFetchAgentRunDiff = vi.fn();
const mockContinueAgentRun = vi.fn();
const mockApproveAgentRun = vi.fn();
const mockRejectAgentRun = vi.fn();
const mockStopAgentRun = vi.fn();
const mockStartAuditorCheck = vi.fn();
const mockPollAuditorJob = vi.fn();
const mockFetchAuditorRules = vi.fn();
const mockApplyAuditorFix = vi.fn();
const mockFetchAuditorViolations = vi.fn();
const mockFetchReviewSummary = vi.fn();
const mockTriggerReviewRun = vi.fn();
const mockFetchReviewJobStatus = vi.fn();

vi.mock("./api", () => ({
  ACTIVE_STATUSES: ["queued", "running"],
  RUN_STATUS: { NEEDS_REVIEW: "needs_review" },
  fetchScenarios: (...args: unknown[]) => mockFetchScenarios(...args),
  fetchScenarioEnvelope: (...args: unknown[]) => mockFetchScenarioEnvelope(...args),
  fetchAgentProfiles: (...args: unknown[]) => mockFetchAgentProfiles(...args),
  createAgentRun: (...args: unknown[]) => mockCreateAgentRun(...args),
  fetchAgentRuns: (...args: unknown[]) => mockFetchAgentRuns(...args),
  fetchAgentRun: (...args: unknown[]) => mockFetchAgentRun(...args),
  fetchAgentRunEvents: (...args: unknown[]) => mockFetchAgentRunEvents(...args),
  fetchAgentRunDiff: (...args: unknown[]) => mockFetchAgentRunDiff(...args),
  continueAgentRun: (...args: unknown[]) => mockContinueAgentRun(...args),
  approveAgentRun: (...args: unknown[]) => mockApproveAgentRun(...args),
  rejectAgentRun: (...args: unknown[]) => mockRejectAgentRun(...args),
  stopAgentRun: (...args: unknown[]) => mockStopAgentRun(...args),
  startAuditorCheck: (...args: unknown[]) => mockStartAuditorCheck(...args),
  pollAuditorJob: (...args: unknown[]) => mockPollAuditorJob(...args),
  fetchAuditorRules: (...args: unknown[]) => mockFetchAuditorRules(...args),
  applyAuditorFix: (...args: unknown[]) => mockApplyAuditorFix(...args),
  fetchAuditorViolations: (...args: unknown[]) => mockFetchAuditorViolations(...args),
  fetchReviewSummary: (...args: unknown[]) => mockFetchReviewSummary(...args),
  triggerReviewRun: (...args: unknown[]) => mockTriggerReviewRun(...args),
  fetchReviewJobStatus: (...args: unknown[]) => mockFetchReviewJobStatus(...args),
}));

describe("agent, auditor, and review hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCreateAgentRun.mockResolvedValue({ runId: "run-1" });
    mockFetchAgentRun.mockResolvedValue({ id: "run-1", status: "running" });
    mockFetchAgentRunEvents.mockResolvedValue({ events: [] });
    mockApproveAgentRun.mockResolvedValue({ success: true });
    mockApplyAuditorFix.mockResolvedValue({ success: true });
    mockTriggerReviewRun.mockResolvedValue({ jobId: "review-1" });
    mockFetchReviewJobStatus.mockResolvedValue({ jobId: "review-1", status: "running" });
  });

  it("does not fetch an agent run until a run id is available", () => {
    renderHookWithQueryClient(() => useAgentRun(null, true, "repo-1"));

    expect(mockFetchAgentRun).not.toHaveBeenCalled();
  });

  it("fetches agent run events with cursor and repo context", async () => {
    renderHookWithQueryClient(() => useAgentRunEvents("run-1", 12, true, "repo-1", "running"));

    await waitFor(() => {
      expect(mockFetchAgentRunEvents).toHaveBeenCalledWith("run-1", 12, "repo-1");
    });
  });

  it("invalidates agent run lists after creating a run", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useCreateAgentRun("repo-2"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const request = {
      scenarioSlug: "git-control-tower",
      prompt: "Review the current change set",
    };

    await act(async () => {
      await result.current.mutateAsync(request);
    });

    await waitFor(() => {
      expect(mockCreateAgentRun).toHaveBeenCalledWith(request, "repo-2");
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["agent", "runs"] });
    });
  });

  it("approving an agent run refreshes agent runs and repo status", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useApproveAgentRun("repo-3"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const request = { actor: "tester", commitMsg: "Apply run changes" };

    await act(async () => {
      await result.current.mutateAsync({ runId: "run-7", request });
    });

    await waitFor(() => {
      expect(mockApproveAgentRun).toHaveBeenCalledWith("run-7", request, "repo-3");
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["agent", "runs"] });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["repo", "status"] });
    });
  });

  it("applying an auditor fix refreshes rule job and violation caches", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useApplyAuditorFix("repo-4"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const request = {
      scenario_names: ["git-control-tower"],
      rule_ids: ["ui-a11y-v1"],
      dry_run: true,
    };

    await act(async () => {
      await result.current.mutateAsync(request);
    });

    await waitFor(() => {
      expect(mockApplyAuditorFix).toHaveBeenCalledWith(request, "repo-4");
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["repo", "rules-run"] });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["repo", "rules-violations"] });
    });
  });

  it("triggering a review run refreshes the scenario review summary", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useTriggerReviewRun("repo-5"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const request = { scenarioName: "git-control-tower", checks: ["tests", "lint"] };

    await act(async () => {
      await result.current.mutateAsync(request);
    });

    await waitFor(() => {
      expect(mockTriggerReviewRun).toHaveBeenCalledWith(request, "repo-5");
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.reviewSummary("git-control-tower", "repo-5"),
      });
    });
  });

  it("does not poll review job status until a job id exists", () => {
    renderHookWithQueryClient(() => useReviewJobStatus(null, "repo-6"));

    expect(mockFetchReviewJobStatus).not.toHaveBeenCalled();
  });
});

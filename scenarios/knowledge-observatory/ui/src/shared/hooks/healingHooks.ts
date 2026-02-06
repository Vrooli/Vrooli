// DOC: docs/reference/api-endpoints.md#documentation-healing
import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { DocAutoFixResponse, DocHealJob, DocHealRequest } from "../services/documentationApi";
import {
  approveDocHealing,
  autoFixDocs,
  fetchDocHealingJob,
  rejectDocHealing,
  startDocHealing,
} from "../services/documentationApi";
import { recordActivity } from "../lib/activityStore";

const shouldPollJob = (job?: DocHealJob) => {
  if (!job) return false;
  return job.status === "pending" || job.status === "running";
};

type DocHealingActions = {
  startHealing: (payload: DocHealRequest) => Promise<DocHealJob>;
  approve: (actor?: string) => Promise<DocHealJob>;
  reject: (actor?: string, reason?: string) => Promise<DocHealJob>;
  clearJob: () => void;
};

type DocHealingState = {
  job?: DocHealJob;
  jobId: string | null;
  isLoading: boolean;
  isBusy: boolean;
  error: Error | null;
  actions: DocHealingActions;
};

export function useDocHealing(scenarioName: string | null): DocHealingState {
  const [jobId, setJobId] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const lastStatusRef = useRef<string | null>(null);

  useEffect(() => {
    setJobId(null);
  }, [scenarioName]);

  const jobQuery = useQuery({
    queryKey: ["docHealJob", jobId],
    queryFn: () => fetchDocHealingJob(jobId ?? ""),
    enabled: Boolean(jobId),
    refetchInterval: (query) => (shouldPollJob(query.state.data) ? 2000 : false),
  });

  const startMutation = useMutation({
    mutationFn: (payload: DocHealRequest) => startDocHealing(payload),
    onSuccess: (job) => {
      if (job?.job_id) {
        setJobId(job.job_id);
        recordActivity({
          type: "doc-healing",
          title: "Documentation healing job",
          description: job.scenario_name,
          status: "running",
        });
      }
    },
  });

  const approveMutation = useMutation({
    mutationFn: (actor?: string) => approveDocHealing(jobId ?? "", actor),
    onSuccess: (job) => {
      if (jobId) {
        queryClient.setQueryData(["docHealJob", jobId], job);
      }
      recordActivity({
        type: "doc-healing",
        title: "Documentation healing approved",
        description: job?.scenario_name || scenarioName || "",
        status: "completed",
      });
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (payload: { actor?: string; reason?: string }) =>
      rejectDocHealing(jobId ?? "", payload.actor, payload.reason),
    onSuccess: (job) => {
      if (jobId) {
        queryClient.setQueryData(["docHealJob", jobId], job);
      }
      recordActivity({
        type: "doc-healing",
        title: "Documentation healing rejected",
        description: job?.scenario_name || scenarioName || "",
        status: "failed",
      });
    },
  });

  const isBusy = startMutation.isPending || approveMutation.isPending || rejectMutation.isPending;
  const startError = startMutation.error instanceof Error ? startMutation.error : null;
  const approveError = approveMutation.error instanceof Error ? approveMutation.error : null;
  const rejectError = rejectMutation.error instanceof Error ? rejectMutation.error : null;
  const queryError = jobQuery.error instanceof Error ? jobQuery.error : null;
  const error = startError ?? approveError ?? rejectError ?? queryError;

  const actions = useMemo(
    () => ({
      startHealing: (payload: DocHealRequest) => startMutation.mutateAsync(payload),
      approve: (actor?: string) => approveMutation.mutateAsync(actor),
      reject: (actor?: string, reason?: string) => rejectMutation.mutateAsync({ actor, reason }),
      clearJob: () => setJobId(null),
    }),
    [approveMutation, rejectMutation, startMutation]
  );

  useEffect(() => {
    const job = jobQuery.data;
    if (!job?.status || !job.scenario_name) return;
    if (lastStatusRef.current === job.status) return;
    lastStatusRef.current = job.status;
    if (job.status === "needs_review") {
      recordActivity({
        type: "doc-healing",
        title: "Documentation healing ready",
        description: job.scenario_name,
        status: "info",
      });
    }
  }, [jobQuery.data]);

  return {
    job: jobQuery.data,
    jobId,
    isLoading: jobQuery.isLoading,
    isBusy,
    error,
    actions,
  };
}

type DocAutoFixState = {
  result: DocAutoFixResponse | null;
  isLoading: boolean;
  error: Error | null;
  autoFix: (dryRun?: boolean) => Promise<DocAutoFixResponse>;
  clear: () => void;
};

export function useDocAutoFix(scenarioName: string | null): DocAutoFixState {
  const [result, setResult] = useState<DocAutoFixResponse | null>(null);

  useEffect(() => {
    setResult(null);
  }, [scenarioName]);

  const mutation = useMutation({
    mutationFn: (dryRun?: boolean) => autoFixDocs(scenarioName ?? "", dryRun),
    onSuccess: (data) => {
      setResult(data);
      recordActivity({
        type: "doc-autofix",
        title: data.dry_run ? "Doc auto-fix preview" : "Doc auto-fix applied",
        description: `${scenarioName}: ${data.moved.length} moved, ${data.skipped.length} skipped`,
        status: "completed",
      });
    },
  });

  const actions = useMemo(
    () => ({
      autoFix: (dryRun?: boolean) => mutation.mutateAsync(dryRun),
      clear: () => setResult(null),
    }),
    [mutation],
  );

  return {
    result,
    isLoading: mutation.isPending,
    error: mutation.error instanceof Error ? mutation.error : null,
    ...actions,
  };
}

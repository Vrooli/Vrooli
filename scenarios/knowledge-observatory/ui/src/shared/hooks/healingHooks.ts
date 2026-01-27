// DOC: docs/reference/api-endpoints.md#documentation-healing
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { DocHealJob, DocHealRequest } from "../services/documentationApi";
import {
  approveDocHealing,
  fetchDocHealingJob,
  rejectDocHealing,
  startDocHealing,
} from "../services/documentationApi";

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
      }
    },
  });

  const approveMutation = useMutation({
    mutationFn: (actor?: string) => approveDocHealing(jobId ?? "", actor),
    onSuccess: (job) => {
      if (jobId) {
        queryClient.setQueryData(["docHealJob", jobId], job);
      }
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (payload: { actor?: string; reason?: string }) =>
      rejectDocHealing(jobId ?? "", payload.actor, payload.reason),
    onSuccess: (job) => {
      if (jobId) {
        queryClient.setQueryData(["docHealJob", jobId], job);
      }
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

  return {
    job: jobQuery.data,
    jobId,
    isLoading: jobQuery.isLoading,
    isBusy,
    error,
    actions,
  };
}

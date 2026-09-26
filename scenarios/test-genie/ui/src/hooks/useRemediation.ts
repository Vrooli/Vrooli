import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  cancelRemediationJob,
  createRemediationJob,
  fetchAgentRoles,
  fetchRemediationJobs,
  fetchRemediationPlan,
  refreshRemediationAgent,
  recoverRemediationJob,
  retryRemediationJob,
  verifyRemediationJob
} from "../lib/api";

export function useRemediation(scenarioName: string, executionId?: string) {
  const client = useQueryClient();
  const key = ["remediation", scenarioName];
  const plan = useQuery({ queryKey: [...key, "plan", executionId], queryFn: () => executionId ? fetchRemediationPlan(scenarioName, executionId) : Promise.reject(new Error("execution id is required")), enabled: Boolean(scenarioName && executionId) });
  const jobs = useQuery({
    queryKey: [...key, "jobs"],
    queryFn: () => fetchRemediationJobs(scenarioName),
    enabled: Boolean(scenarioName),
    // Active jobs need progress, terminal jobs do not. Consecutive transport
    // failures back off instead of adding blind interval pressure.
    refetchInterval: (query) => {
      if (!query.state.data?.some((job) => ["launch_pending", "running", "verification_running"].includes(job.status))) return false;
      return Math.min(30_000, 3_000 * 2 ** query.state.fetchFailureCount);
    }
  });
  const roles = useQuery({ queryKey: ["agent-roles"], queryFn: fetchAgentRoles });
  const invalidate = () => client.invalidateQueries({ queryKey: key });
  const create = useMutation({ mutationFn: (input: { findingIds: string[]; requirementIds?: string[]; roleRef: string; additionalContext?: string }) => executionId ? createRemediationJob(scenarioName, { sourceExecutionId: executionId, ...input }) : Promise.reject(new Error("execution id is required")), onSuccess: invalidate });
  const cancel = useMutation({ mutationFn: (id: string) => cancelRemediationJob(scenarioName, id), onSuccess: invalidate });
  const refresh = useMutation({ mutationFn: (id: string) => refreshRemediationAgent(scenarioName, id), onSuccess: invalidate });
  const recover = useMutation({ mutationFn: (id: string) => recoverRemediationJob(scenarioName, id), onSuccess: invalidate });
  const retry = useMutation({ mutationFn: (id: string) => retryRemediationJob(scenarioName, id), onSuccess: invalidate });
  const verify = useMutation({ mutationFn: (id: string) => verifyRemediationJob(scenarioName, id), onSuccess: invalidate });
  return { plan, jobs, roles, create, cancel, refresh, recover, retry, verify, activeJob: jobs.data?.find((job) => ["created", "launch_pending", "running", "agent_completed", "verification_running"].includes(job.status)) };
}

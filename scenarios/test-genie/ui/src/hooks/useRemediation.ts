import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  cancelRemediationJob,
  createRemediationJob,
  fetchAgentRoles,
  fetchRemediationJobs,
  fetchRemediationPlan,
  refreshRemediationAgent,
  verifyRemediationJob
} from "../lib/api";

export function useRemediation(scenarioName: string, executionId?: string) {
  const client = useQueryClient();
  const key = ["remediation", scenarioName];
  const plan = useQuery({ queryKey: [...key, "plan", executionId], queryFn: () => fetchRemediationPlan(scenarioName, executionId!), enabled: Boolean(scenarioName && executionId) });
  const jobs = useQuery({ queryKey: [...key, "jobs"], queryFn: () => fetchRemediationJobs(scenarioName), enabled: Boolean(scenarioName), refetchInterval: (query) => query.state.data?.some((job) => ["running", "verification_running"].includes(job.status)) ? 5000 : false });
  const roles = useQuery({ queryKey: ["agent-roles"], queryFn: fetchAgentRoles });
  const invalidate = () => client.invalidateQueries({ queryKey: key });
  const create = useMutation({ mutationFn: (input: { findingIds: string[]; requirementIds?: string[]; roleRef: string; additionalContext?: string }) => createRemediationJob(scenarioName, { sourceExecutionId: executionId!, ...input }), onSuccess: invalidate });
  const cancel = useMutation({ mutationFn: (id: string) => cancelRemediationJob(scenarioName, id), onSuccess: invalidate });
  const refresh = useMutation({ mutationFn: (id: string) => refreshRemediationAgent(scenarioName, id), onSuccess: invalidate });
  const verify = useMutation({ mutationFn: (id: string) => verifyRemediationJob(scenarioName, id), onSuccess: invalidate });
  return { plan, jobs, roles, create, cancel, refresh, verify, activeJob: jobs.data?.find((job) => ["created", "running", "agent_completed", "verification_running"].includes(job.status)) };
}

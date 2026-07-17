import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { nodesClient, type Node } from "../../api/nodes";
import { queueClient, type NodeQueue } from "../../api/queue";
import { pairingClient, type IssuePairingCodeResponse } from "../../api/pairing";
import {
  onboardClient,
  OnboardingState,
  type GetOnboardingResponse,
  type ListOnboardingsResponse,
  type StartOnboardingInput,
} from "../../api/onboard";

/** Canonical react-query key for the owner's fleet node list. */
export const NODES_QUERY_KEY = ["fleet", "nodes"] as const;

/** Canonical react-query key for the fleet-wide live queue overlay. */
export const QUEUE_QUERY_KEY = ["fleet", "queue"] as const;

/**
 * List the owner's fleet nodes, with the live presence overlay the server
 * stamps at read time. Owner-gated; the management surface renders an error
 * (handled by the panel) when the owner token is absent.
 */
export function useNodesQuery() {
  return useQuery({
    queryKey: NODES_QUERY_KEY,
    queryFn: async (): Promise<Node[]> => {
      const resp = await nodesClient.listNodes({});
      return resp.nodes;
    },
  });
}

/**
 * The live per-node scheduler overlay (running/queued counts + entries). Polled
 * on a short interval so the dashboard reflects dispatch without a manual
 * refresh. Returns a `nodeId -> NodeQueue` map for O(1) lookup from each row.
 * A queue error must NOT blank the fleet (presence is the primary signal), so
 * the panel treats this as best-effort: it reads `data` and ignores `error`.
 */
export function useFleetQueueQuery() {
  return useQuery({
    queryKey: QUEUE_QUERY_KEY,
    queryFn: async (): Promise<Map<string, NodeQueue>> => {
      const resp = await queueClient.listQueue({});
      return new Map(resp.nodes.map((n) => [n.nodeId, n]));
    },
    refetchInterval: 5_000,
  });
}

/**
 * Revoke a node — severs it atomically server-side (registry RevokeNode;
 * Phase 2 also destroys credentials + kills its channel). Refreshes the list
 * on success so the row flips to REVOKED immediately.
 */
export function useRevokeNodeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => nodesClient.revokeNode({ id }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: NODES_QUERY_KEY }),
  });
}

/**
 * Mint a single-use pairing code (owner-gated PairingService.IssuePairingCode).
 * The plaintext code + control-plane public key are returned ONCE; the caller
 * surfaces them for out-of-band delivery to the node's bootstrap installer.
 * Refreshes the node list on success so the freshly paired node appears once
 * it dials in.
 */
export function useIssuePairingCodeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string): Promise<IssuePairingCodeResponse> =>
      pairingClient.issuePairingCode({ name }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: NODES_QUERY_KEY }),
  });
}

/**
 * Start a one-shot onboarding op (owner-gated OnboardService.StartOnboarding).
 * The SSH password rides along in the request `input` and is never persisted by
 * this hook — the caller holds it in component state only for the duration of
 * the submit and clears it after. Returns the durable op id the form then polls
 * via `useOnboardingQuery` for live step states. Refreshes the node list on
 * success so the freshly onboarded node appears once it dials in.
 */
export function useStartOnboardingMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: StartOnboardingInput) => onboardClient.startOnboarding(input),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: NODES_QUERY_KEY }),
  });
}

/** react-query key for a single onboarding op's live progress. */
export const ONBOARDING_QUERY_KEY = (opId: string) => ["fleet", "onboarding", opId] as const;
export const FAILED_ONBOARDINGS_QUERY_KEY = ["fleet", "onboardings", "failed"] as const;

/**
 * Failed onboarding attempts are durable operational targets, not fleet nodes:
 * the agent has not paired yet, but the host identity and diagnostics remain
 * available after a reload so an operator can correct the host and retry.
 */
export function useFailedOnboardingsQuery() {
  return useQuery({
    queryKey: FAILED_ONBOARDINGS_QUERY_KEY,
    queryFn: async (): Promise<ListOnboardingsResponse> => onboardClient.listOnboardings({ limit: 50 }),
    select: (response) => response.ops.filter((op) => op.state === OnboardingState.FAILED),
  });
}

/** Permanently remove a failed attempt's local operation history. */
export function useRemoveFailedOnboardingMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => onboardClient.removeFailedOnboarding({ id }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: FAILED_ONBOARDINGS_QUERY_KEY }),
  });
}

const TERMINAL_ONBOARDING_STATES: ReadonlySet<OnboardingState> = new Set([
  OnboardingState.SUCCEEDED,
  OnboardingState.FAILED,
  OnboardingState.CANCELLED,
]);

/** True once an op has reached a terminal state (no more progress will arrive). */
export function isTerminalOnboarding(state: OnboardingState): boolean {
  return TERMINAL_ONBOARDING_STATES.has(state);
}

/**
 * Follow one onboarding op's live step states by re-reading GetOnboarding (op +
 * full persisted event history). Enabled only while an op id is present; polls
 * on a short interval until the op is terminal, then stops (refetchInterval
 * returns false). This is the fleet feature's live-update idiom — the durable
 * server record is the source of truth, so a page reload simply re-attaches.
 */
export function useOnboardingQuery(opId: string | null) {
  return useQuery({
    queryKey: ONBOARDING_QUERY_KEY(opId ?? ""),
    enabled: opId !== null,
    queryFn: async (): Promise<GetOnboardingResponse> => onboardClient.getOnboarding({ id: opId ?? "" }),
    refetchInterval: (query) => {
      const op = query.state.data?.op;
      if (op && isTerminalOnboarding(op.state)) return false;
      return 2_000;
    },
  });
}

export type { Node, NodeQueue };

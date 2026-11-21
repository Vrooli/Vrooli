import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  decideJoinRequest,
  forgetMachine,
  issueJoinCode,
  listFleet,
  setMachineGrant,
  type DecideInput,
  type Fleet,
} from "../api/machines";

/** One query key so every mutation refreshes the same fleet snapshot. */
export const FLEET_QUERY_KEY = ["machines", "fleet"] as const;

/**
 * The fleet is a snapshot. Live connection changes invalidate it through the
 * process-wide event stream; the surface never opens a second stream or polls.
 */
export function useFleet(enabled: boolean) {
  return useQuery<Fleet>({
    queryKey: FLEET_QUERY_KEY,
    queryFn: listFleet,
    enabled,
  });
}

export function useFleetMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: FLEET_QUERY_KEY });

  return {
    issueCode: useMutation({ mutationFn: (label: string) => issueJoinCode(label) }),
    decide: useMutation({ mutationFn: (input: DecideInput) => decideJoinRequest(input), onSuccess: invalidate }),
    setGrant: useMutation({
      mutationFn: ({ machineId, preset }: { machineId: string; preset: string }) => setMachineGrant(machineId, preset),
      onSuccess: invalidate,
    }),
    forget: useMutation({ mutationFn: (machineId: string) => forgetMachine(machineId), onSuccess: invalidate }),
  };
}

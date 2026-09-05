import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { DrillStatus, listDrills, runDrill } from "../api/drills";
import { queryKeys } from "./keys";

export function useDrills(planId = "") {
  return useQuery({
    queryKey: queryKeys.drills(planId),
    queryFn: () => listDrills(planId),
    refetchInterval: (query) => query.state.data?.some((d) => d.status === DrillStatus.REQUESTED || d.status === DrillStatus.RUNNING) ? 3000 : false,
  });
}

export function useRunDrill() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ planId, targetId, destinationId }: { planId: string; targetId?: string; destinationId?: string }) =>
      runDrill(planId, targetId, destinationId, `ui:${planId}:${targetId ?? ""}:${destinationId ?? ""}:${Date.now()}`),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: ["drills"] }); },
  });
}

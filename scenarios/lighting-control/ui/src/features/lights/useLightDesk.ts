import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { lightingApi } from "./api";

export const lightDeskKey = ["lighting-control", "desk"] as const;

export function useLightDesk() {
  const queryClient = useQueryClient();
  const desk = useQuery({
    queryKey: lightDeskKey,
    queryFn: () => lightingApi.readDesk(),
    refetchInterval: 15_000,
  });
  const refresh = () => queryClient.invalidateQueries({ queryKey: lightDeskKey });
  const mutation = useMutation({
    mutationFn: async (action: () => Promise<unknown>) => action(),
    onSuccess: refresh,
  });

  return {
    desk,
    run: (action: () => Promise<unknown>) => mutation.mutateAsync(action),
    busy: mutation.isPending,
    refresh,
  };
}

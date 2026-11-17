/** React Query state for the shared notification preference client. */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { NotificationPreferencesUnavailableError, type NotificationPreferencesClient, type NotificationPreference } from "./notification-preferences.js";

export function notificationPreferencesQueryKey(sourceScenarios?: readonly string[]) {
  return ["notification-catalog", ...(sourceScenarios?.length ? [...sourceScenarios].sort() : [])] as const;
}

export function useNotificationPreferences(
  client: NotificationPreferencesClient,
  sourceScenarios?: readonly string[],
  enabled = true,
) {
  const queryClient = useQueryClient();
  const queryKey = notificationPreferencesQueryKey(sourceScenarios);
  const catalog = useQuery({
    queryKey,
    queryFn: () => client.getCatalog(sourceScenarios),
    enabled,
    staleTime: 30_000,
    retry: (failureCount, error) => !(error instanceof NotificationPreferencesUnavailableError) && failureCount < 2,
  });
  const save = useMutation({
    mutationFn: (preference: NotificationPreference) => client.updatePreference(preference),
    onMutate: async (next) => {
      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<Awaited<ReturnType<typeof client.getCatalog>>>(queryKey);
      queryClient.setQueryData<Awaited<ReturnType<typeof client.getCatalog>>>(queryKey, (current) => current && ({
        ...current,
        preferences: [
          ...current.preferences.filter((preference) =>
            preference.source_scenario !== next.source_scenario ||
            preference.category !== next.category ||
            preference.scope_key !== next.scope_key,
          ),
          next,
        ],
      }));
      return { previous };
    },
    onError: (_error, _next, context) => {
      if (context?.previous) queryClient.setQueryData(queryKey, context.previous);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey }),
  });
  return { catalog, save };
}

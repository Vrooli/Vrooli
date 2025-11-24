import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { PromptFileInfo, PromptFile } from '@/types/api';

export function usePromptFiles() {
  return useQuery<PromptFileInfo[]>({
    queryKey: queryKeys.prompts.list(),
    queryFn: () => api.listPromptFiles(),
  });
}

export function usePromptFile(id?: string) {
  return useQuery<PromptFile>({
    queryKey: queryKeys.prompts.file(id || 'none'),
    queryFn: () => api.getPromptFile(id as string),
    enabled: !!id,
  });
}

export function useSavePromptFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, content }: { id: string; content: string }) => api.updatePromptFile(id, content),
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.prompts.file(data.id), data);
    },
  });
}

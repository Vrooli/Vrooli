/**
 * Hooks for prompt testing functionality
 */

import { useMutation } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { TaskType, OperationType, Priority } from '@/types/api';

export interface PromptPreviewInput {
  type: TaskType;
  operation: OperationType;
  title: string;
  priority: Priority;
  notes?: string;
}

export interface PromptPreviewResult {
  prompt: string;
  token_count?: number;
  sections?: Array<{
    name: string;
    token_count?: number;
  }>;
}

export function usePromptPreview() {
  return useMutation({
    mutationFn: (data: PromptPreviewInput) => api.previewPrompt(data),
  });
}

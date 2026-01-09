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
  target?: string;
  targets?: string[];
  auto_steer_profile_id?: string;
  auto_steer_phase_index?: number;
}

export interface PromptPreviewResult {
  prompt: string;
  token_count?: number;
  sections?: Record<string, unknown>;
}

export function usePromptPreview() {
  return useMutation({
    mutationFn: (data: PromptPreviewInput) => api.previewPrompt(data),
  });
}

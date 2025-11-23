/**
 * Hooks for recycler functionality
 */

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { RecyclerSettings } from '@/types/api';

export function useRecyclerSettings() {
  // In a real implementation, this would come from the settings query
  // For now, using local state as a placeholder
  const [settings, setSettings] = useState<RecyclerSettings>({
    enabled_for: 'off',
    recycle_interval: 300,
    model_provider: 'ollama',
    model_name: 'llama3.2:1b',
    completion_threshold: 7,
    failure_threshold: 4,
  });

  const updateSetting = <K extends keyof RecyclerSettings>(
    key: K,
    value: RecyclerSettings[K]
  ) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  };

  return { settings, updateSetting };
}

export interface RecyclerTestInput {
  output: string;
}

export interface RecyclerTestResult {
  should_recycle: boolean;
  completion_score: number;
  confidence?: number;
  reasoning?: string;
}

export function useRecyclerTest() {
  return useMutation({
    mutationFn: (data: RecyclerTestInput) => api.testRecyclerOutput(data),
  });
}

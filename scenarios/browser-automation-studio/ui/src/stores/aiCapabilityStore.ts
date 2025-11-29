import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import { useSettingsStore } from './settingsStore';

export interface AICapability {
  available: boolean;
  reason: 'has_credits' | 'has_api_key' | 'disabled' | 'no_credits' | 'checking' | 'error';
  creditsRemaining?: number;
  apiKeyConfigured?: boolean;
  lastChecked?: Date;
}

interface AICapabilityStore {
  capability: AICapability;
  isChecking: boolean;
  checkCapability: () => Promise<void>;
  setCapability: (capability: AICapability) => void;
}

const DEFAULT_CAPABILITY: AICapability = {
  available: false,
  reason: 'checking',
};

export const useAICapabilityStore = create<AICapabilityStore>((set, get) => ({
  capability: DEFAULT_CAPABILITY,
  isChecking: false,

  checkCapability: async () => {
    if (get().isChecking) return;
    set({ isChecking: true });

    try {
      // First check local API key settings
      const settings = useSettingsStore.getState();
      const hasLocalApiKey = Boolean(
        settings.apiKeys.openaiApiKey ||
        settings.apiKeys.anthropicApiKey ||
        settings.apiKeys.customApiEndpoint
      );

      if (hasLocalApiKey) {
        set({
          capability: {
            available: true,
            reason: 'has_api_key',
            apiKeyConfigured: true,
            lastChecked: new Date(),
          },
          isChecking: false,
        });
        return;
      }

      // Check server-side AI availability (for subscription credits)
      try {
        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/ai/status`, {
          method: 'GET',
          headers: { 'Content-Type': 'application/json' },
        });

        if (response.ok) {
          const data = await response.json();
          const available = Boolean(data.available);
          const credits = typeof data.credits === 'number' ? data.credits : undefined;

          set({
            capability: {
              available,
              reason: available ? (credits && credits > 0 ? 'has_credits' : 'has_api_key') : 'no_credits',
              creditsRemaining: credits,
              apiKeyConfigured: Boolean(data.api_key_configured),
              lastChecked: new Date(),
            },
            isChecking: false,
          });
          return;
        }
      } catch {
        // Server endpoint might not exist yet - that's ok
      }

      // No AI capability found
      set({
        capability: {
          available: false,
          reason: 'no_credits',
          lastChecked: new Date(),
        },
        isChecking: false,
      });
    } catch (error) {
      logger.error('Failed to check AI capability', { component: 'AICapabilityStore', action: 'checkCapability' }, error);
      set({
        capability: {
          available: false,
          reason: 'error',
          lastChecked: new Date(),
        },
        isChecking: false,
      });
    }
  },

  setCapability: (capability) => {
    set({ capability });
  },
}));

// Hook to get AI availability with automatic checking
export const useAICapability = () => {
  const { capability, isChecking, checkCapability } = useAICapabilityStore();

  // Auto-check on first use if not checked recently (within 5 minutes)
  if (
    !isChecking &&
    (!capability.lastChecked || Date.now() - capability.lastChecked.getTime() > 5 * 60 * 1000)
  ) {
    void checkCapability();
  }

  return {
    ...capability,
    isChecking,
    refresh: checkCapability,
  };
};

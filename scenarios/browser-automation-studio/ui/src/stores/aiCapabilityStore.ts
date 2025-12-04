import { create } from 'zustand';
import serviceDefinition from '../../../.vrooli/service.json';
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

type ServiceHealthConfig = {
  lifecycle?: {
    health?: {
      endpoints?: Record<string, unknown>;
    };
  };
};

const SERVICE_HEALTH_ENDPOINTS: Record<string, string> = (() => {
  const raw = (serviceDefinition as ServiceHealthConfig)?.lifecycle?.health?.endpoints ?? {};
  return Object.entries(raw).reduce<Record<string, string>>((acc, [key, value]) => {
    if (typeof value === 'string' && value.trim().length > 0) {
      acc[key] = value.trim();
    }
    return acc;
  }, {});
})();

const getPreferredAiStatusPath = (): string | undefined => {
  if (SERVICE_HEALTH_ENDPOINTS.ai) {
    return SERVICE_HEALTH_ENDPOINTS.ai;
  }
  if (SERVICE_HEALTH_ENDPOINTS.api) {
    return SERVICE_HEALTH_ENDPOINTS.api;
  }
  return '/health';
};

const buildAbsoluteEndpointUrl = (apiUrl: string, endpointPath: string): string => {
  if (/^https?:\/\//i.test(endpointPath)) {
    return endpointPath;
  }

  const normalizedPath = endpointPath.startsWith('/') ? endpointPath : `/${endpointPath}`;

  try {
    const parsed = new URL(apiUrl);
    return `${parsed.protocol}//${parsed.host}${normalizedPath}`;
  } catch {
    const stripped = apiUrl.replace(/\/api\/v1\/?$/, '');
    return `${stripped}${normalizedPath}`;
  }
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

      // Check server-side health/AI capability endpoint if declared
      try {
        const statusPath = getPreferredAiStatusPath();
        if (statusPath) {
          const config = await getConfig();
          const statusUrl = buildAbsoluteEndpointUrl(config.API_URL, statusPath);
          const response = await fetch(statusUrl, {
            method: 'GET',
            headers: { 'Content-Type': 'application/json' },
          });

          if (response.ok) {
            let payload: Record<string, unknown> = {};
            try {
              payload = await response.json();
            } catch {
              // Some health endpoints return plain text; ignore parse errors
            }

            const available = typeof payload.available === 'boolean' ? payload.available : true;
            const credits = typeof payload.credits === 'number' ? payload.credits : undefined;
            const apiKeyConfigured =
              typeof payload.api_key_configured === 'boolean' ? payload.api_key_configured : available;

            set({
              capability: {
                available,
                reason: available ? (credits && credits > 0 ? 'has_credits' : 'has_api_key') : 'no_credits',
                creditsRemaining: credits,
                apiKeyConfigured,
                lastChecked: new Date(),
              },
              isChecking: false,
            });
            return;
          }
        }
      } catch {
        // Endpoint missing or offline; fall through to default capability
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

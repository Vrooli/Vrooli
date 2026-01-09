import { useEffect } from 'react';
import { create } from 'zustand';
import serviceDefinition from '../../../.vrooli/service.json';
import { getConfig } from '../config';
import { logger } from '../utils/logger';

export interface AICapability {
  available: boolean;
  reason: 'has_credits' | 'has_api_key' | 'disabled' | 'no_credits' | 'no_tier_access' | 'checking' | 'error';
  creditsRemaining?: number;
  creditsLimit?: number;
  creditsUsed?: number;
  apiKeyConfigured?: boolean;
  resetDate?: string;
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
      // First check local OpenRouter API key
      let hasLocalApiKey = false;
      try {
        const { useSettingsStore } = await import('./settingsStore');
        const settings = useSettingsStore.getState();
        hasLocalApiKey = Boolean(settings.apiKeys.openrouterApiKey);
      } catch {
        // settings store unavailable; fall through to entitlement checks
      }

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

      // Check entitlement-based AI credits
      try {
        const { useEntitlementStore } = await import('./entitlementStore');
        const entitlementState = useEntitlementStore.getState();

        // Fetch fresh status if needed
        if (!entitlementState.status) {
          await entitlementState.fetchStatus();
        }

        const status = useEntitlementStore.getState().status;

        if (status) {
          // If entitlements are disabled, AI is available
          if (!status.entitlements_enabled) {
            set({
              capability: {
                available: true,
                reason: 'disabled',
                lastChecked: new Date(),
              },
              isChecking: false,
            });
            return;
          }

          // Check if tier has AI access (limit != 0)
          const aiCreditsLimit = status.ai_credits_limit ?? 0;
          const aiCreditsRemaining = status.ai_credits_remaining ?? 0;
          const aiCreditsUsed = status.ai_credits_used ?? 0;

          if (aiCreditsLimit === 0) {
            // No AI access for this tier
            set({
              capability: {
                available: false,
                reason: 'no_tier_access',
                creditsLimit: 0,
                creditsRemaining: 0,
                creditsUsed: 0,
                lastChecked: new Date(),
              },
              isChecking: false,
            });
            return;
          }

          // Unlimited credits
          if (aiCreditsLimit < 0) {
            set({
              capability: {
                available: true,
                reason: 'has_credits',
                creditsLimit: -1,
                creditsRemaining: -1,
                creditsUsed: aiCreditsUsed,
                resetDate: status.ai_reset_date,
                lastChecked: new Date(),
              },
              isChecking: false,
            });
            return;
          }

          // Check if user has remaining credits
          if (aiCreditsRemaining > 0) {
            set({
              capability: {
                available: true,
                reason: 'has_credits',
                creditsLimit: aiCreditsLimit,
                creditsRemaining: aiCreditsRemaining,
                creditsUsed: aiCreditsUsed,
                resetDate: status.ai_reset_date,
                lastChecked: new Date(),
              },
              isChecking: false,
            });
            return;
          }

          // No credits remaining
          set({
            capability: {
              available: false,
              reason: 'no_credits',
              creditsLimit: aiCreditsLimit,
              creditsRemaining: 0,
              creditsUsed: aiCreditsUsed,
              resetDate: status.ai_reset_date,
              lastChecked: new Date(),
            },
            isChecking: false,
          });
          return;
        }
      } catch {
        // Entitlement store unavailable; fall through to server-side checks
      }

      // Check server-side health/AI capability endpoint as fallback
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

  useEffect(() => {
    const isAutomatedRun =
      typeof navigator !== "undefined" &&
      (navigator.webdriver === true ||
        /lighthouse/i.test(navigator.userAgent) ||
        /HeadlessChrome/i.test(navigator.userAgent));

    if (isAutomatedRun) {
      return;
    }

    const shouldCheck =
      !isChecking &&
      (!capability.lastChecked ||
        Date.now() - capability.lastChecked.getTime() > 5 * 60 * 1000);

    if (shouldCheck) {
      void checkCapability();
    }
  }, [capability.lastChecked, checkCapability, isChecking]);

  return {
    ...capability,
    isChecking,
    refresh: checkCapability,
  };
};

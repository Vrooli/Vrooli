import { useState } from 'react';
import { Key, AlertTriangle, Eye, EyeOff } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import type { ApiKeySettings } from '@stores/settingsStore';
import { SettingSection } from './shared';

export function ApiKeysSection() {
  const { apiKeys, setApiKey } = useSettingsStore();
  const [showApiKeys, setShowApiKeys] = useState<Record<keyof ApiKeySettings, boolean>>({
    browserlessApiKey: false,
    openaiApiKey: false,
    anthropicApiKey: false,
    customApiEndpoint: false,
  });

  const toggleApiKeyVisibility = (key: keyof ApiKeySettings) => {
    setShowApiKeys(prev => ({ ...prev, [key]: !prev[key] }));
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-6">
        <Key size={24} className="text-amber-400" />
        <div>
          <h2 className="text-lg font-semibold text-surface">API Keys & Integrations</h2>
          <p className="text-sm text-gray-400">Override default API keys with your own</p>
        </div>
      </div>

      <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-4 mb-6">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-amber-200 font-medium">Security Notice</p>
            <p className="text-xs text-amber-300/80 mt-1">
              API keys are stored locally in your browser. Never share your API keys with others.
              Keys are used to authenticate with external services.
            </p>
          </div>
        </div>
      </div>

      <SettingSection title="Browser Service" tooltip="API key for the headless browser service.">
        <div className="space-y-3">
          <div>
            <label className="text-sm text-gray-300 block mb-2">Browserless API Key</label>
            <div className="relative">
              <input
                type={showApiKeys.browserlessApiKey ? 'text' : 'password'}
                value={apiKeys.browserlessApiKey}
                onChange={(e) => setApiKey('browserlessApiKey', e.target.value)}
                placeholder="Enter your Browserless API key"
                className="w-full px-4 py-3 pr-12 bg-gray-800 border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent"
              />
              <button
                type="button"
                onClick={() => toggleApiKeyVisibility('browserlessApiKey')}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label={showApiKeys.browserlessApiKey ? 'Hide API key' : 'Show API key'}
              >
                {showApiKeys.browserlessApiKey ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Leave empty to use the default shared key
            </p>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="AI Services" tooltip="API keys for AI-powered features.">
        <div className="space-y-5">
          <div>
            <label className="text-sm text-gray-300 block mb-2">OpenAI API Key</label>
            <div className="relative">
              <input
                type={showApiKeys.openaiApiKey ? 'text' : 'password'}
                value={apiKeys.openaiApiKey}
                onChange={(e) => setApiKey('openaiApiKey', e.target.value)}
                placeholder="sk-..."
                className="w-full px-4 py-3 pr-12 bg-gray-800 border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent"
              />
              <button
                type="button"
                onClick={() => toggleApiKeyVisibility('openaiApiKey')}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label={showApiKeys.openaiApiKey ? 'Hide API key' : 'Show API key'}
              >
                {showApiKeys.openaiApiKey ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Used for AI workflow generation
            </p>
          </div>
          <div>
            <label className="text-sm text-gray-300 block mb-2">Anthropic API Key</label>
            <div className="relative">
              <input
                type={showApiKeys.anthropicApiKey ? 'text' : 'password'}
                value={apiKeys.anthropicApiKey}
                onChange={(e) => setApiKey('anthropicApiKey', e.target.value)}
                placeholder="sk-ant-..."
                className="w-full px-4 py-3 pr-12 bg-gray-800 border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent"
              />
              <button
                type="button"
                onClick={() => toggleApiKeyVisibility('anthropicApiKey')}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label={showApiKeys.anthropicApiKey ? 'Hide API key' : 'Show API key'}
              >
                {showApiKeys.anthropicApiKey ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Alternative AI provider for workflow generation
            </p>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Custom Endpoint" tooltip="Override the default API endpoint.">
        <div>
          <label className="text-sm text-gray-300 block mb-2">Custom API Endpoint</label>
          <input
            type="text"
            value={apiKeys.customApiEndpoint}
            onChange={(e) => setApiKey('customApiEndpoint', e.target.value)}
            placeholder="https://api.example.com"
            className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent"
          />
          <p className="text-xs text-gray-500 mt-2">
            For self-hosted or proxy API endpoints
          </p>
        </div>
      </SettingSection>
    </div>
  );
}

import { useMemo } from 'react';
import type { ProxySettings } from '@/domains/recording/types/types';
import {
  FormFieldGroup,
  FormTextInput,
  FormToggleCard,
  FormGrid,
  validateProxyUrl,
} from '@/components/form';

interface ProxySectionProps {
  proxy: ProxySettings;
  onChange: <K extends keyof ProxySettings>(key: K, value: ProxySettings[K]) => void;
}

export function ProxySection({ proxy, onChange }: ProxySectionProps) {
  const proxyUrlError = useMemo(() => validateProxyUrl(proxy.server), [proxy.server]);

  return (
    <div className="space-y-6">
      {/* Enable Proxy */}
      <FormToggleCard
        checked={proxy.enabled ?? false}
        onChange={(checked) => onChange('enabled', checked)}
        label="Enable Proxy"
        description="Route browser traffic through a proxy server for IP masking and geographic targeting"
      />

      {/* Proxy Configuration (shown when enabled) */}
      {proxy.enabled && (
        <>
          {/* Proxy Server */}
          <FormFieldGroup title="Proxy Server">
            <FormTextInput
              value={proxy.server}
              onChange={(v) => onChange('server', v)}
              label="Server URL"
              placeholder="http://proxy.example.com:8080"
              description="Supports http://, https://, socks4://, and socks5:// protocols"
              error={proxyUrlError}
            />
          </FormFieldGroup>

          {/* Bypass List */}
          <FormFieldGroup title="Bypass List">
            <FormTextInput
              value={proxy.bypass}
              onChange={(v) => onChange('bypass', v)}
              label="Bypass Patterns"
              placeholder="localhost,.internal.com,192.168.*"
              description="Comma-separated domains or IPs to bypass the proxy"
            />
          </FormFieldGroup>

          {/* Authentication */}
          <FormFieldGroup
            title="Authentication"
            description="Optional credentials for authenticated proxy servers"
            className="pt-4 border-t border-gray-200 dark:border-gray-700"
          >
            <FormGrid cols={2}>
              <FormTextInput
                value={proxy.username}
                onChange={(v) => onChange('username', v)}
                label="Username"
                placeholder="proxy_user"
              />
              <FormTextInput
                value={proxy.password}
                onChange={(v) => onChange('password', v)}
                label="Password"
                type="password"
                placeholder="Enter password"
              />
            </FormGrid>
          </FormFieldGroup>

          {/* Info Box */}
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
            <p className="text-xs text-blue-700 dark:text-blue-300">
              <strong>Tip:</strong> For best results, use a proxy location that matches your geolocation and timezone settings to create a consistent browser identity.
            </p>
          </div>
        </>
      )}
    </div>
  );
}

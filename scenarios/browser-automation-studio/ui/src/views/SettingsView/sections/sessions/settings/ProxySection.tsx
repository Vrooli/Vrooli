import type { ProxySettings } from '@/domains/recording/types/types';

interface ProxySectionProps {
  proxy: ProxySettings;
  onChange: <K extends keyof ProxySettings>(key: K, value: ProxySettings[K]) => void;
}

export function ProxySection({ proxy, onChange }: ProxySectionProps) {
  return (
    <div className="space-y-6">
      {/* Enable Proxy */}
      <div>
        <label className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer">
          <input
            type="checkbox"
            checked={proxy.enabled ?? false}
            onChange={(e) => onChange('enabled', e.target.checked)}
            className="mt-0.5 rounded border-gray-300 dark:border-gray-600"
          />
          <div>
            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">Enable Proxy</span>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              Route browser traffic through a proxy server for IP masking and geographic targeting
            </p>
          </div>
        </label>
      </div>

      {/* Proxy Configuration (shown when enabled) */}
      {proxy.enabled && (
        <>
          {/* Proxy Server */}
          <div>
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">Proxy Server</h4>
            <input
              type="text"
              value={proxy.server ?? ''}
              onChange={(e) => onChange('server', e.target.value || undefined)}
              placeholder="http://proxy.example.com:8080"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Supports http://, https://, and socks5:// protocols
            </p>
          </div>

          {/* Bypass List */}
          <div>
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">Bypass List</h4>
            <input
              type="text"
              value={proxy.bypass ?? ''}
              onChange={(e) => onChange('bypass', e.target.value || undefined)}
              placeholder="localhost,.internal.com,192.168.*"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Comma-separated domains or IPs to bypass the proxy
            </p>
          </div>

          {/* Authentication */}
          <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">Authentication</h4>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
              Optional credentials for authenticated proxy servers
            </p>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Username</label>
                <input
                  type="text"
                  value={proxy.username ?? ''}
                  onChange={(e) => onChange('username', e.target.value || undefined)}
                  placeholder="proxy_user"
                  className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Password</label>
                <input
                  type="password"
                  value={proxy.password ?? ''}
                  onChange={(e) => onChange('password', e.target.value || undefined)}
                  placeholder="••••••••"
                  className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
                />
              </div>
            </div>
          </div>

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

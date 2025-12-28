import { useMemo } from 'react';
import { Lock, Shield, AlertTriangle } from 'lucide-react';
import type { StorageStateCookie } from '@/domains/recording';

interface CookiesTableProps {
  cookies: StorageStateCookie[];
}

function isExpired(expires: number): boolean {
  // expires is a Unix timestamp in seconds
  return expires > 0 && expires < Date.now() / 1000;
}

function formatExpiry(expires: number): string {
  if (expires === -1 || expires === 0) {
    return 'Session';
  }
  const date = new Date(expires * 1000);
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function CookiesTable({ cookies }: CookiesTableProps) {
  const sortedCookies = useMemo(
    () => [...cookies].sort((a, b) => a.domain.localeCompare(b.domain) || a.name.localeCompare(b.name)),
    [cookies]
  );

  if (cookies.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        No cookies stored in this session.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-sm">
        <thead>
          <tr className="border-b border-gray-200 dark:border-gray-700">
            <th className="text-left py-2 px-3 font-medium text-gray-700 dark:text-gray-300">Domain</th>
            <th className="text-left py-2 px-3 font-medium text-gray-700 dark:text-gray-300">Name</th>
            <th className="text-left py-2 px-3 font-medium text-gray-700 dark:text-gray-300">Value</th>
            <th className="text-left py-2 px-3 font-medium text-gray-700 dark:text-gray-300">Expires</th>
            <th className="text-left py-2 px-3 font-medium text-gray-700 dark:text-gray-300">Flags</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
          {sortedCookies.map((cookie, index) => {
            const expired = isExpired(cookie.expires);
            return (
              <tr
                key={`${cookie.domain}-${cookie.name}-${index}`}
                className={`hover:bg-gray-50 dark:hover:bg-gray-800/50 ${expired ? 'opacity-60' : ''}`}
              >
                <td className="py-2 px-3 text-gray-600 dark:text-gray-400 font-mono text-xs">
                  {cookie.domain}
                </td>
                <td className="py-2 px-3 text-gray-900 dark:text-gray-100 font-medium">
                  {cookie.name}
                </td>
                <td className="py-2 px-3 max-w-[200px]">
                  {cookie.valueMasked ? (
                    <span className="inline-flex items-center gap-1 text-gray-400 dark:text-gray-500 italic">
                      <Lock size={12} />
                      [HIDDEN]
                    </span>
                  ) : (
                    <span
                      className="block truncate text-gray-600 dark:text-gray-400 font-mono text-xs"
                      title={cookie.value}
                    >
                      {cookie.value}
                    </span>
                  )}
                </td>
                <td className="py-2 px-3 text-gray-600 dark:text-gray-400 text-xs whitespace-nowrap">
                  <span className={expired ? 'line-through text-red-500 dark:text-red-400' : ''}>
                    {formatExpiry(cookie.expires)}
                  </span>
                  {expired && (
                    <span className="ml-1.5 inline-flex items-center gap-0.5 text-red-500 dark:text-red-400">
                      <AlertTriangle size={12} />
                    </span>
                  )}
                </td>
                <td className="py-2 px-3">
                  <div className="flex items-center gap-1.5">
                    {cookie.httpOnly && (
                      <span
                        className="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-medium rounded bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300"
                        title="HttpOnly - not accessible via JavaScript"
                      >
                        <Lock size={10} />
                        HttpOnly
                      </span>
                    )}
                    {cookie.secure && (
                      <span
                        className="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-medium rounded bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300"
                        title="Secure - only sent over HTTPS"
                      >
                        <Shield size={10} />
                        Secure
                      </span>
                    )}
                    {cookie.sameSite !== 'None' && (
                      <span className="px-1.5 py-0.5 text-[10px] font-medium rounded bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400">
                        {cookie.sameSite}
                      </span>
                    )}
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

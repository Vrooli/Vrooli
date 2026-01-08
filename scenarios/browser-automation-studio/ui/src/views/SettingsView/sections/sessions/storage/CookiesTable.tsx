import { useCallback, useMemo, useState } from 'react';
import { Lock, Shield, AlertTriangle, Trash2, ChevronDown, ChevronRight, Globe } from 'lucide-react';
import type { StorageStateCookie } from '@/domains/recording';
import { InlineDeleteConfirmation } from './InlineDeleteConfirmation';

interface CookiesTableProps {
  cookies: StorageStateCookie[];
  deleting?: boolean;
  onDeleteDomain?: (domain: string) => Promise<boolean>;
  onDeleteCookie?: (domain: string, name: string) => Promise<boolean>;
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

interface DomainGroup {
  domain: string;
  cookies: StorageStateCookie[];
}

interface DomainSectionProps {
  group: DomainGroup;
  deleting?: boolean;
  onDeleteDomain?: (domain: string) => Promise<boolean>;
  onDeleteCookie?: (domain: string, name: string) => Promise<boolean>;
}

function DomainSection({ group, deleting, onDeleteDomain, onDeleteCookie }: DomainSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [confirmDelete, setConfirmDelete] = useState<'domain' | string | null>(null);

  const toggleExpanded = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  const handleDeleteDomain = useCallback(async () => {
    if (onDeleteDomain) {
      await onDeleteDomain(group.domain);
    }
    setConfirmDelete(null);
  }, [group.domain, onDeleteDomain]);

  const handleDeleteCookie = useCallback(
    async (cookieName: string) => {
      if (onDeleteCookie) {
        await onDeleteCookie(group.domain, cookieName);
      }
      setConfirmDelete(null);
    },
    [group.domain, onDeleteCookie]
  );

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
      {/* Domain header */}
      <div className="flex items-center bg-gray-50 dark:bg-gray-800/50">
        <button
          onClick={toggleExpanded}
          className="flex-1 flex items-center gap-2 px-4 py-3 hover:bg-gray-100 dark:hover:bg-gray-800 text-left"
        >
          {isExpanded ? (
            <ChevronDown size={16} className="text-gray-500" />
          ) : (
            <ChevronRight size={16} className="text-gray-500" />
          )}
          <Globe size={14} className="text-gray-500" />
          <span className="font-medium text-gray-900 dark:text-gray-100 font-mono text-sm">{group.domain}</span>
          <span className="ml-auto text-xs text-gray-500 dark:text-gray-400">
            {group.cookies.length} cookie{group.cookies.length !== 1 ? 's' : ''}
          </span>
        </button>
        {onDeleteDomain && (
          <button
            onClick={() => setConfirmDelete('domain')}
            disabled={deleting}
            className="p-2 mr-2 text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
            title={`Delete all cookies for ${group.domain}`}
          >
            <Trash2 size={14} />
          </button>
        )}
      </div>

      {/* Cookies table */}
      {isExpanded && (
        <table className="min-w-full text-sm">
          <thead>
            <tr className="border-t border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/30">
              <th className="text-left py-1.5 px-4 font-medium text-gray-600 dark:text-gray-400 text-xs">Name</th>
              <th className="text-left py-1.5 px-3 font-medium text-gray-600 dark:text-gray-400 text-xs">Value</th>
              <th className="text-left py-1.5 px-3 font-medium text-gray-600 dark:text-gray-400 text-xs">Expires</th>
              <th className="text-left py-1.5 px-3 font-medium text-gray-600 dark:text-gray-400 text-xs">Flags</th>
              <th className="w-10" />
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
            {group.cookies.map((cookie, index) => {
              const expired = isExpired(cookie.expires);
              return (
                <tr
                  key={`${cookie.name}-${index}`}
                  className={`hover:bg-gray-50 dark:hover:bg-gray-800/50 ${expired ? 'opacity-60' : ''}`}
                >
                  <td className="py-2 px-4 text-gray-900 dark:text-gray-100 font-medium">{cookie.name}</td>
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
                  <td className="py-2 px-2">
                    {onDeleteCookie && (
                      <button
                        onClick={() => setConfirmDelete(cookie.name)}
                        disabled={deleting}
                        className="p-1.5 text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
                        title={`Delete ${cookie.name}`}
                      >
                        <Trash2 size={12} />
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}

      {/* Delete confirmation inline */}
      {confirmDelete && (
        <InlineDeleteConfirmation
          message={
            confirmDelete === 'domain'
              ? `Delete all ${group.cookies.length} cookies for ${group.domain}?`
              : `Delete cookie "${confirmDelete}"?`
          }
          deleting={deleting}
          onConfirm={() =>
            confirmDelete === 'domain' ? handleDeleteDomain() : handleDeleteCookie(confirmDelete)
          }
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  );
}

export function CookiesTable({ cookies, deleting, onDeleteDomain, onDeleteCookie }: CookiesTableProps) {
  // Group cookies by domain
  const domainGroups = useMemo(() => {
    const groups = new Map<string, StorageStateCookie[]>();
    for (const cookie of cookies) {
      const existing = groups.get(cookie.domain) || [];
      existing.push(cookie);
      groups.set(cookie.domain, existing);
    }
    // Sort domains alphabetically, sort cookies within each domain by name
    return Array.from(groups.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([domain, domainCookies]) => ({
        domain,
        cookies: domainCookies.sort((a, b) => a.name.localeCompare(b.name)),
      }));
  }, [cookies]);

  if (cookies.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        No cookies stored in this session.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {domainGroups.map((group) => (
        <DomainSection
          key={group.domain}
          group={group}
          deleting={deleting}
          onDeleteDomain={onDeleteDomain}
          onDeleteCookie={onDeleteCookie}
        />
      ))}
    </div>
  );
}

/**
 * Domains Page
 *
 * Lists all registered domain scenarios with their status and capabilities.
 *
 * [REQ:LD-DOMAIN-DISCOVER] - Domain registration and discovery
 */
import { useQuery } from "@tanstack/react-query";
import { Heart, RefreshCw, ChevronLeft } from "lucide-react";
import { Link } from "react-router-dom";

import { StatusBadge } from "../components/dashboard";
import { fetchDomains, type Domain } from "../lib/api";
import { formatRelativeTime } from "../lib/format";

export default function DomainsPage() {
  const domainsQuery = useQuery({
    queryKey: ["domains"],
    queryFn: fetchDomains,
    refetchInterval: 60000,
  });

  const activeDomains = domainsQuery.data?.domains?.filter((d: Domain) => d.status === "active") ?? [];
  const inactiveDomains = domainsQuery.data?.domains?.filter((d: Domain) => d.status !== "active") ?? [];

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Link to="/" className="text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Registered Domains</h1>
          <p className="text-slate-400">
            {domainsQuery.data?.count ?? 0} domain{domainsQuery.data?.count !== 1 ? "s" : ""} registered
          </p>
        </div>
        <button
          onClick={() => domainsQuery.refetch()}
          className="ml-auto p-2 rounded-lg bg-white/5 hover:bg-white/10 transition-colors"
          disabled={domainsQuery.isFetching}
        >
          <RefreshCw className={`w-5 h-5 ${domainsQuery.isFetching ? "animate-spin" : ""}`} />
        </button>
      </div>

      {/* Loading state */}
      {domainsQuery.isLoading && (
        <div className="text-slate-500 text-center py-12">Loading domains...</div>
      )}

      {/* Empty state */}
      {!domainsQuery.isLoading && (!domainsQuery.data?.domains || domainsQuery.data.domains.length === 0) && (
        <div className="text-center py-12 rounded-xl border border-white/10 bg-white/5">
          <Heart className="w-16 h-16 mx-auto text-slate-700 mb-4" />
          <h2 className="text-xl font-medium text-slate-300 mb-2">No domains registered</h2>
          <p className="text-slate-500 max-w-md mx-auto">
            Domain scenarios will register themselves when they start.
            Start a health scenario to see it appear here.
          </p>
        </div>
      )}

      {/* Active domains */}
      {activeDomains.length > 0 && (
        <div>
          <h2 className="text-lg font-medium mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-500"></span>
            Active Domains ({activeDomains.length})
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {activeDomains.map((domain: Domain) => (
              <Link
                key={domain.name}
                to={`/domains/${domain.name}`}
                className="block rounded-xl border border-white/10 bg-white/5 p-6 hover:bg-white/10 transition-colors"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="font-medium text-lg">{domain.display_name}</h3>
                    <p className="text-sm text-slate-400">{domain.name}</p>
                  </div>
                  <StatusBadge status={domain.status} />
                </div>
                {domain.description && (
                  <p className="mt-3 text-sm text-slate-400 line-clamp-2">{domain.description}</p>
                )}
                {domain.capabilities && domain.capabilities.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {domain.capabilities.slice(0, 3).map((cap: string) => (
                      <span key={cap} className="px-2 py-0.5 text-xs rounded-full bg-blue-500/10 text-blue-400">
                        {cap}
                      </span>
                    ))}
                    {domain.capabilities.length > 3 && (
                      <span className="px-2 py-0.5 text-xs rounded-full bg-slate-500/10 text-slate-400">
                        +{domain.capabilities.length - 3} more
                      </span>
                    )}
                  </div>
                )}
                <div className="mt-4 text-xs text-slate-500">
                  Updated {formatRelativeTime(domain.updated_at)}
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* Inactive domains */}
      {inactiveDomains.length > 0 && (
        <div>
          <h2 className="text-lg font-medium mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-slate-500"></span>
            Inactive Domains ({inactiveDomains.length})
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {inactiveDomains.map((domain: Domain) => (
              <Link
                key={domain.name}
                to={`/domains/${domain.name}`}
                className="block rounded-xl border border-white/10 bg-white/5 p-6 hover:bg-white/10 transition-colors opacity-60"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="font-medium text-lg">{domain.display_name}</h3>
                    <p className="text-sm text-slate-400">{domain.name}</p>
                  </div>
                  <StatusBadge status={domain.status} />
                </div>
                {domain.description && (
                  <p className="mt-3 text-sm text-slate-400 line-clamp-2">{domain.description}</p>
                )}
                <div className="mt-4 text-xs text-slate-500">
                  Last seen {formatRelativeTime(domain.updated_at)}
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

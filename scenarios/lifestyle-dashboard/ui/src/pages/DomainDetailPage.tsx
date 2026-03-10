/**
 * Domain Detail Page
 *
 * Shows detailed information about a specific domain including events,
 * health status, and capabilities.
 *
 * [REQ:LD-DOMAIN-HEALTH] - Domain health status tracking
 */
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, RefreshCw, AlertCircle, Activity, Heart } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { StatusBadge, EventRow } from "../components/dashboard";
import { fetchDomain, fetchDomainHealth, fetchEvents } from "../lib/api";
import { formatRelativeTime, formatDateTime } from "../lib/format";

export default function DomainDetailPage() {
  const { name } = useParams<{ name: string }>();

  const domainQuery = useQuery({
    queryKey: ["domain", name],
    queryFn: () => fetchDomain(name!),
    enabled: !!name,
    refetchInterval: 30000,
  });

  const healthQuery = useQuery({
    queryKey: ["domain-health", name],
    queryFn: () => fetchDomainHealth(name!),
    enabled: !!name,
    refetchInterval: 60000,
  });

  const eventsQuery = useQuery({
    queryKey: ["events", { domain: name }],
    queryFn: () => fetchEvents({ domain: name, limit: 20 }),
    enabled: !!name,
    refetchInterval: 30000,
  });

  const domain = domainQuery.data;
  const health = healthQuery.data;

  if (domainQuery.isLoading) {
    return (
      <div className="text-center py-12 text-slate-500">
        Loading domain details...
      </div>
    );
  }

  if (domainQuery.error || !domain) {
    return (
      <div className="space-y-6">
        <Link to="/domains" className="flex items-center gap-2 text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
          Back to domains
        </Link>
        <div className="text-center py-12 rounded-xl border border-red-500/30 bg-red-500/10">
          <AlertCircle className="w-16 h-16 mx-auto text-red-400 mb-4" />
          <h2 className="text-xl font-medium text-red-400 mb-2">Domain not found</h2>
          <p className="text-slate-400">
            The domain "{name}" could not be found.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start gap-4">
        <Link to="/domains" className="mt-1 text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">{domain.display_name}</h1>
            <StatusBadge status={domain.status} />
          </div>
          <p className="text-slate-400">{domain.name}</p>
        </div>
        <button
          onClick={() => {
            domainQuery.refetch();
            healthQuery.refetch();
            eventsQuery.refetch();
          }}
          className="p-2 rounded-lg bg-white/5 hover:bg-white/10 transition-colors"
          disabled={domainQuery.isFetching}
        >
          <RefreshCw className={`w-5 h-5 ${domainQuery.isFetching ? "animate-spin" : ""}`} />
        </button>
      </div>

      {/* Description and capabilities */}
      {(domain.description || (domain.capabilities && domain.capabilities.length > 0)) && (
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          {domain.description && (
            <p className="text-slate-300">{domain.description}</p>
          )}
          {domain.capabilities && domain.capabilities.length > 0 && (
            <div className="mt-4">
              <h3 className="text-sm font-medium text-slate-400 mb-2">Capabilities</h3>
              <div className="flex flex-wrap gap-2">
                {domain.capabilities.map((cap: string) => (
                  <span key={cap} className="px-3 py-1 text-sm rounded-full bg-blue-500/10 text-blue-400">
                    {cap}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Info grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Health status */}
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Health Status</h3>
          <div className="flex items-center gap-3">
            <Heart className={`w-8 h-8 ${
              health?.status === "healthy" ? "text-green-400" :
              health?.status === "unhealthy" ? "text-red-400" :
              "text-slate-500"
            }`} />
            <div>
              <p className="font-medium capitalize">{health?.status ?? domain.status}</p>
              {health?.last_check && (
                <p className="text-sm text-slate-500">
                  Checked {formatRelativeTime(health.last_check)}
                </p>
              )}
            </div>
          </div>
          {domain.health_url && (
            <p className="mt-3 text-xs text-slate-500 truncate" title={domain.health_url}>
              {domain.health_url}
            </p>
          )}
        </div>

        {/* Registration info */}
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Registered</h3>
          <p className="font-medium">{formatDateTime(domain.registered_at)}</p>
          <p className="text-sm text-slate-500">
            {formatRelativeTime(domain.registered_at)}
          </p>
        </div>

        {/* Last updated */}
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Last Updated</h3>
          <p className="font-medium">{formatDateTime(domain.updated_at)}</p>
          <p className="text-sm text-slate-500">
            {formatRelativeTime(domain.updated_at)}
          </p>
        </div>
      </div>

      {/* Events from this domain */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <h2 className="text-lg font-medium mb-4">
          Recent Events
          {eventsQuery.data?.count !== undefined && (
            <span className="ml-2 text-sm text-slate-500">
              ({eventsQuery.data.count} total)
            </span>
          )}
        </h2>
        {eventsQuery.isLoading ? (
          <div className="text-slate-500">Loading events...</div>
        ) : eventsQuery.data?.events && eventsQuery.data.events.length > 0 ? (
          <div className="divide-y divide-white/5">
            {eventsQuery.data.events.map((event) => (
              <EventRow key={event.id} event={event} />
            ))}
          </div>
        ) : (
          <div className="text-center py-8">
            <Activity className="w-12 h-12 mx-auto text-slate-700 mb-3" />
            <p className="text-slate-400">No events from this domain</p>
          </div>
        )}
      </div>
    </div>
  );
}

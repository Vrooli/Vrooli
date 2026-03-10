/**
 * Events Page
 *
 * Shows all events with filtering capabilities.
 *
 * [REQ:LD-QUERY-FILTER] - Time-range filters, domain filters
 */
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, RefreshCw, Activity, Filter } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { useState } from "react";

import { EventRow } from "../components/dashboard";
import { fetchEvents, fetchDomains } from "../lib/api";

export default function EventsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedDomain, setSelectedDomain] = useState(searchParams.get("domain") || "");

  const domainsQuery = useQuery({
    queryKey: ["domains"],
    queryFn: fetchDomains,
  });

  const eventsQuery = useQuery({
    queryKey: ["events", { domain: selectedDomain, limit: 50 }],
    queryFn: () => fetchEvents({ domain: selectedDomain || undefined, limit: 50 }),
    refetchInterval: 30000,
  });

  const handleDomainChange = (domain: string) => {
    setSelectedDomain(domain);
    if (domain) {
      setSearchParams({ domain });
    } else {
      setSearchParams({});
    }
  };

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Link to="/" className="text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Events</h1>
          <p className="text-slate-400">
            {eventsQuery.data?.count ?? 0} event{eventsQuery.data?.count !== 1 ? "s" : ""}
            {selectedDomain && ` from ${selectedDomain}`}
          </p>
        </div>
        <button
          onClick={() => eventsQuery.refetch()}
          className="ml-auto p-2 rounded-lg bg-white/5 hover:bg-white/10 transition-colors"
          disabled={eventsQuery.isFetching}
        >
          <RefreshCw className={`w-5 h-5 ${eventsQuery.isFetching ? "animate-spin" : ""}`} />
        </button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 rounded-xl border border-white/10 bg-white/5 p-4">
        <Filter className="w-5 h-5 text-slate-400" />
        <div className="flex-1">
          <label htmlFor="domain-filter" className="sr-only">Filter by domain</label>
          <select
            id="domain-filter"
            value={selectedDomain}
            onChange={(e) => handleDomainChange(e.target.value)}
            className="w-full md:w-auto bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">All domains</option>
            {domainsQuery.data?.domains?.map((domain: { name: string; display_name: string }) => (
              <option key={domain.name} value={domain.name}>
                {domain.display_name}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Events list */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        {eventsQuery.isLoading ? (
          <div className="text-center py-12 text-slate-500">Loading events...</div>
        ) : eventsQuery.data?.events && eventsQuery.data.events.length > 0 ? (
          <div className="divide-y divide-white/5">
            {eventsQuery.data.events.map((event) => (
              <EventRow key={event.id} event={event} showDomain />
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Activity className="w-16 h-16 mx-auto text-slate-700 mb-4" />
            <h2 className="text-xl font-medium text-slate-300 mb-2">No events found</h2>
            <p className="text-slate-500">
              {selectedDomain
                ? `No events recorded for ${selectedDomain}`
                : "Events will appear here when domain scenarios report them"}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Events Page
 *
 * Shows all events with filtering capabilities.
 *
 * [REQ:LD-QUERY-FILTER] - Time-range filters, domain filters
 */
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, RefreshCw, Activity, Filter, Calendar, X } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { useState, useCallback, useMemo } from "react";

import { EventRow } from "../components/dashboard";
import { Card } from "../components/ui";
import { fetchEvents, fetchDomains } from "../lib/api";

type TimeRangePreset = "all" | "today" | "week" | "month" | "custom";

function getDateRange(preset: TimeRangePreset): { start?: string; end?: string } {
  if (preset === "all") return {};

  const now = new Date();
  const end = now.toISOString();

  switch (preset) {
    case "today": {
      const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      return { start: start.toISOString(), end };
    }
    case "week": {
      const start = new Date(now);
      start.setDate(start.getDate() - 7);
      return { start: start.toISOString(), end };
    }
    case "month": {
      const start = new Date(now);
      start.setMonth(start.getMonth() - 1);
      return { start: start.toISOString(), end };
    }
    default:
      return {};
  }
}

export default function EventsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedDomain, setSelectedDomain] = useState(searchParams.get("domain") || "");
  const [timeRange, setTimeRange] = useState<TimeRangePreset>(
    (searchParams.get("range") as TimeRangePreset) || "all"
  );
  const [customStart, setCustomStart] = useState(searchParams.get("start") || "");
  const [customEnd, setCustomEnd] = useState(searchParams.get("end") || "");

  const domainsQuery = useQuery({
    queryKey: ["domains"],
    queryFn: fetchDomains,
  });

  // Compute effective date range based on selection
  const dateRange = useMemo(() => {
    if (timeRange === "custom") {
      return {
        start: customStart || undefined,
        end: customEnd || undefined,
      };
    }
    return getDateRange(timeRange);
  }, [timeRange, customStart, customEnd]);

  const eventsQuery = useQuery({
    queryKey: ["events", { domain: selectedDomain, ...dateRange, limit: 50 }],
    queryFn: () => fetchEvents({
      domain: selectedDomain || undefined,
      start: dateRange.start,
      end: dateRange.end,
      limit: 50,
    }),
    refetchInterval: 30000,
  });

  const updateSearchParams = useCallback((updates: Record<string, string>) => {
    const newParams = new URLSearchParams(searchParams);
    Object.entries(updates).forEach(([key, value]) => {
      if (value) {
        newParams.set(key, value);
      } else {
        newParams.delete(key);
      }
    });
    setSearchParams(newParams);
  }, [searchParams, setSearchParams]);

  const handleDomainChange = (domain: string) => {
    setSelectedDomain(domain);
    updateSearchParams({ domain });
  };

  const handleTimeRangeChange = (range: TimeRangePreset) => {
    setTimeRange(range);
    if (range !== "custom") {
      setCustomStart("");
      setCustomEnd("");
      updateSearchParams({ range, start: "", end: "" });
    } else {
      updateSearchParams({ range });
    }
  };

  const handleCustomDateChange = (type: "start" | "end", value: string) => {
    if (type === "start") {
      setCustomStart(value);
      updateSearchParams({ start: value });
    } else {
      setCustomEnd(value);
      updateSearchParams({ end: value });
    }
  };

  const clearFilters = () => {
    setSelectedDomain("");
    setTimeRange("all");
    setCustomStart("");
    setCustomEnd("");
    setSearchParams({});
  };

  const hasFilters = selectedDomain || timeRange !== "all";

  // Build filter description
  const filterDescription = useMemo(() => {
    const parts: string[] = [];
    if (selectedDomain) {
      parts.push(`from ${selectedDomain}`);
    }
    if (timeRange === "today") {
      parts.push("today");
    } else if (timeRange === "week") {
      parts.push("last 7 days");
    } else if (timeRange === "month") {
      parts.push("last 30 days");
    } else if (timeRange === "custom" && (customStart || customEnd)) {
      parts.push("custom range");
    }
    return parts.length > 0 ? parts.join(", ") : "";
  }, [selectedDomain, timeRange, customStart, customEnd]);

  const timeRangeOptions: { value: TimeRangePreset; label: string }[] = [
    { value: "all", label: "All time" },
    { value: "today", label: "Today" },
    { value: "week", label: "Last 7 days" },
    { value: "month", label: "Last 30 days" },
    { value: "custom", label: "Custom range" },
  ];

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Link to="/" className="text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold">Events</h1>
          <p className="text-slate-400">
            {eventsQuery.data?.count ?? 0} event{eventsQuery.data?.count !== 1 ? "s" : ""}
            {filterDescription && ` (${filterDescription})`}
          </p>
        </div>
        {hasFilters && (
          <button
            onClick={clearFilters}
            className="flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg bg-slate-700 text-slate-300 hover:bg-slate-600 transition-colors"
          >
            <X className="w-4 h-4" />
            Clear filters
          </button>
        )}
        <button
          onClick={() => eventsQuery.refetch()}
          className="p-2 rounded-lg bg-white/5 hover:bg-white/10 transition-colors"
          disabled={eventsQuery.isFetching}
        >
          <RefreshCw className={`w-5 h-5 ${eventsQuery.isFetching ? "animate-spin" : ""}`} />
        </button>
      </div>

      {/* Filters */}
      <Card className="space-y-4 p-4">
        {/* Filter row */}
        <div className="flex flex-wrap items-center gap-4">
          <Filter className="w-5 h-5 text-slate-400" />

          {/* Domain filter */}
          <div>
            <label htmlFor="domain-filter" className="sr-only">Filter by domain</label>
            <select
              id="domain-filter"
              value={selectedDomain}
              onChange={(e) => handleDomainChange(e.target.value)}
              className="bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">All domains</option>
              {domainsQuery.data?.domains?.map((domain: { name: string; display_name: string }) => (
                <option key={domain.name} value={domain.name}>
                  {domain.display_name}
                </option>
              ))}
            </select>
          </div>

          {/* Time range filter [REQ:LD-QUERY-FILTER] */}
          <div className="flex items-center gap-2">
            <Calendar className="w-4 h-4 text-slate-400" />
            <select
              id="time-range-filter"
              value={timeRange}
              onChange={(e) => handleTimeRangeChange(e.target.value as TimeRangePreset)}
              className="bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              data-testid="time-range-filter"
            >
              {timeRangeOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Custom date range inputs */}
        {timeRange === "custom" && (
          <div className="flex flex-wrap items-center gap-4 pt-3 border-t border-white/10">
            <span className="text-sm text-slate-400">From:</span>
            <input
              type="date"
              value={customStart}
              onChange={(e) => handleCustomDateChange("start", e.target.value)}
              className="bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              data-testid="custom-start-date"
            />
            <span className="text-sm text-slate-400">To:</span>
            <input
              type="date"
              value={customEnd}
              onChange={(e) => handleCustomDateChange("end", e.target.value)}
              max={new Date().toISOString().split("T")[0]}
              className="bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              data-testid="custom-end-date"
            />
          </div>
        )}
      </Card>

      {/* Events list */}
      <Card padding="lg">
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
              {hasFilters
                ? "No events match the current filters. Try adjusting your filters."
                : "Events will appear here when domain scenarios report them"}
            </p>
            {hasFilters && (
              <button
                onClick={clearFilters}
                className="mt-4 px-4 py-2 text-sm bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
              >
                Clear all filters
              </button>
            )}
          </div>
        )}
      </Card>
    </div>
  );
}

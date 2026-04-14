import { useState, useMemo, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Loader2, AlertCircle, ArrowRight } from "lucide-react";
import { fetchResources, fetchSetupOrder } from "../../lib/api";
import { cn } from "../../lib/utils";
import type { Resource } from "../../types";
import { SearchInput, type SearchInputHandle } from "../ui/SearchInput";

interface StepSelectResourcesProps {
  selected: Set<string>;
  onToggle: (name: string) => void;
}

const STATUS_COLORS: Record<string, string> = {
  running: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
  installed: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  stopped: "bg-slate-500/20 text-slate-300 border-slate-500/30",
};

function statusBadge(status: string) {
  const colors = STATUS_COLORS[status] ?? STATUS_COLORS.stopped;
  return (
    <span className={cn("inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium", colors)}>
      {status}
    </span>
  );
}

interface ResourceCardProps {
  resource: Resource;
  isSelected: boolean;
  onToggle: (name: string) => void;
}

function ResourceCard({ resource, isSelected, onToggle }: ResourceCardProps) {
  return (
    <button
      type="button"
      onClick={() => onToggle(resource.name)}
      data-testid={`resource-card-${resource.name}`}
      aria-pressed={isSelected}
      aria-label={`${resource.name} - ${resource.status}${isSelected ? " (selected)" : ""}`}
      className={cn(
        "flex items-start gap-3 rounded-xl border p-3 text-left transition-all duration-150 sm:p-4",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50",
        "hover:scale-[1.02] active:scale-[0.98]",
        isSelected
          ? "border-emerald-500/50 bg-emerald-500/10 shadow-[0_0_12px_rgba(16,185,129,0.08)]"
          : "border-white/10 bg-white/5 hover:bg-white/10 hover:border-white/20"
      )}
    >
      <div
        className={cn(
          "mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border transition-colors",
          isSelected
            ? "border-emerald-500 bg-emerald-500 text-white"
            : "border-white/30 bg-transparent"
        )}
      >
        {isSelected && <Check className="h-3 w-3" aria-hidden="true" />}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium">{resource.name}</span>
          {statusBadge(resource.status)}
        </div>
        {resource.installed && <p className="mt-1 text-xs text-slate-300">Installed locally</p>}
      </div>
    </button>
  );
}

function groupByCategory(resources: Resource[]) {
  const groups: Record<string, Resource[]> = {};
  for (const r of resources) {
    const cat = r.category || "Other";
    if (!groups[cat]) groups[cat] = [];
    groups[cat].push(r);
  }
  return groups;
}

export function StepSelectResources({ selected, onToggle }: StepSelectResourcesProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const searchRef = useRef<SearchInputHandle>(null);

  const { data: resources, isLoading, error } = useQuery({
    queryKey: ["resources"],
    queryFn: fetchResources,
  });

  const { data: setupOrder } = useQuery({
    queryKey: ["setup-order"],
    queryFn: fetchSetupOrder,
  });

  const filteredResources = useMemo(() => {
    const all = resources ?? [];
    if (!searchTerm) return all;
    const term = searchTerm.toLowerCase();
    return all.filter((r) => r.name.toLowerCase().includes(term) || r.category.toLowerCase().includes(term));
  }, [resources, searchTerm]);

  const groups = useMemo(() => groupByCategory(filteredResources), [filteredResources]);

  return (
    <div data-testid="step-select-resources">
      <h1 className="text-xl font-semibold sm:text-2xl">Select Resources</h1>
      <p className="mt-2 text-sm text-slate-300 sm:text-base">
        Choose the resources you want to enable. They are grouped by category.
      </p>

      {isLoading && (
        <div className="flex items-center justify-center py-16" data-testid="step-resources-loading" role="status">
          <Loader2 className="h-6 w-6 animate-spin text-slate-300" aria-hidden="true" />
          <span className="ml-2 text-slate-300">Loading resources...</span>
        </div>
      )}

      {!isLoading && error && (
        <div className="flex items-center justify-center py-16 text-red-400" data-testid="step-resources-error" role="alert">
          <AlertCircle className="mr-2 h-5 w-5" aria-hidden="true" />
          Failed to load resources. Make sure the API is running.
        </div>
      )}

      {!isLoading && !error && (
        <>
          {/* Search filter */}
          <div className="mt-3">
            <SearchInput
              ref={searchRef}
              value={searchTerm}
              onChange={setSearchTerm}
              placeholder="Filter resources..."
              ariaLabel="Filter resources by name or category"
              testId="resource-search"
              clearTestId="resource-search-clear"
            />
          </div>

          <div className="mt-2 flex items-center justify-between text-sm text-slate-300">
            <span aria-live="polite">
              {selected.size} resource{selected.size !== 1 ? "s" : ""} selected
            </span>
            {searchTerm && (
              <span aria-live="polite" data-testid="resource-filter-count">
                {filteredResources.length} of {(resources ?? []).length} shown
              </span>
            )}
          </div>

          {setupOrder && selected.size > 0 && (
            <div data-testid="setup-order-hint" className="mt-4 rounded-lg border border-blue-500/30 bg-blue-500/10 p-3">
              <p className="text-sm font-medium text-blue-400 flex items-center gap-1">
                <ArrowRight className="h-4 w-4" aria-hidden="true" /> Recommended setup order:
              </p>
              <p className="mt-1 text-sm text-slate-300">
                {setupOrder.setup_order
                  .filter((entry) => selected.has(entry.name))
                  .sort((a, b) => a.order - b.order)
                  .map((entry) => entry.name)
                  .join(" → ")}
              </p>
            </div>
          )}

          {searchTerm && filteredResources.length === 0 && (
            <div className="mt-6 text-center py-8 text-slate-300" data-testid="resource-no-results">
              <p className="text-sm font-medium">No resources match &ldquo;{searchTerm}&rdquo;</p>
              <button
                type="button"
                onClick={() => { setSearchTerm(""); searchRef.current?.focus(); }}
                className="mt-2 text-xs text-emerald-400 underline underline-offset-2 hover:text-emerald-300 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500 rounded"
              >
                Clear filter
              </button>
            </div>
          )}

          {Object.entries(groups).map(([category, items]) => {
            const allInCategorySelected = items.every((r) => selected.has(r.name));
            const someInCategorySelected = items.some((r) => selected.has(r.name));
            return (
            <div key={category} className="mt-6">
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-sm font-medium uppercase tracking-wider text-slate-300">
                  {category}
                </h2>
                <button
                  type="button"
                  onClick={() => {
                    // Toggle all: if every item is selected, deselect all; otherwise select the unselected ones
                    const shouldSelect = !allInCategorySelected;
                    for (const r of items) {
                      if (selected.has(r.name) !== shouldSelect) onToggle(r.name);
                    }
                  }}
                  data-testid={`category-toggle-${category}`}
                  className="text-xs text-emerald-400 hover:text-emerald-300 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500 rounded px-1.5 py-0.5 transition-colors"
                  aria-label={allInCategorySelected ? `Deselect all ${category} resources` : `Select all ${category} resources`}
                >
                  {allInCategorySelected ? "Deselect All" : someInCategorySelected ? "Select Rest" : "Select All"}
                </button>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                {items.map((resource) => (
                  <ResourceCard
                    key={resource.name}
                    resource={resource}
                    isSelected={selected.has(resource.name)}
                    onToggle={onToggle}
                  />
                ))}
              </div>
            </div>
            );
          })}
        </>
      )}
    </div>
  );
}

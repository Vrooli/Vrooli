/**
 * Tab-aware search input. Active tab filters the loaded list
 * client-side by `displayName` / `owner` / `scopePath`. History tab
 * forwards the query to the server's full-text search across owner /
 * runId / sandbox id.
 */

import { Search, X } from "lucide-react";

import { Input } from "../ui/input";

interface SidebarSearchBarProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export function SidebarSearchBar({ value, onChange, placeholder }: SidebarSearchBarProps) {
  return (
    <div className="relative">
      <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-slate-500 pointer-events-none" />
      <Input
        type="search"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder ?? "Search sandboxes..."}
        className="h-8 pl-7 pr-7 text-xs"
        data-testid="sidebar-search"
      />
      {value && (
        <button
          type="button"
          aria-label="Clear search"
          onClick={() => onChange("")}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded hover:bg-slate-800 text-slate-500"
          data-testid="sidebar-search-clear"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  );
}

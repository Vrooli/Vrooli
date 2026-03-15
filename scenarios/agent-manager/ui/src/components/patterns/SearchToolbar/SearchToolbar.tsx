import * as React from "react";
import { cn } from "../../../lib/utils";
import { SearchInput } from "./SearchInput";
import { type FilterOption } from "./FilterDropdown";
import { type SortOption } from "./SortDropdown";
import { FilterPopoverButton } from "./FilterPopoverButton";

export interface FilterConfig {
  id: string;
  label: string;
  value: string;
  options: FilterOption[];
  onChange: (value: string) => void;
  allLabel?: string;
  defaultValue?: string;
}

interface SearchToolbarProps {
  searchValue: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder?: string;
  filters?: FilterConfig[];
  sortOptions?: SortOption[];
  currentSort?: string;
  onSortChange?: (value: string) => void;
  defaultSort?: string;
  className?: string;
  children?: React.ReactNode;
}

export function SearchToolbar({
  searchValue,
  onSearchChange,
  searchPlaceholder,
  filters,
  sortOptions,
  currentSort,
  onSortChange,
  defaultSort,
  className,
  children,
}: SearchToolbarProps) {
  const hasFilters = filters && filters.length > 0;
  const hasSort =
    sortOptions && sortOptions.length > 0 && currentSort && onSortChange;

  return (
    <div className={cn("flex gap-2 items-center", className)}>
      <SearchInput
        value={searchValue}
        onChange={onSearchChange}
        placeholder={searchPlaceholder}
      />
      {(hasFilters || hasSort) && (
        <FilterPopoverButton
          filters={filters}
          sortOptions={sortOptions}
          currentSort={currentSort}
          onSortChange={onSortChange}
          defaultSort={defaultSort}
        />
      )}
      {children}
    </div>
  );
}

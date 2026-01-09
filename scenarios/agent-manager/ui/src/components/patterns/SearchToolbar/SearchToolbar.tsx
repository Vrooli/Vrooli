import * as React from "react";
import { cn } from "../../../lib/utils";
import { SearchInput } from "./SearchInput";
import { FilterDropdown, type FilterOption } from "./FilterDropdown";
import { SortDropdown, type SortOption } from "./SortDropdown";

export interface FilterConfig {
  id: string;
  label: string;
  value: string;
  options: FilterOption[];
  onChange: (value: string) => void;
  allLabel?: string;
}

interface SearchToolbarProps {
  searchValue: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder?: string;
  filters?: FilterConfig[];
  sortOptions?: SortOption[];
  currentSort?: string;
  onSortChange?: (value: string) => void;
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
  className,
  children,
}: SearchToolbarProps) {
  return (
    <div className={cn("flex flex-wrap gap-3 items-center", className)}>
      <SearchInput
        value={searchValue}
        onChange={onSearchChange}
        placeholder={searchPlaceholder}
      />
      {filters?.map((filter) => (
        <FilterDropdown
          key={filter.id}
          value={filter.value}
          onChange={filter.onChange}
          options={filter.options}
          label={filter.label}
          allLabel={filter.allLabel}
        />
      ))}
      {sortOptions && sortOptions.length > 0 && currentSort && onSortChange && (
        <SortDropdown
          value={currentSort}
          onChange={onSortChange}
          options={sortOptions}
        />
      )}
      {children}
    </div>
  );
}

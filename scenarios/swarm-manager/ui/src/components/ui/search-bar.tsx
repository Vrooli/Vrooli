/**
 * SearchBar Component
 *
 * A reusable search input with consistent styling, used across list pages.
 * Extracts the common search pattern from IdeasPage and ScenariosPage.
 */

import { Search } from "lucide-react";
import { Input, type InputProps } from "./input";

export interface SearchBarProps extends Omit<InputProps, "leftIcon" | "type"> {
  /** Optional custom width class (default: "flex-1 sm:w-80") */
  widthClass?: string;
}

/**
 * SearchBar provides a consistent search input pattern across the app.
 *
 * Usage:
 * ```tsx
 * <SearchBar
 *   placeholder="Search ideas..."
 *   value={searchTerm}
 *   onChange={(e) => setSearchTerm(e.target.value)}
 *   data-testid="ideas-search"
 * />
 * ```
 */
export function SearchBar({
  widthClass = "flex-1 sm:w-80",
  className,
  ...props
}: SearchBarProps) {
  return (
    <div className={widthClass}>
      <Input
        type="text"
        leftIcon={<Search className="h-4 w-4" />}
        className={className}
        {...props}
      />
    </div>
  );
}

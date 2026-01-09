import React, { useState, useRef, useEffect } from 'react';
import { Filter, RefreshCw, ChevronDown } from 'lucide-react';
import { usePopoverPosition } from '@hooks/usePopoverPosition';

export type StatusFilter = 'all' | 'running' | 'completed' | 'failed' | 'cancelled';

/** Default filter order */
const DEFAULT_FILTERS: StatusFilter[] = ['all', 'running', 'completed', 'failed', 'cancelled'];

/** Status badge color mapping */
const statusBadgeColors: Record<StatusFilter, { bg: string; text: string; activeBg: string }> = {
  all: { bg: 'bg-gray-600', text: 'text-gray-200', activeBg: 'bg-gray-500' },
  running: { bg: 'bg-green-600/80', text: 'text-green-100', activeBg: 'bg-green-500' },
  completed: { bg: 'bg-emerald-600/80', text: 'text-emerald-100', activeBg: 'bg-emerald-500' },
  failed: { bg: 'bg-red-600/80', text: 'text-red-100', activeBg: 'bg-red-500' },
  cancelled: { bg: 'bg-amber-600/80', text: 'text-amber-100', activeBg: 'bg-amber-500' },
};

export interface ExecutionFiltersProps {
  /** Currently selected filter */
  statusFilter: StatusFilter;
  /** Called when filter changes */
  onStatusFilterChange: (filter: StatusFilter) => void;
  /** Counts for each status */
  statusCounts: Partial<Record<StatusFilter, number>>;
  /** Called when refresh is requested */
  onRefresh: () => void;
  /** Whether refresh is in progress */
  isRefreshing: boolean;
  /** Which filters to show (defaults to all) */
  filters?: StatusFilter[];
  /** Whether to show the filter icon */
  showFilterIcon?: boolean;
  /** Whether to show the refresh button label on desktop */
  showRefreshLabel?: boolean;
  /** Optional test ID prefix for filter buttons */
  testIdPrefix?: string;
  /** Optional className for the container */
  className?: string;
}

/** Format filter label */
const formatFilterLabel = (filter: StatusFilter): string =>
  filter === 'all' ? 'All' : filter.charAt(0).toUpperCase() + filter.slice(1);

/**
 * Reusable execution filter controls with notification badges.
 *
 * Features:
 * - Desktop: inline button group with count badges
 * - Mobile: dropdown selector
 * - Refresh button
 * - Configurable filter set
 */
export const ExecutionFilters: React.FC<ExecutionFiltersProps> = ({
  statusFilter,
  onStatusFilterChange,
  statusCounts,
  onRefresh,
  isRefreshing,
  filters = DEFAULT_FILTERS,
  showFilterIcon = true,
  showRefreshLabel = false,
  testIdPrefix,
  className = '',
}) => {
  const [isMobileOpen, setIsMobileOpen] = useState(false);

  const containerRef = useRef<HTMLDivElement | null>(null);
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const dropdownRef = useRef<HTMLDivElement | null>(null);

  const { floatingStyles } = usePopoverPosition(buttonRef, dropdownRef, {
    isOpen: isMobileOpen,
    placementPriority: ['bottom-start', 'bottom-end', 'top-start', 'top-end'],
    matchReferenceWidth: true,
  });

  // Close dropdown on outside click
  useEffect(() => {
    if (!isMobileOpen) return;

    const handleClickOutside = (event: MouseEvent | TouchEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        setIsMobileOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('touchstart', handleClickOutside);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('touchstart', handleClickOutside);
    };
  }, [isMobileOpen]);

  const getCount = (filter: StatusFilter): number => statusCounts[filter] ?? 0;

  const renderBadge = (filter: StatusFilter, isActive: boolean) => {
    const count = getCount(filter);
    const colors = statusBadgeColors[filter];

    return (
      <span
        className={`
          inline-flex items-center justify-center min-w-[1.25rem] h-5 px-1.5 text-xs font-medium rounded-full
          ${isActive ? colors.activeBg : colors.bg} ${colors.text}
          transition-colors
        `}
      >
        {count}
      </span>
    );
  };

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      {/* Desktop filters */}
      <div className="hidden md:flex items-center gap-2 flex-wrap">
        {showFilterIcon && <Filter size={14} className="text-gray-500" />}
        <div className="flex items-center gap-1 bg-gray-800/50 rounded-lg p-1">
          {filters.map((filter) => {
            const isActive = statusFilter === filter;
            return (
              <button
                key={filter}
                data-testid={testIdPrefix ? `${testIdPrefix}-${filter}` : undefined}
                onClick={() => onStatusFilterChange(filter)}
                className={`
                  flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium rounded-md transition-colors
                  ${isActive
                    ? 'bg-gray-700 text-white'
                    : 'text-gray-400 hover:text-gray-300 hover:bg-gray-700/50'
                  }
                `}
              >
                <span>{formatFilterLabel(filter)}</span>
                {renderBadge(filter, isActive)}
              </button>
            );
          })}
        </div>
      </div>

      {/* Mobile filter dropdown */}
      <div className="flex items-center gap-2 md:hidden flex-1">
        <div className="relative flex-1" ref={containerRef}>
          <button
            ref={buttonRef}
            type="button"
            onClick={() => setIsMobileOpen((prev) => !prev)}
            className="w-full flex items-center justify-between gap-2 rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white hover:border-blue-500 transition-colors"
            aria-haspopup="listbox"
            aria-expanded={isMobileOpen}
          >
            <span className="flex items-center gap-2">
              {showFilterIcon && <Filter size={14} />}
              <span>{formatFilterLabel(statusFilter)}</span>
              {renderBadge(statusFilter, true)}
            </span>
            <ChevronDown
              size={16}
              className={`transition-transform ${isMobileOpen ? 'rotate-180' : ''}`}
            />
          </button>
          {isMobileOpen && (
            <div
              ref={dropdownRef}
              style={floatingStyles}
              className="z-20 rounded-lg border border-gray-700 bg-gray-800 shadow-lg overflow-hidden"
            >
              {filters.map((filter) => {
                const isActive = statusFilter === filter;
                return (
                  <button
                    key={filter}
                    type="button"
                    onClick={() => {
                      onStatusFilterChange(filter);
                      setIsMobileOpen(false);
                    }}
                    className={`
                      w-full flex items-center justify-between px-4 py-2.5 text-sm transition-colors
                      ${isActive
                        ? 'bg-blue-500/20 text-white'
                        : 'text-gray-300 hover:bg-gray-700/50 hover:text-white'
                      }
                    `}
                  >
                    <span>{formatFilterLabel(filter)}</span>
                    {renderBadge(filter, isActive)}
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Refresh button */}
      <button
        type="button"
        onClick={onRefresh}
        disabled={isRefreshing}
        className={`
          flex items-center gap-2 p-2 text-gray-400 hover:text-white hover:bg-gray-700
          rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed
          ${showRefreshLabel ? 'px-3' : ''}
        `}
        title="Refresh"
      >
        <RefreshCw size={16} className={isRefreshing ? 'animate-spin' : ''} />
        {showRefreshLabel && <span className="text-sm hidden sm:inline">Refresh</span>}
      </button>
    </div>
  );
};

export default ExecutionFilters;

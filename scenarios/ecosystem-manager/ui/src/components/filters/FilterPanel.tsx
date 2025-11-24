/**
 * FilterPanel Component
 * Provides filtering controls for tasks (search, type, operation, priority, column visibility)
 */

import { X, Search, Filter as FilterIcon } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Checkbox } from '../ui/checkbox';
import { useAppState } from '../../contexts/AppStateContext';
import { useDebounce } from '../../hooks/useDebounce';
import { useQueryParams } from '../../hooks/useQueryParams';
import { useState, useEffect, useLayoutEffect, useRef, type PointerEvent as ReactPointerEvent } from 'react';
import type { TaskStatus, TaskType, OperationType, Priority, TaskSort } from '../../types/api';

const COLUMN_LABELS: Record<TaskStatus, string> = {
  pending: 'Pending',
  'in-progress': 'Active',
  completed: 'Completed',
  'completed-finalized': 'Finished',
  failed: 'Failed',
  'failed-blocked': 'Blocked',
  archived: 'Archived',
};

export function FilterPanel() {
  const {
    filters,
    updateFilter,
    clearFilters,
    columnVisibility,
    toggleColumnVisibility,
    setFilterPanelOpen,
  } = useAppState();

  // Local state for search input (to avoid lag)
  const [searchValue, setSearchValue] = useState(filters.search);
  const debouncedSearch = useDebounce(searchValue, 300);
  const [position, setPosition] = useState({ x: 0, y: 80 });
  const [isDragging, setIsDragging] = useState(false);
  const dragOffset = useRef({ x: 0, y: 0 });
  const panelRef = useRef<HTMLDivElement | null>(null);

  // Sync debounced search with filters
  useEffect(() => {
    updateFilter('search', debouncedSearch);
  }, [debouncedSearch, updateFilter]);

  // Sync filters with URL
  useQueryParams(filters, (newFilters) => {
    Object.entries(newFilters).forEach(([key, value]) => {
      updateFilter(key as keyof typeof filters, value as string);
    });
  });

  // Initialize starting position once panel dimensions are known (useLayoutEffect to avoid flicker)
  useLayoutEffect(() => {
    const panel = panelRef.current;
    if (!panel || typeof window === 'undefined') return;

    const margin = 16;
    const width = panel.offsetWidth || 360;
    const startX = window.innerWidth - width - margin;
    setPosition({ x: Math.max(margin, startX), y: 80 });
  }, []);

  // Handle dragging
  useEffect(() => {
    if (!isDragging) return;

    const handlePointerMove = (event: PointerEvent) => {
      const panel = panelRef.current;
      if (!panel) return;

      const margin = 12;
      const width = panel.offsetWidth || 360;
      const height = panel.offsetHeight || 400;
      const maxX = window.innerWidth - width - margin;
      const maxY = window.innerHeight - height - margin;

      setPosition({
        x: Math.min(Math.max(event.clientX - dragOffset.current.x, margin), maxX),
        y: Math.min(Math.max(event.clientY - dragOffset.current.y, margin), maxY),
      });
    };

    const handlePointerUp = () => {
      setIsDragging(false);
      document.body.style.userSelect = '';
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', handlePointerUp);
    document.body.style.userSelect = 'none';

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerUp);
      document.body.style.userSelect = '';
    };
  }, [isDragging]);

  const defaultSort: TaskSort = 'updated_desc';
  const activeFilterCount = Object.entries(filters).reduce((count, [key, value]) => {
    if (!value) return count;
    if (key === 'sort' && value === defaultSort) return count;
    return count + 1;
  }, 0);
  const visibleColumns = (Object.keys(columnVisibility) as TaskStatus[]).filter(
    (status) => columnVisibility[status]
  );

  const handleClearAll = () => {
    clearFilters();
    setSearchValue('');
  };

  const handleHeaderPointerDown = (event: ReactPointerEvent) => {
    if (event.button !== 0) return;
    const target = event.target as HTMLElement;
    if (target.closest('button')) return;

    const panel = panelRef.current;
    if (!panel) return;

    const rect = panel.getBoundingClientRect();
    dragOffset.current = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    };

    setIsDragging(true);
    event.preventDefault();
  };

  return (
    <div
      ref={panelRef}
      className="fixed w-full sm:w-96 max-w-[calc(100vw-2rem)] sm:max-w-none bg-card border border-border rounded-lg shadow-2xl z-20 overflow-hidden"
      style={{ left: position.x, top: position.y }}
    >
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 py-3 border-b border-border/60 bg-muted/60 cursor-move select-none"
        onPointerDown={handleHeaderPointerDown}
      >
        <div className="flex items-center gap-2">
          <FilterIcon className="h-4 w-4 text-muted-foreground" />
          <h3 className="font-medium text-sm">Filters</h3>
          {activeFilterCount > 0 && (
            <span className="px-2 py-0.5 text-xs bg-blue-500 text-white rounded-full">
              {activeFilterCount}
            </span>
          )}
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setFilterPanelOpen(false)}
          className="h-8 w-8 p-0"
          onPointerDown={(e) => e.stopPropagation()}
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Filter Controls */}
      <div className="p-4 space-y-4 max-h-[calc(100vh-200px)] overflow-y-auto">
        {/* Search */}
        <div className="space-y-2">
          <Label htmlFor="search" className="text-xs text-muted-foreground">
            Search
          </Label>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              id="search"
              type="text"
              placeholder="Search tasks by title or ID..."
              value={searchValue}
              onChange={(e) => setSearchValue(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        {/* Type Filter */}
        <div className="space-y-2">
          <Label htmlFor="type" className="text-xs text-muted-foreground">
            Type
          </Label>
          <Select
            value={filters.type || 'all'}
            onValueChange={(value) => updateFilter('type', value === 'all' ? '' : (value as TaskType))}
          >
            <SelectTrigger id="type">
              <SelectValue placeholder="All types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All types</SelectItem>
              <SelectItem value="resource">Resource</SelectItem>
              <SelectItem value="scenario">Scenario</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Operation Filter */}
        <div className="space-y-2">
          <Label htmlFor="operation" className="text-xs text-muted-foreground">
            Operation
          </Label>
          <Select
            value={filters.operation || 'all'}
            onValueChange={(value) => updateFilter('operation', value === 'all' ? '' : (value as OperationType))}
          >
            <SelectTrigger id="operation">
              <SelectValue placeholder="All operations" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All operations</SelectItem>
              <SelectItem value="generator">Generator</SelectItem>
              <SelectItem value="improver">Improver</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Priority Filter */}
        <div className="space-y-2">
          <Label htmlFor="priority" className="text-xs text-muted-foreground">
            Priority
          </Label>
          <Select
            value={filters.priority || 'all'}
            onValueChange={(value) => updateFilter('priority', value === 'all' ? '' : (value as Priority))}
          >
            <SelectTrigger id="priority">
              <SelectValue placeholder="All priorities" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All priorities</SelectItem>
              <SelectItem value="critical">Critical</SelectItem>
              <SelectItem value="high">High</SelectItem>
              <SelectItem value="medium">Medium</SelectItem>
              <SelectItem value="low">Low</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Sort */}
        <div className="space-y-2">
          <Label htmlFor="sort" className="text-xs text-muted-foreground">
            Sort by
          </Label>
          <Select
            value={filters.sort || defaultSort}
            onValueChange={(value) => updateFilter('sort', value as TaskSort)}
          >
            <SelectTrigger id="sort">
              <SelectValue placeholder="Most recently updated" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="updated_desc">Most recently updated</SelectItem>
              <SelectItem value="updated_asc">Least recently updated</SelectItem>
              <SelectItem value="created_desc">Newest first (created)</SelectItem>
              <SelectItem value="created_asc">Oldest first (created)</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Divider */}
        <div className="border-t border-border/60 pt-4">
          <Label className="text-xs text-muted-foreground mb-3 block">
            Column Visibility
          </Label>
          <div className="space-y-2">
            {(Object.keys(columnVisibility) as TaskStatus[]).map((status) => (
              <div key={status} className="flex items-center gap-2">
                <Checkbox
                  id={`column-${status}`}
                  checked={columnVisibility[status]}
                  onCheckedChange={() => toggleColumnVisibility(status)}
                />
                <label
                  htmlFor={`column-${status}`}
                  className="text-sm cursor-pointer flex-1"
                >
                  {COLUMN_LABELS[status]}
                </label>
              </div>
            ))}
          </div>
          <p className="text-xs text-muted-foreground mt-3">
            {visibleColumns.length} of {Object.keys(columnVisibility).length} columns visible
          </p>
        </div>
      </div>

      {/* Footer Actions */}
      <div className="px-4 py-3 border-t border-border/60 bg-muted/60 flex items-center justify-between">
        <Button
          variant="outline"
          size="sm"
          onClick={handleClearAll}
          disabled={activeFilterCount === 0}
          className="text-xs"
        >
          Clear All
        </Button>
        <p className="text-xs text-muted-foreground">
          {activeFilterCount} {activeFilterCount === 1 ? 'filter' : 'filters'} active
        </p>
      </div>
    </div>
  );
}

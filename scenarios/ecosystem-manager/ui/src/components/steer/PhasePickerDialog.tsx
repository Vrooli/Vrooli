import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Search, Check } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn, formatPhaseName } from '@/lib/utils';
import { usePhaseUsage, type SortOption } from '@/hooks/usePhaseUsage';
import type { PhaseInfo } from '@/types/api';

interface PhasePickerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value?: string;
  onSelect: (phaseName: string) => void;
  phaseNames: PhaseInfo[];
  isLoading?: boolean;
  title?: string;
  description?: string;
}

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: 'name', label: 'Name (A-Z)' },
  { value: 'recent', label: 'Recent' },
  { value: 'most-used', label: 'Most Used' },
];

export function PhasePickerDialog({
  open,
  onOpenChange,
  value,
  onSelect,
  phaseNames,
  isLoading,
  title = 'Select Phase',
  description = 'Choose a steering phase for task execution.',
}: PhasePickerDialogProps) {
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState<SortOption>('recent');
  const [focusedIndex, setFocusedIndex] = useState(0);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  const { trackUsage, sortByRecent, sortByFrequency, sortByName } = usePhaseUsage();

  // Filter by search
  const filteredPhases = useMemo(() => {
    if (!search.trim()) return phaseNames;

    const searchLower = search.toLowerCase();
    return phaseNames.filter((phase) => {
      const nameLower = phase.name.toLowerCase();
      const descLower = (phase.description || '').toLowerCase();
      return nameLower.includes(searchLower) || descLower.includes(searchLower);
    });
  }, [phaseNames, search]);

  // Apply sort
  const sortedPhases = useMemo(() => {
    switch (sortBy) {
      case 'recent':
        return sortByRecent(filteredPhases);
      case 'most-used':
        return sortByFrequency(filteredPhases);
      case 'name':
      default:
        return sortByName(filteredPhases);
    }
  }, [filteredPhases, sortBy, sortByRecent, sortByFrequency, sortByName]);

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setSearch('');
      setFocusedIndex(0);
      // Focus search input after a short delay
      setTimeout(() => searchInputRef.current?.focus(), 100);
    }
  }, [open]);

  // Keep focused index in bounds
  useEffect(() => {
    if (focusedIndex >= sortedPhases.length) {
      setFocusedIndex(Math.max(0, sortedPhases.length - 1));
    }
  }, [sortedPhases.length, focusedIndex]);

  const handleSelect = useCallback(
    (phaseName: string) => {
      trackUsage(phaseName);
      onSelect(phaseName);
      onOpenChange(false);
    },
    [trackUsage, onSelect, onOpenChange]
  );

  // Grid has 2 columns on sm+ screens, 1 column on smaller
  // We'll assume 2 columns for keyboard nav since that's the common case
  const GRID_COLUMNS = 2;

  // Keyboard navigation with grid support
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (sortedPhases.length === 0) return;

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          // Move down one row (skip GRID_COLUMNS items)
          setFocusedIndex((prev) => Math.min(prev + GRID_COLUMNS, sortedPhases.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          // Move up one row (skip GRID_COLUMNS items)
          setFocusedIndex((prev) => Math.max(prev - GRID_COLUMNS, 0));
          break;
        case 'ArrowRight':
          e.preventDefault();
          // Move to next item
          setFocusedIndex((prev) => Math.min(prev + 1, sortedPhases.length - 1));
          break;
        case 'ArrowLeft':
          e.preventDefault();
          // Move to previous item
          setFocusedIndex((prev) => Math.max(prev - 1, 0));
          break;
        case 'Enter':
          e.preventDefault();
          if (sortedPhases[focusedIndex]) {
            handleSelect(sortedPhases[focusedIndex].name);
          }
          break;
        case 'Escape':
          e.preventDefault();
          onOpenChange(false);
          break;
      }
    },
    [sortedPhases, focusedIndex, handleSelect, onOpenChange]
  );

  // Scroll focused item into view
  useEffect(() => {
    if (!gridRef.current) return;
    const focusedElement = gridRef.current.querySelector(`[data-index="${focusedIndex}"]`);
    if (focusedElement) {
      focusedElement.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  }, [focusedIndex]);

  const currentSortOption = SORT_OPTIONS.find((o) => o.value === sortBy) || SORT_OPTIONS[0];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {/* Search and Sort Controls */}
        <div className="flex gap-2 mt-2" onKeyDown={handleKeyDown}>
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <Input
              ref={searchInputRef}
              placeholder="Search phases..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>

          <Select value={sortBy} onValueChange={(val) => setSortBy(val as SortOption)}>
            <SelectTrigger className="w-[140px] shrink-0">
              <SelectValue placeholder="Sort by" />
            </SelectTrigger>
            <SelectContent>
              {SORT_OPTIONS.map(({ value: optValue, label }) => (
                <SelectItem key={optValue} value={optValue}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Phase Grid */}
        <div
          ref={gridRef}
          className="flex-1 overflow-y-auto mt-4 min-h-[300px] max-h-[400px]"
          onKeyDown={handleKeyDown}
          tabIndex={-1}
        >
          {isLoading ? (
            <div className="flex items-center justify-center h-full text-slate-400">
              Loading phases...
            </div>
          ) : sortedPhases.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-slate-400">
              <p>No phases found</p>
              {search && (
                <p className="text-sm mt-1">
                  Try a different search term
                </p>
              )}
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {sortedPhases.map((phase, index) => (
                <button
                  key={phase.name}
                  data-index={index}
                  type="button"
                  onClick={() => handleSelect(phase.name)}
                  className={cn(
                    'flex flex-col items-start p-3 rounded-lg border text-left transition-colors',
                    'hover:bg-slate-800 hover:border-slate-600',
                    value === phase.name && 'border-blue-500 bg-blue-500/10',
                    index === focusedIndex && 'ring-2 ring-blue-500 ring-offset-1 ring-offset-slate-900'
                  )}
                >
                  <div className="flex items-center gap-2 w-full">
                    <span className="font-medium text-sm text-slate-100">
                      {formatPhaseName(phase.name)}
                    </span>
                    {value === phase.name && (
                      <Check className="h-4 w-4 text-blue-400 ml-auto shrink-0" />
                    )}
                  </div>
                  {phase.description && (
                    <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                      {phase.description}
                    </p>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Footer with count */}
        <div className="text-xs text-slate-500 mt-2 pt-2 border-t border-slate-700">
          {sortedPhases.length} of {phaseNames.length} phases
          {search && ` matching "${search}"`}
        </div>
      </DialogContent>
    </Dialog>
  );
}

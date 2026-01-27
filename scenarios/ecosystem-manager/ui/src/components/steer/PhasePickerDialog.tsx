import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Check, Cloud, CloudOff, Loader2, RefreshCw, Search } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn, formatPhaseName, normalizeSteerMode } from '@/lib/utils';
import { usePhaseUsage, type SortOption } from '@/hooks/usePhaseUsage';
import { useSteerSkills, useSyncSkills } from '@/hooks/useSkills';
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

const BUILT_IN_PHASES: PhaseInfo[] = [
  { name: 'progress', description: 'Advance core objectives and operational targets.' },
  { name: 'ux', description: 'Improve usability, accessibility, and user flows.' },
  { name: 'refactor', description: 'Raise code quality without changing behavior.' },
  { name: 'test', description: 'Expand coverage and harden edge cases.' },
  { name: 'explore', description: 'Explore options before committing to a path.' },
  { name: 'polish', description: 'Finalize copy, visuals, and small fixes.' },
  { name: 'performance', description: 'Profile and optimize slow paths.' },
  { name: 'security', description: 'Reduce vulnerabilities and tighten validation.' },
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

  const { data: steerSkills = [], isLoading: skillsLoading, isError: skillsError } = useSteerSkills();
  const syncSkills = useSyncSkills();
  const { trackUsage, sortByRecent, sortByFrequency, sortByName } = usePhaseUsage();
  const normalizedValue = normalizeSteerMode(value);
  const promptIsLoading = typeof isLoading === 'boolean' ? isLoading : skillsLoading;
  const promptHasError = skillsError;
  const promptHasData = phaseNames.length > 0 || steerSkills.length > 0;

  const filterPhases = useCallback(
    (phases: PhaseInfo[]) => {
      if (!search.trim()) return phases;
      const searchLower = search.toLowerCase();
      return phases.filter((phase) => {
        const nameLower = phase.name.toLowerCase();
        const descLower = (phase.description || '').toLowerCase();
        return nameLower.includes(searchLower) || descLower.includes(searchLower);
      });
    },
    [search]
  );

  const sortPhases = useCallback(
    (phases: PhaseInfo[]) => {
      switch (sortBy) {
        case 'recent':
          return sortByRecent(phases);
        case 'most-used':
          return sortByFrequency(phases);
        case 'name':
        default:
          return sortByName(phases);
      }
    },
    [sortBy, sortByRecent, sortByFrequency, sortByName]
  );

  const filteredPromptPhases = useMemo(() => filterPhases(phaseNames), [filterPhases, phaseNames]);
  const filteredBuiltInPhases = useMemo(() => filterPhases(BUILT_IN_PHASES), [filterPhases]);

  const sortedPromptPhases = useMemo(
    () => sortPhases(filteredPromptPhases),
    [filteredPromptPhases, sortPhases]
  );
  const sortedBuiltInPhases = useMemo(
    () => sortPhases(filteredBuiltInPhases),
    [filteredBuiltInPhases, sortPhases]
  );

  const combinedPhases = useMemo(
    () => [...sortedPromptPhases, ...sortedBuiltInPhases],
    [sortedPromptPhases, sortedBuiltInPhases]
  );

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
    if (focusedIndex >= combinedPhases.length) {
      setFocusedIndex(Math.max(0, combinedPhases.length - 1));
    }
  }, [combinedPhases.length, focusedIndex]);

  const handleSelect = useCallback(
    (phaseName: string) => {
      const normalized = normalizeSteerMode(phaseName);
      trackUsage(normalized);
      onSelect(normalized);
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
      if (combinedPhases.length === 0) return;

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          // Move down one row (skip GRID_COLUMNS items)
          setFocusedIndex((prev) => Math.min(prev + GRID_COLUMNS, combinedPhases.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          // Move up one row (skip GRID_COLUMNS items)
          setFocusedIndex((prev) => Math.max(prev - GRID_COLUMNS, 0));
          break;
        case 'ArrowRight':
          e.preventDefault();
          // Move to next item
          setFocusedIndex((prev) => Math.min(prev + 1, combinedPhases.length - 1));
          break;
        case 'ArrowLeft':
          e.preventDefault();
          // Move to previous item
          setFocusedIndex((prev) => Math.max(prev - 1, 0));
          break;
        case 'Enter':
          e.preventDefault();
          if (combinedPhases[focusedIndex]) {
            handleSelect(combinedPhases[focusedIndex].name);
          }
          break;
        case 'Escape':
          e.preventDefault();
          onOpenChange(false);
          break;
      }
    },
    [combinedPhases, focusedIndex, handleSelect, onOpenChange]
  );

  // Scroll focused item into view
  useEffect(() => {
    if (!gridRef.current) return;
    const focusedElement = gridRef.current.querySelector(`[data-index="${focusedIndex}"]`);
    if (focusedElement) {
      focusedElement.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  }, [focusedIndex]);

  const promptTotal = phaseNames.length;
  const builtInTotal = BUILT_IN_PHASES.length;
  const promptFilteredEmpty = !promptIsLoading && !promptHasError && promptTotal > 0 && filteredPromptPhases.length === 0;
  const promptEmpty = !promptIsLoading && !promptHasError && promptTotal === 0;
  const promptUnavailable = promptHasError || promptEmpty;
  const builtInFilteredEmpty = filteredBuiltInPhases.length === 0 && search.trim().length > 0;
  const totalDisplayed = combinedPhases.length;

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
          <div className="space-y-6">
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-slate-500">
                  {promptUnavailable ? (
                    <CloudOff className="h-3.5 w-3.5 text-slate-500" />
                  ) : (
                    <Cloud className="h-3.5 w-3.5 text-emerald-400" />
                  )}
                  <span>Prompt-manager phases</span>
                </div>
                <div className="flex items-center gap-2">
                  {promptIsLoading && (
                    <span className="text-xs text-slate-500">Syncing...</span>
                  )}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => syncSkills.mutate()}
                    disabled={syncSkills.isPending}
                  >
                    {syncSkills.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>

              {promptIsLoading && !promptHasData ? (
                <div className="flex items-center justify-center h-32 text-sm text-slate-400 border border-dashed border-slate-700 rounded-lg">
                  Loading prompt-manager phases...
                </div>
              ) : promptHasError ? (
                <div className="flex flex-col items-center justify-center gap-2 h-32 text-sm text-slate-400 border border-dashed border-slate-700 rounded-lg">
                  <span>Prompt-manager unavailable. Sync to retry.</span>
                  <Button
                    type="button"
                    size="sm"
                    onClick={() => syncSkills.mutate()}
                    disabled={syncSkills.isPending}
                  >
                    {syncSkills.isPending ? 'Syncing...' : 'Sync prompt-manager'}
                  </Button>
                </div>
              ) : promptEmpty ? (
                <div className="flex flex-col items-center justify-center gap-2 h-32 text-sm text-slate-400 border border-dashed border-slate-700 rounded-lg">
                  <span>No prompt-manager phases synced yet.</span>
                  <Button
                    type="button"
                    size="sm"
                    onClick={() => syncSkills.mutate()}
                    disabled={syncSkills.isPending}
                  >
                    {syncSkills.isPending ? 'Syncing...' : 'Sync prompt-manager'}
                  </Button>
                </div>
              ) : promptFilteredEmpty ? (
                <div className="flex items-center justify-center h-24 text-sm text-slate-400 border border-dashed border-slate-700 rounded-lg">
                  No prompt-manager phases match "{search}"
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {sortedPromptPhases.map((phase, index) => (
                    <button
                      key={phase.name}
                      data-index={index}
                      type="button"
                      onClick={() => handleSelect(phase.name)}
                      className={cn(
                        'flex flex-col items-start p-3 rounded-lg border text-left transition-colors',
                        'hover:bg-slate-800 hover:border-slate-600',
                        normalizedValue === normalizeSteerMode(phase.name) && 'border-blue-500 bg-blue-500/10',
                        index === focusedIndex && 'ring-2 ring-blue-500 ring-offset-1 ring-offset-slate-900'
                      )}
                    >
                      <div className="flex items-center gap-2 w-full">
                        <span className="font-medium text-sm text-slate-100">
                          {formatPhaseName(phase.name)}
                        </span>
                        {normalizedValue === normalizeSteerMode(phase.name) && (
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

            <div className="space-y-3">
              <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-slate-500">
                <span>Built-in phases</span>
              </div>

              {builtInFilteredEmpty ? (
                <div className="flex items-center justify-center h-24 text-sm text-slate-400 border border-dashed border-slate-700 rounded-lg">
                  No built-in phases match "{search}"
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {sortedBuiltInPhases.map((phase, index) => {
                    const combinedIndex = sortedPromptPhases.length + index;
                    return (
                      <button
                        key={phase.name}
                        data-index={combinedIndex}
                        type="button"
                        onClick={() => handleSelect(phase.name)}
                        className={cn(
                          'flex flex-col items-start p-3 rounded-lg border text-left transition-colors',
                          'hover:bg-slate-800 hover:border-slate-600',
                          normalizedValue === normalizeSteerMode(phase.name) && 'border-blue-500 bg-blue-500/10',
                          combinedIndex === focusedIndex && 'ring-2 ring-blue-500 ring-offset-1 ring-offset-slate-900'
                        )}
                      >
                        <div className="flex items-center gap-2 w-full">
                          <span className="font-medium text-sm text-slate-100">
                            {formatPhaseName(phase.name)}
                          </span>
                          {normalizedValue === normalizeSteerMode(phase.name) && (
                            <Check className="h-4 w-4 text-blue-400 ml-auto shrink-0" />
                          )}
                        </div>
                        {phase.description && (
                          <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                            {phase.description}
                          </p>
                        )}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            {totalDisplayed === 0 && !promptIsLoading && !promptHasError && (
              <div className="flex flex-col items-center justify-center h-28 text-slate-400">
                <p>No phases found</p>
                {search && <p className="text-sm mt-1">Try a different search term</p>}
              </div>
            )}
          </div>
        </div>

        {/* Footer with count */}
        <div className="text-xs text-slate-500 mt-2 pt-2 border-t border-slate-700">
          {totalDisplayed} of {promptTotal + builtInTotal} phases
          {` • ${sortedPromptPhases.length} prompt-manager`}
          {` • ${sortedBuiltInPhases.length} built-in`}
          {search && ` • matching "${search}"`}
        </div>
      </DialogContent>
    </Dialog>
  );
}

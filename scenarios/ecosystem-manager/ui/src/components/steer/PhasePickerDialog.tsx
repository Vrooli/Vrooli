import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Check, Cloud, CloudOff, Loader2, RefreshCw, Search } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn, normalizeSkillId } from '@/lib/utils';
import { usePhaseUsage, type SortOption } from '@/hooks/usePhaseUsage';
import { useSteerSkills, useSyncSkills } from '@/hooks/useSkills';
import type { PhaseInfo } from '@/types/api';

interface PhasePickerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  values?: string[];
  onConfirm: (phaseIds: string[]) => void;
  phaseNames: PhaseInfo[];
  isLoading?: boolean;
  title?: string;
  description?: string;
  selectionMode?: 'single' | 'multiple';
  confirmLabel?: string;
}

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: 'name', label: 'Name (A-Z)' },
  { value: 'recent', label: 'Recent' },
  { value: 'most-used', label: 'Most Used' },
];

const BUILT_IN_PHASES: PhaseInfo[] = [
  { id: 'progress', name: 'Progress', description: 'Advance core objectives and operational targets.', modes: [], source: 'builtin' },
  { id: 'ux', name: 'UX', description: 'Improve usability, accessibility, and user flows.', modes: [], source: 'builtin' },
  { id: 'refactor', name: 'Refactor', description: 'Raise code quality without changing behavior.', modes: [], source: 'builtin' },
  { id: 'test', name: 'Test', description: 'Expand coverage and harden edge cases.', modes: [], source: 'builtin' },
  { id: 'explore', name: 'Explore', description: 'Explore options before committing to a path.', modes: [], source: 'builtin' },
  { id: 'polish', name: 'Polish', description: 'Finalize copy, visuals, and small fixes.', modes: [], source: 'builtin' },
  { id: 'performance', name: 'Performance', description: 'Profile and optimize slow paths.', modes: [], source: 'builtin' },
  { id: 'security', name: 'Security', description: 'Reduce vulnerabilities and tighten validation.', modes: [], source: 'builtin' },
];

export function prioritizeSelectedPhases(
  phases: PhaseInfo[],
  selectedSkillIds: string[],
): PhaseInfo[] {
  if (phases.length === 0 || selectedSkillIds.length === 0) {
    return phases;
  }

  const selected = new Set(selectedSkillIds.map((id) => normalizeSkillId(id)));
  return [...phases].sort((a, b) => {
    const aSelected = selected.has(normalizeSkillId(a.id)) ? 1 : 0;
    const bSelected = selected.has(normalizeSkillId(b.id)) ? 1 : 0;
    return bSelected - aSelected;
  });
}

export function PhasePickerDialog({
  open,
  onOpenChange,
  values,
  onConfirm,
  phaseNames,
  isLoading,
  title = 'Select Focus Modes',
  description = 'Choose one or more steering skills for this set.',
  selectionMode = 'single',
  confirmLabel = 'Confirm',
}: PhasePickerDialogProps) {
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState<SortOption>('recent');
  const [focusedIndex, setFocusedIndex] = useState(0);
  const [draftSelection, setDraftSelection] = useState<string[]>(values ?? []);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  const { data: steerSkills = [], isLoading: skillsLoading, isError: skillsError } = useSteerSkills();
  const syncSkills = useSyncSkills();
  const { trackUsage, sortByRecent, sortByFrequency, sortByName } = usePhaseUsage();
  const normalizedSelection = useMemo(
    () => draftSelection.map((id) => normalizeSkillId(id)).filter(Boolean),
    [draftSelection]
  );
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
        const modesLower = (phase.modes || []).join(' ').toLowerCase();
        return nameLower.includes(searchLower) || descLower.includes(searchLower) || modesLower.includes(searchLower);
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
    () => {
      const sorted = sortPhases(filteredPromptPhases);
      return prioritizeSelectedPhases(sorted, normalizedSelection);
    },
    [filteredPromptPhases, normalizedSelection, sortPhases]
  );
  const sortedBuiltInPhases = useMemo(
    () => {
      const sorted = sortPhases(filteredBuiltInPhases);
      return prioritizeSelectedPhases(sorted, normalizedSelection);
    },
    [filteredBuiltInPhases, normalizedSelection, sortPhases]
  );

  const combinedPhases = useMemo(
    () => [...sortedPromptPhases, ...sortedBuiltInPhases],
    [sortedPromptPhases, sortedBuiltInPhases]
  );

  useEffect(() => {
    if (open) {
      setSearch('');
      setFocusedIndex(0);
      setDraftSelection(values ?? []);
      setTimeout(() => searchInputRef.current?.focus(), 100);
    }
  }, [open, values]);

  useEffect(() => {
    if (focusedIndex >= combinedPhases.length) {
      setFocusedIndex(Math.max(0, combinedPhases.length - 1));
    }
  }, [combinedPhases.length, focusedIndex]);

  const toggleSelection = useCallback(
    (phaseId: string) => {
      const normalized = normalizeSkillId(phaseId);
      setDraftSelection((prev) => {
        if (selectionMode === 'single') {
          return [normalized];
        }
        const has = prev.some((id) => normalizeSkillId(id) === normalized);
        return has ? prev.filter((id) => normalizeSkillId(id) !== normalized) : [...prev, normalized];
      });
      trackUsage(normalized);
    },
    [selectionMode, trackUsage]
  );

  const handleConfirm = () => {
    onConfirm(normalizedSelection);
    onOpenChange(false);
  };

  const handleCancel = () => {
    setDraftSelection(values ?? []);
    onOpenChange(false);
  };

  const GRID_COLUMNS = 2;

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (combinedPhases.length === 0) return;

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setFocusedIndex((prev) => Math.min(prev + GRID_COLUMNS, combinedPhases.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setFocusedIndex((prev) => Math.max(prev - GRID_COLUMNS, 0));
          break;
        case 'ArrowRight':
          e.preventDefault();
          setFocusedIndex((prev) => Math.min(prev + 1, combinedPhases.length - 1));
          break;
        case 'ArrowLeft':
          e.preventDefault();
          setFocusedIndex((prev) => Math.max(prev - 1, 0));
          break;
        case 'Enter':
        case ' ':
          e.preventDefault();
          if (combinedPhases[focusedIndex]) {
            toggleSelection(combinedPhases[focusedIndex].id);
          }
          break;
        case 'Escape':
          e.preventDefault();
          onOpenChange(false);
          break;
      }
    },
    [combinedPhases, focusedIndex, onOpenChange, toggleSelection]
  );

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
  const hasSelection = normalizedSelection.length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <div className="flex gap-2 mt-2" onKeyDown={handleKeyDown}>
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <Input
              ref={searchInputRef}
              placeholder="Search skills..."
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

        <div
          ref={gridRef}
          className="flex-1 overflow-y-auto mt-4 min-h-[280px] max-h-[420px]"
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
                  <span>Prompt-manager skills</span>
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
                  Loading prompt-manager skills...
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
                  <span>No prompt-manager skills synced yet.</span>
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
                  No prompt-manager skills match "{search}"
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {sortedPromptPhases.map((phase, index) => {
                    const normalizedId = normalizeSkillId(phase.id);
                    const isSelected = normalizedSelection.includes(normalizedId);
                    return (
                      <button
                        key={`${phase.id}-${index}`}
                        data-index={index}
                        type="button"
                        onClick={() => toggleSelection(phase.id)}
                        className={cn(
                          'flex items-start gap-2 p-3 rounded-lg border text-left transition-colors',
                          'hover:bg-slate-800 hover:border-slate-600',
                          isSelected && 'border-blue-500 bg-blue-500/10',
                          index === focusedIndex && 'ring-2 ring-blue-500 ring-offset-1 ring-offset-slate-900'
                        )}
                      >
                        <Checkbox checked={isSelected} className="mt-0.5" />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 w-full">
                            <span className="font-medium text-sm text-slate-100 truncate">{phase.name}</span>
                            {isSelected && <Check className="h-4 w-4 text-blue-400 ml-auto shrink-0" />}
                          </div>
                          {phase.description && (
                            <p className="text-xs text-slate-400 mt-1 line-clamp-2">{phase.description}</p>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            <div className="space-y-3">
              <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-slate-500">
                <span>Built-in skills</span>
              </div>

              {builtInFilteredEmpty ? (
                <div className="flex items-center justify-center h-24 text-sm text-slate-400 border border-dashed border-slate-700 rounded-lg">
                  No built-in skills match "{search}"
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {sortedBuiltInPhases.map((phase, index) => {
                    const combinedIndex = sortedPromptPhases.length + index;
                    const normalizedId = normalizeSkillId(phase.id);
                    const isSelected = normalizedSelection.includes(normalizedId);
                    return (
                      <button
                        key={phase.id}
                        data-index={combinedIndex}
                        type="button"
                        onClick={() => toggleSelection(phase.id)}
                        className={cn(
                          'flex items-start gap-2 p-3 rounded-lg border text-left transition-colors',
                          'hover:bg-slate-800 hover:border-slate-600',
                          isSelected && 'border-blue-500 bg-blue-500/10',
                          combinedIndex === focusedIndex && 'ring-2 ring-blue-500 ring-offset-1 ring-offset-slate-900'
                        )}
                      >
                        <Checkbox checked={isSelected} className="mt-0.5" />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 w-full">
                            <span className="font-medium text-sm text-slate-100 truncate">{phase.name}</span>
                            {isSelected && <Check className="h-4 w-4 text-blue-400 ml-auto shrink-0" />}
                          </div>
                          {phase.description && (
                            <p className="text-xs text-slate-400 mt-1 line-clamp-2">{phase.description}</p>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            {totalDisplayed === 0 && !promptIsLoading && !promptHasError && (
              <div className="flex flex-col items-center justify-center h-28 text-slate-400">
                <p>No skills found</p>
                {search && <p className="text-sm mt-1">Try a different search term</p>}
              </div>
            )}
          </div>
        </div>

        <div className="text-xs text-slate-500 mt-2 pt-2 border-t border-slate-700">
          {totalDisplayed} of {promptTotal + builtInTotal} skills
          {` • ${normalizedSelection.length} selected`}
          {search && ` • matching "${search}"`}
        </div>

        <DialogFooter className="mt-2">
          <Button type="button" variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
          <Button type="button" onClick={handleConfirm} disabled={!hasSelection}>
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

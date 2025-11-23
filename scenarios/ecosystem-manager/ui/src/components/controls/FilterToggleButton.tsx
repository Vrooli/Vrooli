import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { Filter } from 'lucide-react';

export function FilterToggleButton() {
  const { isFilterPanelOpen, setIsFilterPanelOpen, filters } = useAppState();

  // Count active filters
  const activeFilterCount = [
    filters.search,
    filters.type,
    filters.operation,
    filters.priority,
  ].filter(Boolean).length;

  return (
    <Button
      variant={isFilterPanelOpen ? 'default' : 'outline'}
      size="sm"
      onClick={() => setIsFilterPanelOpen(!isFilterPanelOpen)}
      className="gap-2 relative"
      aria-label={`${isFilterPanelOpen ? 'Close' : 'Open'} filter panel${activeFilterCount > 0 ? ` (${activeFilterCount} active filters)` : ''}`}
    >
      <Filter className="h-4 w-4" />
      <span>Filters</span>
      {activeFilterCount > 0 && (
        <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center font-medium">
          {activeFilterCount}
        </span>
      )}
    </Button>
  );
}

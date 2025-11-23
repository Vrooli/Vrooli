import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { Filter } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';

export function FilterToggleButton() {
  const { isFilterPanelOpen, setIsFilterPanelOpen, filters } = useAppState();

  // Count active filters
  const activeFilterCount = [
    filters.search,
    filters.type,
    filters.operation,
    filters.priority,
  ].filter(Boolean).length;

  const label = `${isFilterPanelOpen ? 'Close' : 'Open'} filter panel${activeFilterCount > 0 ? ` (${activeFilterCount} active filters)` : ''}`;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant={isFilterPanelOpen ? 'default' : 'outline'}
          size="sm"
          onClick={() => setIsFilterPanelOpen(!isFilterPanelOpen)}
          className="relative px-3"
          aria-label={label}
        >
          <Filter className="h-4 w-4" />
          {activeFilterCount > 0 && (
            <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center font-medium">
              {activeFilterCount}
            </span>
          )}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top">Filters</TooltipContent>
    </Tooltip>
  );
}

import { useQueueStatus, useToggleProcessor } from '../../hooks/useQueueStatus';
import { Button } from '../ui/button';
import { Play, Square, Loader2 } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';

export function ProcessorStatusButton() {
  const { data: queueStatus = {}, isLoading } = useQueueStatus();
  const toggleProcessor = useToggleProcessor();

  const isActive = (queueStatus as any)?.active ?? false;
  const slotsUsed = (queueStatus as any)?.slots_used ?? 0;
  const maxSlots =
    (queueStatus as any)?.max_concurrent ??
    (queueStatus as any)?.max_slots ??
    5;

  const handleToggle = () => {
    const action = isActive ? 'stop' : 'start';
    toggleProcessor.mutate(action);
  };

  const label = isActive ? 'Stop task processor' : 'Start task processor';

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant={isActive ? 'default' : 'outline'}
          size="sm"
          onClick={handleToggle}
          disabled={toggleProcessor.isPending || isLoading}
          className="gap-1 px-3"
          aria-label={label}
        >
          {toggleProcessor.isPending || isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : isActive ? (
            <Square className="h-4 w-4" />
          ) : (
            <Play className="h-4 w-4" />
          )}
          {isActive && (
            <span className="text-[11px] leading-none px-1 py-0.5 rounded bg-white/15">
              {slotsUsed}/{maxSlots}
            </span>
          )}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top">{isActive ? 'Stop processor' : 'Start processor'}</TooltipContent>
    </Tooltip>
  );
}

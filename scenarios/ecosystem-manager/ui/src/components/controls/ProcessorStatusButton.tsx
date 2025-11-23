import { useQueueStatus, useToggleProcessor } from '../../hooks/useQueueStatus';
import { Button } from '../ui/button';
import { Play, Square, Loader2 } from 'lucide-react';

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

  if (isLoading) {
    return (
      <Button variant="outline" size="sm" disabled>
        <Loader2 className="h-4 w-4 animate-spin" />
      </Button>
    );
  }

  return (
    <Button
      variant={isActive ? 'default' : 'outline'}
      size="sm"
      onClick={handleToggle}
      disabled={toggleProcessor.isPending}
      className="gap-2"
      aria-label={isActive ? 'Stop task processor' : 'Start task processor'}
    >
      {toggleProcessor.isPending ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : isActive ? (
        <Square className="h-4 w-4" />
      ) : (
        <Play className="h-4 w-4" />
      )}
      <span className="font-medium">
        {isActive ? 'Active' : 'Paused'}
      </span>
      {isActive && (
        <span className="text-xs opacity-75">
          ({slotsUsed}/{maxSlots})
        </span>
      )}
    </Button>
  );
}

import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { Plus } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';

export function NewTaskButton() {
  const { setActiveModal } = useAppState();

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          size="sm"
          onClick={() => setActiveModal('create-task')}
          className="px-3"
          aria-label="Create new task"
        >
          <Plus className="h-4 w-4" />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top">New task</TooltipContent>
    </Tooltip>
  );
}

import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { ScrollText } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';

export function LogsButton() {
  const { setActiveModal } = useAppState();

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setActiveModal('system-logs')}
          className="px-3"
          aria-label="Open system logs"
        >
          <ScrollText className="h-4 w-4" />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top">System logs</TooltipContent>
    </Tooltip>
  );
}

import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { Settings } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';

export function SettingsButton() {
  const { setActiveModal } = useAppState();

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setActiveModal('settings')}
          className="px-3"
          aria-label="Open settings"
        >
          <Settings className="h-4 w-4" />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top">Settings</TooltipContent>
    </Tooltip>
  );
}

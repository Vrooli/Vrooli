import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { Settings } from 'lucide-react';

export function SettingsButton() {
  const { setActiveModal } = useAppState();

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => setActiveModal('settings')}
      className="gap-2"
    >
      <Settings className="h-4 w-4" />
      <span>Settings</span>
    </Button>
  );
}

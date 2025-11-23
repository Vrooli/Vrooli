import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { ScrollText } from 'lucide-react';

export function LogsButton() {
  const { setActiveModal } = useAppState();

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => setActiveModal('system-logs')}
      className="gap-2"
    >
      <ScrollText className="h-4 w-4" />
      <span>Logs</span>
    </Button>
  );
}

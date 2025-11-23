import { useAppState } from '../../contexts/AppStateContext';
import { Button } from '../ui/button';
import { Plus } from 'lucide-react';

export function NewTaskButton() {
  const { setActiveModal } = useAppState();

  return (
    <Button
      size="sm"
      onClick={() => setActiveModal('create-task')}
      className="gap-2"
    >
      <Plus className="h-4 w-4" />
      <span>New Task</span>
    </Button>
  );
}

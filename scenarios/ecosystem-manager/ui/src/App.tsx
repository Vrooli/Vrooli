import { useState } from "react";
import { useAppState } from "./contexts/AppStateContext";
import { KanbanBoard } from "./components/kanban/KanbanBoard";
import { CreateTaskModal } from "./components/modals/CreateTaskModal";
import { TaskDetailsModal } from "./components/modals/TaskDetailsModal";
import { SettingsModal } from "./components/modals/SettingsModal";
import { SystemLogsModal } from "./components/modals/SystemLogsModal";
import { FilterPanel } from "./components/filters/FilterPanel";
import { FloatingControls } from "./components/controls/FloatingControls";
import type { Task } from "./types/api";

export default function App() {
  const { activeModal, setActiveModal, isFilterPanelOpen } = useAppState();

  const [selectedTask, setSelectedTask] = useState<Task | null>(null);

  return (
    <div className="h-screen bg-background text-foreground flex flex-col">
      {/* Floating Controls */}
      <FloatingControls />

      {/* Main Content */}
      <main className="p-0 flex-1 min-h-0 overflow-hidden">
        <KanbanBoard
          onViewTaskDetails={(task) => setSelectedTask(task)}
          onDeleteTask={(taskId) => console.log('Delete task:', taskId)}
        />
      </main>

      {/* Filter Panel */}
      {isFilterPanelOpen && <FilterPanel />}

      {/* Create Task Modal */}
      <CreateTaskModal
        open={activeModal === 'create-task'}
        onOpenChange={(open) => setActiveModal(open ? 'create-task' : null)}
      />

      {/* Task Details Modal */}
      <TaskDetailsModal
        task={selectedTask}
        open={!!selectedTask}
        onOpenChange={(open) => !open && setSelectedTask(null)}
      />

      {/* Settings Modal */}
      <SettingsModal
        open={activeModal === 'settings'}
        onOpenChange={(open) => setActiveModal(open ? 'settings' : null)}
      />

      {/* System Logs Modal */}
      <SystemLogsModal
        open={activeModal === 'system-logs'}
        onOpenChange={(open) => setActiveModal(open ? 'system-logs' : null)}
      />
    </div>
  );
}

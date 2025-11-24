import { useState } from "react";
import { useAppState } from "./contexts/AppStateContext";
import { KanbanBoard } from "./components/kanban/KanbanBoard";
import { CreateTaskModal } from "./components/modals/CreateTaskModal";
import { TaskDetailsModal } from "./components/modals/TaskDetailsModal";
import { SettingsModal } from "./components/modals/SettingsModal";
import { SystemLogsModal } from "./components/modals/SystemLogsModal";
import { FilterPanel } from "./components/filters/FilterPanel";
import { FloatingControls } from "./components/controls/FloatingControls";
import { useDeleteTask } from "./hooks/useTaskMutations";
import type { Task } from "./types/api";
import { useEffect } from "react";
import { ensureDiscoveryLoaded } from "./stores/discoveryStore";

export default function App() {
  const { activeModal, setActiveModal, isFilterPanelOpen } = useAppState();

  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const deleteTask = useDeleteTask();

  const handleDeleteTask = (task: Task) => {
    const label = task.title || task.id;
    if (!confirm(`Delete task "${label}"? This cannot be undone.`)) {
      return;
    }

    deleteTask.mutate(task.id, {
      onSuccess: () => {
        if (selectedTask?.id === task.id) {
          setSelectedTask(null);
        }
      },
    });
  };

  useEffect(() => {
    ensureDiscoveryLoaded();
  }, []);

  return (
    <div className="h-screen bg-background text-foreground flex flex-col">
      {/* Floating Controls */}
      <FloatingControls onSelectTask={(task) => setSelectedTask(task)} />

      {/* Main Content */}
      <main className="p-0 flex-1 min-h-0 overflow-hidden">
        <KanbanBoard
          onViewTaskDetails={(task) => setSelectedTask(task)}
          onDeleteTask={handleDeleteTask}
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

import { useState } from "react";
import { Moon, Sun, Laptop } from "lucide-react";
import { Button } from "./components/ui/button";
import { useTheme } from "./contexts/ThemeContext";
import { useWebSocket } from "./contexts/WebSocketContext";
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
  const { theme, setTheme } = useTheme();
  const { isConnected: wsConnected } = useWebSocket();
  const { activeModal, setActiveModal, isFilterPanelOpen } = useAppState();

  const [selectedTask, setSelectedTask] = useState<Task | null>(null);

  const themeIcons = {
    light: Sun,
    dark: Moon,
    auto: Laptop,
  };

  const ThemeIcon = themeIcons[theme];

  const cycleTheme = () => {
    const themes: Array<'light' | 'dark' | 'auto'> = ['light', 'dark', 'auto'];
    const currentIndex = themes.indexOf(theme);
    const nextTheme = themes[(currentIndex + 1) % themes.length];
    setTheme(nextTheme);
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col">
      {/* Header */}
      <header className="sticky top-0 z-10 border-b border-white/10 bg-slate-900/95 backdrop-blur">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between px-4 sm:px-6 py-3 sm:py-4 gap-3 sm:gap-0">
          <div className="flex items-center gap-3 sm:gap-4 flex-wrap">
            <div>
              <h1 className="text-lg sm:text-xl font-semibold">Ecosystem Manager</h1>
              <p className="text-xs text-slate-400">
                Resource & Scenario Generation Platform
              </p>
            </div>
            {/* WebSocket Status Indicator */}
            <div className="flex items-center gap-2 px-2.5 sm:px-3 py-1 sm:py-1.5 rounded-full bg-black/20 border border-white/10">
              <div className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-green-400' : 'bg-red-400'}`} />
              <span className="text-xs text-slate-400">
                {wsConnected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
          </div>

          {/* Theme Toggle */}
          <Button
            variant="outline"
            size="sm"
            onClick={cycleTheme}
            className="w-9 h-9 p-0 absolute top-3 right-4 sm:relative sm:top-0 sm:right-0"
            title={`Theme: ${theme}`}
          >
            <ThemeIcon className="h-4 w-4" />
          </Button>
        </div>
      </header>

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

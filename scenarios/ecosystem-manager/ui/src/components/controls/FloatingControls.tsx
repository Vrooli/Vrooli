import { ProcessorStatusButton } from './ProcessorStatusButton';
import { FilterToggleButton } from './FilterToggleButton';
import { RefreshCountdown } from './RefreshCountdown';
import { ProcessMonitor } from './ProcessMonitor';
import { LogsButton } from './LogsButton';
import { SettingsButton } from './SettingsButton';
import { NewTaskButton } from './NewTaskButton';

export function FloatingControls() {
  return (
    <div className="fixed top-4 right-4 z-30 flex flex-col sm:flex-row items-stretch sm:items-center gap-2 bg-background/95 backdrop-blur-sm border rounded-lg shadow-lg px-3 py-2 max-w-[calc(100vw-2rem)]">
      {/* Left section: Status and monitoring */}
      <div className="flex items-center gap-2 sm:pr-2 sm:border-r justify-between sm:justify-start">
        <ProcessorStatusButton />
        <RefreshCountdown />
      </div>

      {/* Middle section: Filters and processes */}
      <div className="flex items-center gap-2 sm:pr-2 sm:border-r justify-between sm:justify-start border-t sm:border-t-0 pt-2 sm:pt-0">
        <FilterToggleButton />
        <ProcessMonitor />
      </div>

      {/* Right section: Actions */}
      <div className="flex items-center gap-2 justify-between sm:justify-start border-t sm:border-t-0 pt-2 sm:pt-0">
        <LogsButton />
        <SettingsButton />
        <NewTaskButton />
      </div>
    </div>
  );
}

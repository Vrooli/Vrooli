import { useState } from 'react';
import { useRunningProcesses } from '../../hooks/useRunningProcesses';
import { Button } from '../ui/button';
import { ProcessCard } from './ProcessCard';
import { Activity, ChevronDown } from 'lucide-react';

export function ProcessMonitor() {
  const { data: processes = [], isLoading } = useRunningProcesses();
  const [isOpen, setIsOpen] = useState(false);

  const processCount = processes.length;

  return (
    <div className="relative">
      <Button
        variant="outline"
        size="sm"
        onClick={() => setIsOpen(!isOpen)}
        className="gap-2"
        disabled={isLoading}
      >
        <Activity className="h-4 w-4" />
        <span>Processes</span>
        {processCount > 0 && (
          <span className="px-1.5 py-0.5 rounded-full bg-primary text-primary-foreground text-xs font-medium">
            {processCount}
          </span>
        )}
        <ChevronDown className={`h-3 w-3 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </Button>

      {isOpen && (
        <>
          {/* Backdrop to close dropdown */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Dropdown panel */}
          <div className="absolute right-0 top-full mt-2 w-80 bg-background border rounded-lg shadow-lg z-50 max-h-96 overflow-y-auto">
            <div className="p-3 border-b">
              <h3 className="font-semibold text-sm">Running Processes</h3>
              <p className="text-xs text-muted-foreground mt-0.5">
                {processCount} active {processCount === 1 ? 'process' : 'processes'}
              </p>
            </div>

            <div className="p-2 space-y-2">
              {processCount === 0 ? (
                <div className="text-center py-8 text-sm text-muted-foreground">
                  No processes running
                </div>
              ) : (
                processes.map((process) => (
                  <ProcessCard key={process.process_id} process={process} />
                ))
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

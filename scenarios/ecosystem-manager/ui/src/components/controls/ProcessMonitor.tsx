import { useState } from 'react';
import { useRunningProcesses } from '../../hooks/useRunningProcesses';
import { Button } from '../ui/button';
import { ProcessCard } from './ProcessCard';
import { Activity, ChevronDown } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';

export function ProcessMonitor() {
  const { data: processes = [], isLoading } = useRunningProcesses();
  const [isOpen, setIsOpen] = useState(false);

  const processCount = processes.length;

  return (
    <div className="relative">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsOpen(!isOpen)}
            className="relative px-3"
            disabled={isLoading}
            aria-label={`View running processes${processCount > 0 ? ` (${processCount})` : ''}`}
          >
            <Activity className="h-4 w-4" />
            {processCount > 0 && (
              <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center font-medium">
                {processCount}
              </span>
            )}
            <ChevronDown className={`h-3 w-3 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="top">Running processes</TooltipContent>
      </Tooltip>

      {isOpen && (
        <>
          {/* Backdrop to close dropdown */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Dropdown panel */}
          <div className="absolute right-0 bottom-full mb-2 w-80 bg-background border rounded-lg shadow-lg z-50 max-h-96 overflow-y-auto">
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

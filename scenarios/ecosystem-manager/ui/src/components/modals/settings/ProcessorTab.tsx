/**
 * ProcessorTab Component
 * Settings for queue processor configuration
 */

import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Checkbox } from '@/components/ui/checkbox';
import type { ProcessorSettings } from '@/types/api';

interface ProcessorTabProps {
  settings: ProcessorSettings;
  onChange: (updates: Partial<ProcessorSettings>) => void;
}

export function ProcessorTab({ settings, onChange }: ProcessorTabProps) {
  return (
    <div className="space-y-6">
      {/* Processor Active Toggle */}
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <Checkbox
            id="processor-active"
            checked={settings.active}
            onCheckedChange={(checked) => onChange({ active: !!checked })}
          />
          <Label htmlFor="processor-active" className="cursor-pointer font-medium">
            Enable Processor
          </Label>
        </div>
        <p className="text-xs text-slate-500 ml-8">
          When enabled, the processor will automatically pick up and execute queued tasks
        </p>
      </div>

      {/* Concurrent Slots */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Label htmlFor="concurrent-slots">Concurrent Task Slots</Label>
          <span className="text-sm font-medium text-slate-300">
            {settings.concurrent_slots}
          </span>
        </div>
        <Slider
          id="concurrent-slots"
          min={1}
          max={5}
          step={1}
          value={[settings.concurrent_slots]}
          onValueChange={(value) => onChange({ concurrent_slots: value[0] })}
          className="w-full"
        />
        <p className="text-xs text-slate-500">
          Maximum number of tasks that can run simultaneously. Higher values increase throughput but consume more resources.
        </p>
      </div>

      {/* Refresh Interval */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Label htmlFor="refresh-interval">Refresh Interval</Label>
          <span className="text-sm font-medium text-slate-300">
            {settings.refresh_interval}s
          </span>
        </div>
        <Slider
          id="refresh-interval"
          min={5}
          max={300}
          step={5}
          value={[settings.refresh_interval]}
          onValueChange={(value) => onChange({ refresh_interval: value[0] })}
          className="w-full"
        />
        <p className="text-xs text-slate-500">
          How often the processor checks for new tasks in the queue (in seconds). Lower values provide faster response but increase system load.
        </p>
      </div>

      {/* Max Tasks (Disabled) */}
      <div className="space-y-3 opacity-50">
        <div className="flex items-center justify-between">
          <Label className="cursor-not-allowed">Max Tasks</Label>
          <span className="text-sm font-medium text-slate-500">
            Unlimited
          </span>
        </div>
        <Slider
          disabled
          min={0}
          max={100}
          step={1}
          value={[0]}
          className="w-full cursor-not-allowed"
        />
        <p className="text-xs text-slate-500">
          This setting is disabled to prevent queue starvation. The processor will continue processing all queued tasks.
        </p>
      </div>

      {/* Info Box */}
      <div className="rounded-lg border border-blue-500/20 bg-blue-500/10 p-4">
        <h4 className="text-sm font-medium text-blue-400 mb-2">Processor Behavior</h4>
        <ul className="text-xs text-slate-400 space-y-1">
          <li>• Tasks are executed in priority order (critical → high → medium → low)</li>
          <li>• The processor respects rate limits and automatically pauses when needed</li>
          <li>• Failed tasks can be retried manually or via recycler</li>
          <li>• Processor status updates in real-time via WebSocket</li>
        </ul>
      </div>
    </div>
  );
}

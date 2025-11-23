/**
 * AgentTab Component
 * Settings for Claude Code agent configuration
 */

import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Slider } from '@/components/ui/slider';
import { Checkbox } from '@/components/ui/checkbox';
import type { AgentSettings } from '@/types/api';

interface AgentTabProps {
  settings: AgentSettings;
  onChange: (updates: Partial<AgentSettings>) => void;
}

export function AgentTab({ settings, onChange }: AgentTabProps) {
  return (
    <div className="space-y-6">
      {/* Maximum Turns */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Label htmlFor="max-turns">Maximum Turns</Label>
          <span className="text-sm font-medium text-slate-300">
            {settings.max_turns}
          </span>
        </div>
        <Slider
          id="max-turns"
          min={5}
          max={100}
          step={1}
          value={[settings.max_turns]}
          onValueChange={(value) => onChange({ max_turns: value[0] })}
          className="w-full"
        />
        <p className="text-xs text-slate-500">
          Maximum number of tool calls the agent can make per task. Higher values allow more complex operations but increase token usage.
        </p>
      </div>

      {/* Task Timeout */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Label htmlFor="task-timeout">Task Timeout</Label>
          <span className="text-sm font-medium text-slate-300">
            {settings.task_timeout_minutes} min
          </span>
        </div>
        <Slider
          id="task-timeout"
          min={5}
          max={240}
          step={5}
          value={[settings.task_timeout_minutes]}
          onValueChange={(value) => onChange({ task_timeout_minutes: value[0] })}
          className="w-full"
        />
        <p className="text-xs text-slate-500">
          Maximum time a task can run before being automatically terminated. Prevents runaway processes.
        </p>
      </div>

      {/* Idle Timeout Cap */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Label htmlFor="idle-timeout">Idle Timeout Cap</Label>
          <span className="text-sm font-medium text-slate-300">
            {settings.idle_timeout_cap_minutes} min
          </span>
        </div>
        <Slider
          id="idle-timeout"
          min={2}
          max={240}
          step={2}
          value={[settings.idle_timeout_cap_minutes]}
          onValueChange={(value) => onChange({ idle_timeout_cap_minutes: value[0] })}
          className="w-full"
        />
        <p className="text-xs text-slate-500">
          Maximum idle time before the agent session is considered stale. Lower values help free up resources faster.
        </p>
      </div>

      {/* Allowed Tools */}
      <div className="space-y-3">
        <Label htmlFor="allowed-tools">Allowed Tools</Label>
        <Input
          id="allowed-tools"
          type="text"
          value={settings.allowed_tools || ''}
          onChange={(e) => onChange({ allowed_tools: e.target.value })}
          placeholder="Read,Write,Edit,Bash,Glob,Grep"
        />
        <p className="text-xs text-slate-500">
          Comma-separated list of tools the agent can use. Leave empty to allow all tools.
        </p>
      </div>

      {/* Skip Permissions */}
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <Checkbox
            id="skip-permissions"
            checked={settings.skip_permissions}
            onCheckedChange={(checked) => onChange({ skip_permissions: !!checked })}
          />
          <Label htmlFor="skip-permissions" className="cursor-pointer font-medium">
            Skip Permissions Prompts
          </Label>
        </div>
        <p className="text-xs text-slate-500 ml-8">
          When enabled, the agent will not prompt for permission before executing certain operations. Use with caution.
        </p>
      </div>

      {/* Info Box */}
      <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4">
        <h4 className="text-sm font-medium text-amber-400 mb-2">Agent Configuration Tips</h4>
        <ul className="text-xs text-slate-400 space-y-1">
          <li>• Higher max turns allow more complex multi-step operations</li>
          <li>• Task timeout should be generous for resource-intensive operations</li>
          <li>• Restricting tools can improve safety but may limit agent capabilities</li>
          <li>• Skip permissions is recommended for autonomous batch processing</li>
        </ul>
      </div>
    </div>
  );
}

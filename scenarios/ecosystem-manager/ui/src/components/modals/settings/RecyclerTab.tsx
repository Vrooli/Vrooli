/**
 * RecyclerTab Component
 * Configure recycler settings and run tests
 */

import { Info } from 'lucide-react';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Slider } from '@/components/ui/slider';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { RecyclerSettings } from '@/types/api';

const MODEL_PROVIDERS = [
  { value: 'ollama', label: 'Ollama (Local)' },
  { value: 'openrouter', label: 'OpenRouter (Cloud)' },
];

const OLLAMA_MODELS = [
  'llama3.2:1b',
  'llama3.2:3b',
  'phi3:mini',
  'qwen2.5:0.5b',
  'qwen2.5:1.5b',
];

const OPENROUTER_MODELS = [
  'anthropic/claude-3-haiku',
  'google/gemini-flash-1.5',
  'meta-llama/llama-3.2-3b-instruct',
];

interface RecyclerTabProps {
  settings: RecyclerSettings;
  onChange: (updates: Partial<RecyclerSettings>) => void;
}

export function RecyclerTab({ settings, onChange }: RecyclerTabProps) {
  const updateSetting = <K extends keyof RecyclerSettings>(
    key: K,
    value: RecyclerSettings[K]
  ) => {
    onChange({ [key]: value });
  };

  const modelOptions =
    settings.model_provider === 'ollama' ? OLLAMA_MODELS : OPENROUTER_MODELS;

  return (
    <div className="space-y-6">
      {/* Info Banner */}
      <div className="bg-blue-900/20 border border-blue-400/30 rounded-md p-4 flex gap-3">
        <Info className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
        <div className="text-xs text-slate-300 space-y-1">
          <p>
            <strong>Recycler</strong> refreshes notes for completed/failed tasks and uses scenario
            completeness scores to decide whether they should be automatically re-queued for
            another improvement pass. Processing is event-driven with a periodic sweep as a safety
            net.
          </p>
          <p>
            It pairs a lightweight LLM summary with `vrooli scenario completeness &lt;scenario&gt;`
            so recycling decisions come from the same scoring pipeline used across scenarios. Retry
            and backoff controls keep recycling from thrashing when a task is unhealthy.
          </p>
        </div>
      </div>

      {/* Settings Form */}
      <div className="space-y-4">
        {/* Enabled For */}
        <div className="space-y-2">
          <Label htmlFor="recycler-enabled">Enabled For</Label>
          <Select
            value={settings.enabled_for}
            onValueChange={(val) => updateSetting('enabled_for', val as 'off' | 'resources' | 'scenarios' | 'both')}
          >
            <SelectTrigger id="recycler-enabled">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="off">Off</SelectItem>
              <SelectItem value="resources">Resources Only</SelectItem>
              <SelectItem value="scenarios">Scenarios Only</SelectItem>
              <SelectItem value="both">Both</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Sweep Interval */}
        <div className="space-y-2">
          <Label htmlFor="recycler-interval">
            Sweep Interval (seconds): {settings.recycle_interval}
          </Label>
          <Slider
            id="recycler-interval"
            min={30}
            max={1800}
            step={30}
            value={[settings.recycle_interval]}
            onValueChange={([val]) => updateSetting('recycle_interval', val)}
          />
          <p className="text-xs text-slate-400">
            Backstop sweep to catch manual moves; event-driven triggers run immediately.
          </p>
        </div>

        {/* Retry / Backoff */}
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="recycler-max-retries">
              Max Retries: {settings.max_retries}
            </Label>
            <Slider
              id="recycler-max-retries"
              min={0}
              max={10}
              step={1}
              value={[settings.max_retries]}
              onValueChange={([val]) => updateSetting('max_retries', val)}
            />
            <p className="text-xs text-slate-400">
              Retries when recycler processing fails (0 = no retries)
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="recycler-retry-delay">
              Retry Delay (seconds): {settings.retry_delay_seconds}
            </Label>
            <Slider
              id="recycler-retry-delay"
              min={1}
              max={60}
              step={1}
              value={[settings.retry_delay_seconds]}
              onValueChange={([val]) => updateSetting('retry_delay_seconds', val)}
            />
            <p className="text-xs text-slate-400">
              Linear backoff per attempt (delay × attempt)
            </p>
          </div>
        </div>

        {/* Model Provider */}
        <div className="space-y-2">
          <Label htmlFor="recycler-provider">Model Provider</Label>
          <Select
            value={settings.model_provider}
            onValueChange={(val) => updateSetting('model_provider', val as 'ollama' | 'openrouter')}
          >
            <SelectTrigger id="recycler-provider">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MODEL_PROVIDERS.map((provider) => (
                <SelectItem key={provider.value} value={provider.value}>
                  {provider.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Model Name */}
        <div className="space-y-2">
          <Label htmlFor="recycler-model">Model Name</Label>
          <Select
            value={settings.model_name}
            onValueChange={(val) => updateSetting('model_name', val)}
          >
            <SelectTrigger id="recycler-model">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {modelOptions.map((model) => (
                <SelectItem key={model} value={model}>
                  {model}
                </SelectItem>
              ))}
              <SelectItem value="custom">Custom...</SelectItem>
            </SelectContent>
          </Select>
          {settings.model_name === 'custom' && (
            <Input
              placeholder="Enter custom model name"
              value={settings.model_name}
              onChange={(e) => updateSetting('model_name', e.target.value)}
            />
          )}
        </div>

        {/* Thresholds */}
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="completion-threshold">
              Completion Threshold: {settings.completion_threshold}
            </Label>
            <Slider
              id="completion-threshold"
              min={1}
              max={10}
              step={1}
              value={[settings.completion_threshold]}
              onValueChange={([val]) => updateSetting('completion_threshold', val)}
            />
            <p className="text-xs text-slate-400">
              Min score (1-10) to consider output complete
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="failure-threshold">
              Failure Threshold: {settings.failure_threshold}
            </Label>
            <Slider
              id="failure-threshold"
              min={1}
              max={10}
              step={1}
              value={[settings.failure_threshold]}
              onValueChange={([val]) => updateSetting('failure_threshold', val)}
            />
            <p className="text-xs text-slate-400">
              Max score (1-10) to retry failed tasks
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

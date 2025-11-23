/**
 * RecyclerTab Component
 * Configure recycler settings and run tests
 */

import { useState } from 'react';
import { Recycle, Play, Info } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Slider } from '@/components/ui/slider';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useRecyclerSettings, useRecyclerTest } from '@/hooks/useRecycler';
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

export function RecyclerTab() {
  const { settings, updateSetting } = useRecyclerSettings();
  const { mutate: runTest, data: testResults, isPending: isTesting } = useRecyclerTest();

  const [testOutput, setTestOutput] = useState('');

  if (!settings) {
    return <div className="text-slate-400 text-center py-8">Loading recycler settings...</div>;
  }

  const modelOptions =
    settings.model_provider === 'ollama' ? OLLAMA_MODELS : OPENROUTER_MODELS;

  const handleRunTest = () => {
    if (!testOutput.trim()) return;
    runTest({ output: testOutput });
  };

  return (
    <div className="space-y-6">
      {/* Info Banner */}
      <div className="bg-blue-900/20 border border-blue-400/30 rounded-md p-4 flex gap-3">
        <Info className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
        <div className="text-xs text-slate-300 space-y-1">
          <p>
            <strong>Recycler</strong> analyzes completed/failed task outputs to determine if they
            should be automatically re-queued for improvement.
          </p>
          <p>
            It uses a lightweight LLM to assess output quality and make intelligent decisions
            about task lifecycle management.
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

        {/* Recycle Interval */}
        <div className="space-y-2">
          <Label htmlFor="recycler-interval">
            Recycle Interval (seconds): {settings.recycle_interval}
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
            How often to check completed/failed tasks for recycling
          </p>
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

      {/* Recycler Testbed */}
      <div className="pt-6 border-t border-white/10 space-y-4">
        <div className="flex items-center gap-2">
          <Recycle className="h-4 w-4 text-slate-400" />
          <h4 className="text-sm font-medium">Recycler Testbed</h4>
        </div>

        <div className="space-y-2">
          <Label htmlFor="test-output">Test Output</Label>
          <Textarea
            id="test-output"
            value={testOutput}
            onChange={(e) => setTestOutput(e.target.value)}
            placeholder="Paste task output to test recycler analysis..."
            rows={6}
          />
        </div>

        <Button
          onClick={handleRunTest}
          disabled={isTesting || !testOutput.trim()}
        >
          {isTesting ? (
            'Analyzing...'
          ) : (
            <>
              <Play className="h-4 w-4 mr-2" />
              Run Test
            </>
          )}
        </Button>

        {/* Test Results */}
        {testResults && (
          <div className="bg-slate-900 border border-white/10 rounded-md p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Analysis Result</span>
              <span className={`text-sm font-semibold ${
                testResults.should_recycle ? 'text-yellow-400' : 'text-green-400'
              }`}>
                {testResults.should_recycle ? 'RECYCLE' : 'FINALIZE'}
              </span>
            </div>

            <div className="grid grid-cols-2 gap-4 text-xs">
              <div>
                <div className="text-slate-400">Completion Score</div>
                <div className="text-lg font-mono font-semibold">
                  {testResults.completion_score}/10
                </div>
              </div>
              <div>
                <div className="text-slate-400">Confidence</div>
                <div className="text-lg font-mono font-semibold">
                  {Math.round((testResults.confidence || 0) * 100)}%
                </div>
              </div>
            </div>

            {testResults.reasoning && (
              <div className="pt-2 border-t border-white/10">
                <div className="text-xs text-slate-400 mb-1">Reasoning</div>
                <div className="text-sm text-slate-300">{testResults.reasoning}</div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

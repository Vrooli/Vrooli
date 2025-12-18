import { Clock } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { RangeSlider } from '@shared/ui';
import { SettingSection, ToggleSwitch } from './shared';

export function WorkflowSection() {
  const { workflowDefaults, setWorkflowDefault } = useSettingsStore();

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-6">
        <Clock size={24} className="text-green-400" />
        <div>
          <h2 className="text-lg font-semibold text-surface">Workflow Defaults</h2>
          <p className="text-sm text-gray-400">Default settings applied to new workflows</p>
        </div>
      </div>

      <SettingSection title="Timeouts" tooltip="Control how long workflows and steps wait before timing out.">
        <div className="space-y-5">
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Workflow Timeout</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.defaultTimeout}s</span>
            </div>
            <RangeSlider
              min={10}
              max={300}
              step={5}
              value={workflowDefaults.defaultTimeout}
              onChange={(next) => setWorkflowDefault('defaultTimeout', next)}
              ariaLabel="Workflow timeout"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>10s</span>
              <span>5 min</span>
            </div>
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Step Timeout</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.stepTimeout}s</span>
            </div>
            <RangeSlider
              min={5}
              max={60}
              step={1}
              value={workflowDefaults.stepTimeout}
              onChange={(next) => setWorkflowDefault('stepTimeout', next)}
              ariaLabel="Step timeout"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>5s</span>
              <span>60s</span>
            </div>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Retry Behavior" tooltip="Configure automatic retry behavior for failed steps.">
        <div className="space-y-5">
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Retry Attempts</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.retryAttempts}</span>
            </div>
            <RangeSlider
              min={0}
              max={5}
              step={1}
              value={workflowDefaults.retryAttempts}
              onChange={(next) => setWorkflowDefault('retryAttempts', next)}
              ariaLabel="Retry attempts"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>No retries</span>
              <span>5 retries</span>
            </div>
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Retry Delay</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.retryDelay}ms</span>
            </div>
            <RangeSlider
              min={100}
              max={5000}
              step={100}
              value={workflowDefaults.retryDelay}
              onChange={(next) => setWorkflowDefault('retryDelay', next)}
              ariaLabel="Retry delay"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>100ms</span>
              <span>5s</span>
            </div>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Screenshots" tooltip="Configure when to capture screenshots during execution.">
        <div className="space-y-3">
          <ToggleSwitch
            checked={workflowDefaults.screenshotOnFailure}
            onChange={(v) => setWorkflowDefault('screenshotOnFailure', v)}
            label="Capture on Failure"
            description="Take a screenshot when a step fails"
          />
          <ToggleSwitch
            checked={workflowDefaults.screenshotOnSuccess}
            onChange={(v) => setWorkflowDefault('screenshotOnSuccess', v)}
            label="Capture on Success"
            description="Take a screenshot after each successful step"
            className="border-t border-gray-800"
          />
        </div>
      </SettingSection>

      <SettingSection title="Browser Options" tooltip="Control browser behavior during execution.">
        <div className="space-y-5">
          <ToggleSwitch
            checked={workflowDefaults.headless}
            onChange={(v) => setWorkflowDefault('headless', v)}
            label="Headless Mode"
            description="Run browser without visible window"
          />
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Slow Motion</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.slowMo}ms</span>
            </div>
            <RangeSlider
              min={0}
              max={1000}
              step={50}
              value={workflowDefaults.slowMo}
              onChange={(next) => setWorkflowDefault('slowMo', next)}
              ariaLabel="Slow motion delay"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Normal</span>
              <span>Very Slow (1s)</span>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Add a delay between each action for debugging
            </p>
          </div>
        </div>
      </SettingSection>
    </div>
  );
}

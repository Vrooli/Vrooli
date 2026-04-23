/**
 * Execution settings tab - Execution defaults, governance, and agent behavior.
 */

import { Card } from "../ui/card";
import { Input } from "../ui/input";
import { selectors } from "../../consts/selectors";
import { DEFAULT_SETTINGS } from "../../services/settings-service";
import type { Settings } from "../../types";
import { ToggleButtons } from "./ToggleButtons";

export interface ExecutionTabProps {
  form: Settings;
  patch: (updates: Partial<Settings>) => void;
}

export function ExecutionTab({ form, patch }: ExecutionTabProps) {
  return (
    <div className="space-y-6">
      {/* Execution Defaults */}
      <Card data-testid={selectors.settings.executionDefaults}>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Execution Defaults</h3>
            <p className="mt-1 text-sm text-slate-400">Default mode used when queue requests omit explicit values.</p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            defaultMode: DEFAULT_SETTINGS.defaultMode,
            autoFixup: DEFAULT_SETTINGS.autoFixup,
            maxFixupAttempts: DEFAULT_SETTINGS.maxFixupAttempts,
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Default Mode</label>
            <ToggleButtons
              value={form.defaultMode}
              options={[
                { value: "manual" as const, label: "manual" },
                { value: "yolo" as const, label: "yolo" },
              ]}
              onChange={(v) => patch({ defaultMode: v })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Auto-Fixup</label>
            <p className="mt-1 text-xs text-slate-400">When enabled, automatically re-runs execution when review finds issues.</p>
            <ToggleButtons
              value={form.autoFixup}
              options={[
                { value: false as const, label: "Disabled" },
                { value: true as const, label: "Enabled" },
              ]}
              onChange={(v) => patch({ autoFixup: v })}
            />
          </div>
          {form.autoFixup && (
            <div>
              <label className="block text-sm font-medium text-slate-300">Max Fixup Attempts</label>
              <p className="mt-1 text-xs text-slate-400">Maximum number of automatic fix-up attempts (0-5).</p>
              <Input
                type="number"
                min={0}
                max={5}
                className="mt-1"
                value={form.maxFixupAttempts}
                onChange={(e) => patch({ maxFixupAttempts: Math.max(0, Math.min(5, Number(e.target.value || 0))) })}
              />
            </div>
          )}
        </div>
      </Card>

      {/* Governance */}
      <Card>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Governance</h3>
            <p className="mt-1 text-sm text-slate-400">Concurrency limits, queue depth, circuit breaker, and cost controls.</p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            maxConcurrentExecutions: DEFAULT_SETTINGS.maxConcurrentExecutions,
            maxQueueDepth: DEFAULT_SETTINGS.maxQueueDepth,
            circuitBreakerThreshold: DEFAULT_SETTINGS.circuitBreakerThreshold,
            circuitBreakerCooldownMinutes: DEFAULT_SETTINGS.circuitBreakerCooldownMinutes,
            executionCostCapPerRun: DEFAULT_SETTINGS.executionCostCapPerRun,
            costPerTurnEstimate: DEFAULT_SETTINGS.costPerTurnEstimate,
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Max Concurrent Executions</label>
            <p className="mt-1 text-xs text-slate-400">Maximum simultaneous agent runs (1-20).</p>
            <Input
              type="number"
              min={1}
              max={20}
              className="mt-1"
              value={form.maxConcurrentExecutions}
              onChange={(e) => patch({ maxConcurrentExecutions: Math.max(1, Math.min(20, Number(e.target.value || 1))) })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Max Queue Depth</label>
            <p className="mt-1 text-xs text-slate-400">Maximum pending items in queue (0 = unlimited, max 100).</p>
            <Input
              type="number"
              min={0}
              max={100}
              className="mt-1"
              value={form.maxQueueDepth}
              onChange={(e) => patch({ maxQueueDepth: Math.max(0, Math.min(100, Number(e.target.value || 0))) })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Circuit Breaker Threshold</label>
            <p className="mt-1 text-xs text-slate-400">Consecutive failures before circuit trips (1-10).</p>
            <Input
              type="number"
              min={1}
              max={10}
              className="mt-1"
              value={form.circuitBreakerThreshold}
              onChange={(e) => patch({ circuitBreakerThreshold: Math.max(1, Math.min(10, Number(e.target.value || 1))) })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Circuit Breaker Cooldown (minutes)</label>
            <p className="mt-1 text-xs text-slate-400">Time before auto-reset after trip (5-1440).</p>
            <Input
              type="number"
              min={5}
              max={1440}
              className="mt-1"
              value={form.circuitBreakerCooldownMinutes}
              onChange={(e) => patch({ circuitBreakerCooldownMinutes: Math.max(5, Math.min(1440, Number(e.target.value || 5))) })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Cost Cap Per Run ($)</label>
            <p className="mt-1 text-xs text-slate-400">Estimated cost cap per execution (0 = unlimited).</p>
            <Input
              type="number"
              min={0}
              step={0.5}
              className="mt-1"
              value={form.executionCostCapPerRun}
              onChange={(e) => patch({ executionCostCapPerRun: Math.max(0, Number(e.target.value || 0)) })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Cost Per Turn Estimate ($)</label>
            <p className="mt-1 text-xs text-slate-400">Estimated cost per agent turn for cap calculation (0.00-5.00).</p>
            <Input
              type="number"
              min={0}
              max={5}
              step={0.01}
              className="mt-1"
              value={form.costPerTurnEstimate}
              onChange={(e) => patch({ costPerTurnEstimate: Math.max(0, Math.min(5, Number(e.target.value || 0))) })}
            />
          </div>
        </div>
      </Card>

      {/* Agent Behavior */}
      <Card data-testid={selectors.settings.agentSettings}>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Agent Behavior</h3>
            <p className="mt-1 text-sm text-slate-400">Controls for spawned agent runs.</p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            agentMaxTurns: DEFAULT_SETTINGS.agentMaxTurns,
            agentTimeoutSeconds: DEFAULT_SETTINGS.agentTimeoutSeconds,
            agentRequiresApproval: DEFAULT_SETTINGS.agentRequiresApproval,
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Max Turns</label>
            <p className="mt-1 text-xs text-slate-400">Maximum conversation turns per agent run (5-1000).</p>
            <Input
              type="number"
              min={5}
              max={1000}
              className="mt-1"
              value={form.agentMaxTurns}
              onChange={(e) => patch({ agentMaxTurns: Math.max(5, Math.min(1000, Number(e.target.value || 5))) })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300">Timeout (minutes)</label>
            <p className="mt-1 text-xs text-slate-400">Maximum wall-clock time before automatic cancellation (1-60).</p>
            <Input
              type="number"
              min={1}
              max={60}
              className="mt-1"
              value={Math.round(form.agentTimeoutSeconds / 60)}
              onChange={(e) => patch({ agentTimeoutSeconds: Math.max(60, Math.min(3600, Number(e.target.value || 1) * 60)) })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Require Approval</label>
            <p className="mt-1 text-xs text-slate-400">Pause agent runs for human approval before execution.</p>
            <ToggleButtons
              value={form.agentRequiresApproval}
              options={[
                { value: false as const, label: "Disabled" },
                { value: true as const, label: "Enabled" },
              ]}
              onChange={(v) => patch({ agentRequiresApproval: v })}
            />
          </div>
        </div>
      </Card>
    </div>
  );
}

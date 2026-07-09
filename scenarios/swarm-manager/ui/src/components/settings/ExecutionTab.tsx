/**
 * Execution settings tab - Execution defaults, governance, and agent behavior.
 */

import { useCallback, useEffect, useState } from "react";
import type { AutoFilerStatusResponse } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import { RefreshCw } from "lucide-react";
import { Card } from "../ui/card";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { selectors } from "../../consts/selectors";
import { DEFAULT_SETTINGS } from "../../services/settings-service";
import { autoFilerService } from "../../services";
import type { Settings } from "../../types";
import { ToggleButtons } from "./ToggleButtons";
import { GoalDrainToggle } from "./GoalDrainToggle";

export interface ExecutionTabProps {
  form: Settings;
  patch: (updates: Partial<Settings>) => void;
}

function formatAutoFilerTime(value: string): string {
  if (!value) return "Not run yet";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function ExecutionTab({ form, patch }: ExecutionTabProps) {
  const [autoFilerStatus, setAutoFilerStatus] = useState<AutoFilerStatusResponse | null>(null);
  const [autoFilerStatusError, setAutoFilerStatusError] = useState<string | null>(null);
  const [autoFilerRunPending, setAutoFilerRunPending] = useState(false);

  const loadAutoFilerStatus = useCallback(async (active: () => boolean = () => true) => {
    setAutoFilerStatusError(null);
    try {
      const status = await autoFilerService.getStatus();
      if (active()) setAutoFilerStatus(status);
    } catch (error) {
      if (active()) {
        setAutoFilerStatus(null);
        setAutoFilerStatusError(error instanceof Error ? error.message : "Unable to load auto-filer status");
      }
    }
  }, []);

  useEffect(() => {
    let active = true;
    void loadAutoFilerStatus(() => active);
    return () => {
      active = false;
    };
  }, [
    loadAutoFilerStatus,
    form.autoFiler.enabled,
    form.autoFiler.mode,
    form.autoFiler.strategy,
    form.autoFiler.maxOpenAutoFiled,
    form.autoFiler.velocityWindowDays,
    form.autoFiler.minVelocityTransitions,
  ]);

  const handleRunAutoFilerNow = useCallback(async () => {
    setAutoFilerRunPending(true);
    setAutoFilerStatusError(null);
    try {
      setAutoFilerStatus(await autoFilerService.runNow());
    } catch (error) {
      setAutoFilerStatusError(error instanceof Error ? error.message : "Unable to run auto-filer cycle");
    } finally {
      setAutoFilerRunPending(false);
    }
  }, []);

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
            laneConcurrencyLimits: { ...DEFAULT_SETTINGS.laneConcurrencyLimits },
            maxQueueDepth: DEFAULT_SETTINGS.maxQueueDepth,
            circuitBreakerThreshold: DEFAULT_SETTINGS.circuitBreakerThreshold,
            circuitBreakerCooldownMinutes: DEFAULT_SETTINGS.circuitBreakerCooldownMinutes,
            executionCostCapPerRun: DEFAULT_SETTINGS.executionCostCapPerRun,
            costPerTurnEstimate: DEFAULT_SETTINGS.costPerTurnEstimate,
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Lane Concurrency Limits</label>
            <p className="mt-1 text-xs text-slate-400">
              Per-phase-kind concurrency caps (each 1-50). Lanes mirror the
              Operations Center columns: <em>investigate</em> covers
              workshop / clarify / classify / research, <em>execute</em>
              covers backlog process runs, <em>review</em> covers review
              and finalize, <em>reconcile</em> covers reconciliation
              phases.
            </p>
            <div className="mt-2 grid grid-cols-2 gap-3 sm:grid-cols-4">
              {(["investigate", "execute", "review", "reconcile"] as const).map((lane) => (
                <div key={lane}>
                  <label className="block text-xs font-medium capitalize text-slate-400">{lane}</label>
                  <Input
                    type="number"
                    min={1}
                    max={50}
                    className="mt-1"
                    value={form.laneConcurrencyLimits[lane] ?? DEFAULT_SETTINGS.laneConcurrencyLimits[lane]}
                    onChange={(e) => {
                      const next = Math.max(1, Math.min(50, Number(e.target.value || 1)));
                      patch({
                        laneConcurrencyLimits: {
                          ...form.laneConcurrencyLimits,
                          [lane]: next,
                        },
                      });
                    }}
                  />
                </div>
              ))}
            </div>
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

      {/* Fix-Before-Feature Gate */}
      <Card data-testid={selectors.settings.fixBeforeFeature}>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Fix-Before-Feature Gate</h3>
            <p className="mt-1 text-sm text-slate-400">
              When a feature item is queued onto a scenario that already has open
              fix/chore work, advise or block until that work is cleared.
            </p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            fixBeforeFeature: DEFAULT_SETTINGS.fixBeforeFeature,
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Gate Mode</label>
            <p className="mt-1 text-xs text-slate-400">
              <em>off</em> ignores open fix work; <em>suggest</em> attaches a
              non-blocking advisory; <em>block</em> adds a forceable blocking
              reason that can be overridden with an explicit force.
            </p>
            <ToggleButtons
              value={form.fixBeforeFeature}
              options={[
                { value: "off" as const, label: "off" },
                { value: "suggest" as const, label: "suggest" },
                { value: "block" as const, label: "block" },
              ]}
              onChange={(v) => patch({ fixBeforeFeature: v })}
            />
          </div>
        </div>
      </Card>

      {/* Backlog Auto-Filer */}
      <Card>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Backlog Auto-Filer</h3>
            <p className="mt-1 text-sm text-slate-400">Governed automatic filing for maintenance findings.</p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            autoFiler: { ...DEFAULT_SETTINGS.autoFiler },
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Policy</label>
            <ToggleButtons
              value={form.autoFiler.enabled}
              options={[
                { value: false as const, label: "Disabled" },
                { value: true as const, label: "Enabled" },
              ]}
              onChange={(v) => patch({ autoFiler: { ...form.autoFiler, enabled: v } })}
            />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-slate-300">Mode</label>
              <ToggleButtons
                value={form.autoFiler.mode}
                options={[
                  { value: "suggest" as const, label: "suggest" },
                  { value: "auto_add" as const, label: "auto-add" },
                ]}
                onChange={(v) => patch({ autoFiler: { ...form.autoFiler, mode: v } })}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-300">Strategy</label>
              <ToggleButtons
                value={form.autoFiler.strategy}
                options={[
                  { value: "feature_pending" as const, label: "feature-pending" },
                  { value: "importance" as const, label: "importance" },
                ]}
                onChange={(v) => patch({ autoFiler: { ...form.autoFiler, strategy: v } })}
              />
            </div>
          </div>
          <div className="grid gap-4 border-t border-white/5 pt-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-slate-300">Max Open Auto-Filed</label>
              <Input
                type="number"
                min={1}
                max={100}
                className="mt-1"
                value={form.autoFiler.maxOpenAutoFiled}
                onChange={(e) => patch({ autoFiler: { ...form.autoFiler, maxOpenAutoFiled: Math.max(1, Math.min(100, Number(e.target.value || 1))) } })}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-300">Velocity Window Days</label>
              <Input
                type="number"
                min={1}
                max={90}
                className="mt-1"
                value={form.autoFiler.velocityWindowDays}
                onChange={(e) => patch({ autoFiler: { ...form.autoFiler, velocityWindowDays: Math.max(1, Math.min(90, Number(e.target.value || 1))) } })}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-300">Min Velocity Transitions</label>
              <Input
                type="number"
                min={1}
                max={1000}
                className="mt-1"
                value={form.autoFiler.minVelocityTransitions}
                onChange={(e) => patch({ autoFiler: { ...form.autoFiler, minVelocityTransitions: Math.max(1, Math.min(1000, Number(e.target.value || 1))) } })}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-300">Interval Minutes</label>
              <Input
                type="number"
                min={1}
                max={1440}
                className="mt-1"
                value={form.autoFiler.intervalMinutes}
                onChange={(e) => patch({ autoFiler: { ...form.autoFiler, intervalMinutes: Math.max(1, Math.min(1440, Number(e.target.value || 1))) } })}
              />
            </div>
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Goal Name</label>
            <Input
              className="mt-1"
              value={form.autoFiler.goalName}
              onChange={(e) => patch({ autoFiler: { ...form.autoFiler, goalName: e.target.value } })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <h4 className="text-sm font-medium text-slate-300">Operator Status</h4>
                <p className="mt-1 text-xs text-slate-500">
                  Latest governed filing cycle and policy brakes.
                </p>
              </div>
              <div className="flex items-center gap-2">
                <span className={`rounded-full px-2 py-0.5 text-xs ${autoFilerStatus?.enabled ? "bg-emerald-500/10 text-emerald-300" : "bg-slate-700 text-slate-300"}`}>
                  {autoFilerStatus?.enabled ? "enabled" : "disabled"}
                </span>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => void handleRunAutoFilerNow()}
                  disabled={autoFilerRunPending}
                >
                  <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${autoFilerRunPending ? "animate-spin" : ""}`} />
                  Run now
                </Button>
              </div>
            </div>
            {autoFilerStatusError ? (
              <p className="mt-3 text-xs text-amber-300">{autoFilerStatusError}</p>
            ) : (
              <div className="mt-3 grid gap-3 sm:grid-cols-3">
                <div className="rounded border border-white/5 bg-slate-950/30 px-3 py-2">
                  <div className="text-[11px] uppercase text-slate-500">Last Cycle</div>
                  <div className="mt-1 text-sm text-slate-200">{formatAutoFilerTime(autoFilerStatus?.lastCycleTime ?? "")}</div>
                </div>
                <div className="rounded border border-white/5 bg-slate-950/30 px-3 py-2">
                  <div className="text-[11px] uppercase text-slate-500">Open / Cap</div>
                  <div className="mt-1 text-sm text-slate-200">
                    {autoFilerStatus ? `${autoFilerStatus.openAutoFiled} / ${autoFilerStatus.maxOpenAutoFiled}` : "--"}
                  </div>
                </div>
                <div className="rounded border border-white/5 bg-slate-950/30 px-3 py-2">
                  <div className="text-[11px] uppercase text-slate-500">Velocity Brake</div>
                  <div className="mt-1 text-sm text-slate-200">
                    {autoFilerStatus?.brake
                      ? `${autoFilerStatus.brake.observed}/${autoFilerStatus.brake.minimum}${autoFilerStatus.brake.braked ? " braked" : ""}`
                      : "--"}
                  </div>
                </div>
                <div className="rounded border border-white/5 bg-slate-950/30 px-3 py-2">
                  <div className="text-[11px] uppercase text-slate-500">Findings / Filed</div>
                  <div className="mt-1 text-sm text-slate-200">
                    {autoFilerStatus ? `${autoFilerStatus.findings} / ${autoFilerStatus.created}` : "--"}
                  </div>
                </div>
                <div className="rounded border border-white/5 bg-slate-950/30 px-3 py-2">
                  <div className="text-[11px] uppercase text-slate-500">Dismissed</div>
                  <div className="mt-1 text-sm text-slate-200">
                    {autoFilerStatus ? autoFilerStatus.dismissalCount : "--"}
                  </div>
                </div>
                <div className="rounded border border-white/5 bg-slate-950/30 px-3 py-2">
                  <div className="text-[11px] uppercase text-slate-500">Reconciled</div>
                  <div className="mt-1 text-sm text-slate-200">
                    {autoFilerStatus ? `${autoFilerStatus.reconciledClosed} closed, ${autoFilerStatus.reconciledNoted} noted` : "--"}
                  </div>
                </div>
              </div>
            )}
            {autoFilerStatus?.lastError ? (
              <p className="mt-3 text-xs text-amber-300">{autoFilerStatus.lastError}</p>
            ) : null}
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
        </div>
      </Card>

      <GoalDrainToggle />
    </div>
  );
}

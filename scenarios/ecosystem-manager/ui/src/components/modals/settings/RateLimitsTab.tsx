/**
 * RateLimitsTab Component
 * View and manage rate limit status
 */

import { useEffect, useMemo, useState } from 'react';
import { Clock, AlertCircle, RotateCcw, Gauge, SlidersHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { useQueueStatus } from '@/hooks/useQueueStatus';
import { useResetRateLimit } from '@/hooks/useRateLimits';
import type { ProcessorSettings } from '@/types/api';

interface RateLimitsTabProps {
  settings: ProcessorSettings;
  onChange: (updates: Partial<ProcessorSettings>) => void;
}

export function RateLimitsTab({ settings, onChange }: RateLimitsTabProps) {
  const { data: queueStatus } = useQueueStatus();
  const resetRateLimit = useResetRateLimit();
  const [countdown, setCountdown] = useState(0);

  const isRateLimited = queueStatus?.rate_limited || false;
  const retryAfter = queueStatus?.rate_limit_retry_after || 0;
  const pauseUntil = queueStatus?.rate_limit_pause_until
    ? new Date(queueStatus.rate_limit_pause_until)
    : null;

  const maxConcurrent = useMemo(() => {
    if (queueStatus?.max_concurrent) return queueStatus.max_concurrent;
    return settings.concurrent_slots || 1;
  }, [queueStatus?.max_concurrent, settings.concurrent_slots]);

  const slotsUsed = queueStatus?.slots_used ?? 0;
  const availableSlots = queueStatus?.available_slots ?? Math.max(maxConcurrent - slotsUsed, 0);
  const tasksRemaining = queueStatus?.tasks_remaining ?? 0;
  const cooldownSeconds = queueStatus?.cooldown_seconds ?? settings.cooldown_seconds ?? 0;

  const utilization = useMemo(() => {
    if (maxConcurrent <= 0) return 0;
    const raw = (slotsUsed / maxConcurrent) * 100;
    return Math.min(100, Math.max(0, Math.round(raw)));
  }, [slotsUsed, maxConcurrent]);

  // Countdown timer
  useEffect(() => {
    if (isRateLimited && retryAfter > 0) {
      setCountdown(retryAfter);
      const interval = setInterval(() => {
        setCountdown((prev) => {
          if (prev <= 1) {
            clearInterval(interval);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(interval);
    }
  }, [isRateLimited, retryAfter]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const formatDuration = (seconds: number) => {
    if (!seconds || seconds < 0) return 'now';
    const mins = Math.floor(seconds / 60);
    if (mins >= 60) {
      const hrs = Math.floor(mins / 60);
      const remMins = mins % 60;
      return `${hrs}h ${remMins}m`;
    }
    if (mins === 0) return `${seconds}s`;
    return `${mins}m`;
  };

  const handleReset = () => {
    if (confirm('Reset rate limit? This should only be done if you have confirmed with the API provider that the limit has been lifted.')) {
      resetRateLimit.mutate();
    }
  };

  const handleCooldownChange = (value: number) => {
    if (Number.isNaN(value)) return;
    const clamped = Math.max(5, Math.min(300, value));
    onChange({ cooldown_seconds: clamped });
  };

  return (
    <div className="space-y-6">
      {/* Overview */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2 text-sm text-slate-300">
          <Clock className="h-4 w-4" />
          <div>
            <p className="font-medium">Limits & throttling</p>
            <p className="text-xs text-slate-500">
              Monitor queue usage, tweak throttles, and see how close you are to provider limits.
            </p>
          </div>
        </div>
        <div className="text-xs text-slate-500">
          Task cooldown {formatDuration(cooldownSeconds)}
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        {/* Queue usage */}
        <div className="rounded-xl border border-white/5 bg-slate-900/60 p-5 shadow-inner">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-400">Queue usage</p>
              <h3 className="text-lg font-semibold text-white mt-1">
                {slotsUsed}/{maxConcurrent} slots in use
              </h3>
            </div>
            <div className="rounded-full bg-slate-800/80 p-3">
              <Gauge className="h-5 w-5 text-emerald-400" />
            </div>
          </div>
          <div className="mt-4 h-2 rounded-full bg-slate-800 overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-emerald-400 via-amber-300 to-orange-400 transition-all duration-500"
              style={{ width: `${utilization}%` }}
            />
          </div>
          <div className="mt-4 grid grid-cols-3 gap-3 text-xs text-slate-300">
            <div className="rounded-lg border border-white/5 bg-white/5 p-3">
              <p className="text-slate-400">Available</p>
              <p className="text-sm font-semibold text-white">{availableSlots} slots</p>
            </div>
            <div className="rounded-lg border border-white/5 bg-white/5 p-3">
              <p className="text-slate-400">Queued</p>
              <p className="text-sm font-semibold text-white">{tasksRemaining} tasks</p>
            </div>
            <div className="rounded-lg border border-white/5 bg-white/5 p-3">
              <p className="text-slate-400">Throttle</p>
              <p className="text-sm font-semibold text-white">
                {settings.cooldown_seconds}s cooldown
              </p>
            </div>
          </div>
        </div>

        {/* Rate limit status */}
        <div className={`rounded-xl border p-5 shadow-inner ${
          isRateLimited
            ? 'border-amber-400/30 bg-amber-500/10'
            : 'border-emerald-400/20 bg-emerald-500/5'
        }`}>
          <div className="flex items-start gap-4">
            <div className={`p-3 rounded-full ${
              isRateLimited ? 'bg-amber-500/20' : 'bg-emerald-500/15'
            }`}>
              {isRateLimited ? (
                <AlertCircle className="h-6 w-6 text-amber-300" />
              ) : (
                <Clock className="h-6 w-6 text-emerald-300" />
              )}
            </div>
            <div className="flex-1">
              <div className="flex items-center justify-between gap-2">
                <h3 className="text-lg font-semibold">
                  {isRateLimited ? 'Provider rate limit active' : 'No active provider limits'}
                </h3>
                {isRateLimited && (
                  <span className="rounded-full bg-amber-500/20 px-2 py-1 text-[11px] font-medium text-amber-200">
                    Paused
                  </span>
                )}
              </div>
              <p className="text-sm text-slate-200 mt-1 mb-3">
                {isRateLimited
                  ? 'The processor is temporarily paused by the provider. We will resume automatically once the cooldown expires.'
                  : 'All clear. The processor is running under the current throttle without provider-imposed pauses.'}
              </p>

              {isRateLimited && countdown > 0 && (
                <div className="bg-black/30 border border-white/10 rounded-md px-4 py-3 mb-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-slate-300">Time until retry</span>
                    <span className="text-2xl font-mono font-bold text-amber-200">
                      {formatTime(countdown)}
                    </span>
                  </div>
                  <div className="mt-2 h-2 bg-slate-800 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-amber-300 transition-all duration-1000"
                      style={{
                        width: `${retryAfter > 0 ? ((retryAfter - countdown) / retryAfter) * 100 : 0}%`,
                      }}
                    />
                  </div>
                  <div className="mt-2 flex items-center justify-between text-[11px] text-slate-400">
                    <span>Auto-resume {pauseUntil ? `around ${pauseUntil.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}` : 'after cooldown'}</span>
                    <span>{formatDuration(retryAfter)} remaining</span>
                  </div>
                </div>
              )}

              <div className="grid grid-cols-2 gap-3 text-xs text-slate-300">
                <div className="rounded-lg border border-white/5 bg-white/5 p-3">
                  <p className="text-slate-400">Current throttle</p>
                  <p className="text-sm font-semibold text-white">
                    {maxConcurrent} concurrent · {settings.cooldown_seconds}s cooldown
                  </p>
                </div>
                <div className="rounded-lg border border-white/5 bg-white/5 p-3">
                  <p className="text-slate-400">Resume target</p>
                  <p className="text-sm font-semibold text-white">
                    {isRateLimited
                      ? (pauseUntil
                        ? pauseUntil.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
                        : formatDuration(retryAfter))
                      : 'Not paused'}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Limit controls */}
      <div className="rounded-xl border border-white/5 bg-slate-900/70 p-4 shadow-inner space-y-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <SlidersHorizontal className="h-4 w-4 text-slate-300" />
            <div>
              <h4 className="text-sm font-semibold text-white">Limit tuning</h4>
              <p className="text-xs text-slate-500">
                Adjust throttle to balance throughput against provider limits.
              </p>
            </div>
          </div>
          <span className="text-[11px] text-slate-500">Changes save with the main Save button.</span>
        </div>

        <div className="grid gap-5 md:grid-cols-2">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label htmlFor="limit-concurrent">Max concurrent tasks</Label>
              <span className="text-sm font-medium text-slate-200">
                {settings.concurrent_slots}
              </span>
            </div>
            <Slider
              id="limit-concurrent"
              min={1}
              max={5}
              step={1}
              value={[settings.concurrent_slots]}
              onValueChange={(value) => onChange({ concurrent_slots: value[0] })}
            />
            <p className="text-xs text-slate-500">
              Lower values slow throughput but reduce the chance of hitting provider limits.
            </p>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="limit-cooldown">Task cooldown (seconds)</Label>
              <span className="text-sm font-medium text-slate-200">
                {settings.cooldown_seconds}s
              </span>
            </div>
            <Input
              id="limit-cooldown"
              type="number"
              min={5}
              max={300}
              step={5}
              value={settings.cooldown_seconds}
              onChange={(e) => handleCooldownChange(Number(e.target.value))}
              className="bg-slate-950/70"
            />
            <p className="text-xs text-slate-500">
              Delay before the recycler revisits completed/failed tasks. Longer cooldowns reduce churn after errors.
            </p>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between gap-3 pt-4 border-t border-white/5">
        <div className="text-xs text-slate-500">
          Keep the reset button as a last resort after confirming the provider has cleared the block.
        </div>
        <Button
          variant="outline"
          onClick={handleReset}
          disabled={!isRateLimited || resetRateLimit.isPending}
        >
          <RotateCcw className="h-4 w-4 mr-2" />
          {resetRateLimit.isPending ? 'Resetting...' : 'Reset Rate Limit'}
        </Button>
      </div>

      {/* Info Section */}
      <div className="bg-slate-900/60 border border-blue-400/20 rounded-md p-4">
        <h4 className="text-sm font-medium text-blue-300 mb-2">Rate limit guidance</h4>
        <ul className="text-xs text-slate-300 space-y-1">
          <li>• Rate limits come from your AI provider (OpenRouter, Anthropic, etc.)</li>
          <li>• Tighten concurrency or increase cooldown if you see frequent pauses</li>
          <li>• The processor auto-pauses and resumes; manual reset is for confirmed recoveries</li>
          <li>• Consider upgrading your API plan if you repeatedly exhaust capacity</li>
        </ul>
      </div>
    </div>
  );
}

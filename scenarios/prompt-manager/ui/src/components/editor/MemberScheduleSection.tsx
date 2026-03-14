import { useState, useEffect, useMemo, useCallback } from 'react'
import { Clock, Play, Save, X, Loader2, ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { HeartbeatConfig } from '@/services/heartbeatService'
import {
  type ScheduleMode,
  type IntervalUnit,
  DAY_OPTIONS,
  DEFAULT_DAYS,
  parseFixedNumber,
  sortDays,
  parseSchedule,
  formatScheduleSummary,
} from '@/lib/scheduleUtils'

interface MemberScheduleSectionProps {
  schedule: string
  heartbeatConfig: HeartbeatConfig | null
  isSaving: boolean
  onSaveSchedule: (schedule: string) => Promise<boolean>
  onTriggerHeartbeat: () => void
  onSetHeartbeatEnabled: (enabled: boolean) => void
  /** Whether the agent is currently running a heartbeat */
  isRunning?: boolean
  /** Duration string for the current run (e.g. "2m 34s") */
  runDuration?: string
  /** Run ID for currently running heartbeat (if known) */
  runningRunId?: string | null
  /** Open the run view for a run ID */
  onOpenRun?: (runId: string) => void
}

function formatTimestamp(value?: string): string | null {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

export function MemberScheduleSection({
  schedule,
  heartbeatConfig,
  isSaving,
  onSaveSchedule,
  onTriggerHeartbeat,
  onSetHeartbeatEnabled,
  isRunning = false,
  runDuration,
  runningRunId = null,
  onOpenRun,
}: MemberScheduleSectionProps) {
  const [isScheduleEditing, setIsScheduleEditing] = useState(false)
  const [scheduleMode, setScheduleMode] = useState<ScheduleMode>('custom')
  const [dailyTime, setDailyTime] = useState('09:00')
  const [dailyDays, setDailyDays] = useState<number[]>(DEFAULT_DAYS)
  const [intervalValue, setIntervalValue] = useState(6)
  const [intervalUnit, setIntervalUnit] = useState<IntervalUnit>('hours')
  const [intervalTime, setIntervalTime] = useState('00:00')
  const [customCron, setCustomCron] = useState('')

  const scheduleSummary = useMemo(() => formatScheduleSummary(schedule), [schedule])
  const nextExecutionLabel = useMemo(
    () => formatTimestamp(heartbeatConfig?.nextExecution),
    [heartbeatConfig?.nextExecution]
  )
  const lastExecutionLabel = useMemo(
    () => formatTimestamp(heartbeatConfig?.lastExecution?.startedAt),
    [heartbeatConfig?.lastExecution?.startedAt]
  )

  const loadScheduleEditorState = useCallback((value: string) => {
    const parsed = parseSchedule(value)
    if (parsed.mode === 'daily') {
      setScheduleMode('daily')
      setDailyTime(parsed.dailyTime ?? '09:00')
      setDailyDays(parsed.dailyDays ?? DEFAULT_DAYS)
    } else if (parsed.mode === 'interval') {
      setScheduleMode('interval')
      setIntervalUnit(parsed.intervalUnit ?? 'hours')
      setIntervalValue(parsed.intervalValue ?? 1)
      setIntervalTime(parsed.intervalTime ?? '00:00')
    } else {
      setScheduleMode('custom')
    }
    setCustomCron(value)
  }, [])

  useEffect(() => {
    loadScheduleEditorState(schedule)
  }, [schedule, loadScheduleEditorState])

  const scheduleDraft = useMemo(() => {
    if (scheduleMode === 'daily') {
      const [hourRaw, minuteRaw] = dailyTime.split(':')
      const hour = parseFixedNumber(hourRaw ?? '0', 0, 23) ?? 0
      const minute = parseFixedNumber(minuteRaw ?? '0', 0, 59) ?? 0
      const days = dailyDays.length ? sortDays(dailyDays) : DEFAULT_DAYS
      const dayField = days.length === DEFAULT_DAYS.length ? '*' : days.join(',')
      return `${minute} ${hour} * * ${dayField}`
    }
    if (scheduleMode === 'interval') {
      const value = Math.max(1, Math.floor(intervalValue || 1))
      if (intervalUnit === 'minutes') {
        return `*/${value} * * * *`
      }
      const [hourRaw, minuteRaw] = intervalTime.split(':')
      const hour = parseFixedNumber(hourRaw ?? '0', 0, 23) ?? 0
      const minute = parseFixedNumber(minuteRaw ?? '0', 0, 59) ?? 0
      if (intervalUnit === 'hours') {
        return `${minute} */${value} * * *`
      }
      return `${minute} ${hour} */${value} * *`
    }
    return customCron.trim()
  }, [scheduleMode, dailyTime, dailyDays, intervalValue, intervalUnit, intervalTime, customCron])

  const isScheduleDirty = scheduleDraft.trim() !== schedule.trim()
  const isScheduleValid = Boolean(scheduleDraft.trim())

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <label className="text-sm font-medium">Schedule</label>
        </div>
        {isScheduleEditing ? (
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => {
                loadScheduleEditorState(schedule)
                setIsScheduleEditing(false)
              }}
              className={cn(
                'flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium rounded-lg transition-colors',
                'bg-muted text-muted-foreground hover:bg-muted/80'
              )}
            >
              <X className="h-3.5 w-3.5" />
              Cancel
            </button>
            <button
              type="button"
              onClick={() => {
                void onSaveSchedule(scheduleDraft).then((success) => {
                  if (success) {
                    setIsScheduleEditing(false)
                  }
                })
              }}
              disabled={!isScheduleDirty || !isScheduleValid || isSaving}
              className={cn(
                'flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium rounded-lg transition-colors',
                isScheduleDirty && isScheduleValid
                  ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                  : 'bg-muted text-muted-foreground cursor-not-allowed',
                isSaving && 'opacity-50'
              )}
            >
              <Save className="h-3.5 w-3.5" />
              {isSaving ? 'Saving...' : 'Save'}
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setIsScheduleEditing(true)}
            className={cn(
              'px-2.5 py-1.5 text-xs font-medium rounded-lg transition-colors',
              'bg-muted text-muted-foreground hover:bg-muted/80'
            )}
          >
            Edit schedule
          </button>
        )}
      </div>

      {isRunning && (
        <div className="flex items-center justify-between gap-2 px-2 py-1.5 bg-purple-500/10 border border-purple-500/20 rounded-md">
          <div className="flex items-center gap-2">
            <Loader2 className="h-3.5 w-3.5 text-purple-500 animate-spin" />
            <span className="text-xs font-medium text-purple-500">
              Currently running{runDuration ? ` (${runDuration})` : ''}
            </span>
          </div>
          {runningRunId && onOpenRun && (
            <button
              type="button"
              onClick={() => onOpenRun(runningRunId)}
              className={cn(
                'inline-flex items-center gap-1 px-2 py-1 text-[11px] rounded-md',
                'bg-purple-500/20 hover:bg-purple-500/30 text-purple-400 transition-colors'
              )}
              aria-label="Open run view"
              title="Open run view"
            >
              <ExternalLink className="h-3 w-3" />
              <span>Open Run</span>
            </button>
          )}
        </div>
      )}

      {!isScheduleEditing ? (
        <div className="rounded-lg border border-border bg-muted/40 p-3 space-y-2">
          <div className="flex items-start justify-between gap-3">
            <div className="space-y-1">
              <p className="text-sm font-medium text-foreground">{scheduleSummary}</p>
              {scheduleSummary === 'Custom schedule' && (
                <p className="text-xs text-muted-foreground">Cron: {schedule}</p>
              )}
              {nextExecutionLabel && (
                <p className="text-xs text-muted-foreground">Next run: {nextExecutionLabel}</p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={onTriggerHeartbeat}
                disabled={!heartbeatConfig || isSaving}
                className={cn(
                  'inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium rounded-lg transition-colors',
                  heartbeatConfig
                    ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                    : 'bg-muted text-muted-foreground cursor-not-allowed'
                )}
                aria-label="Run now"
                title="Run now"
              >
                <Play className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Run now</span>
              </button>
              <button
                type="button"
                onClick={() => onSetHeartbeatEnabled(!(heartbeatConfig?.enabled ?? false))}
                disabled={isSaving}
                aria-pressed={heartbeatConfig?.enabled ?? false}
                className={cn(
                  'px-2 py-0.5 text-[11px] font-medium rounded-full border transition-colors',
                  heartbeatConfig?.enabled
                    ? 'bg-emerald-500/15 text-emerald-500 border-emerald-500/30 hover:bg-emerald-500/25'
                    : 'bg-muted text-muted-foreground border-border hover:bg-muted/80'
                )}
                title={heartbeatConfig?.enabled ? 'Disable heartbeat' : 'Enable heartbeat'}
              >
                {heartbeatConfig ? (heartbeatConfig.enabled ? 'Enabled' : 'Disabled') : 'Not configured'}
              </button>
            </div>
          </div>
          {lastExecutionLabel && heartbeatConfig?.lastExecution && (
            <div className="text-xs text-muted-foreground">
              Last run: {lastExecutionLabel} · {heartbeatConfig.lastExecution.status}
              {heartbeatConfig.lastExecution.error && (
                <span className="text-destructive"> · {heartbeatConfig.lastExecution.error}</span>
              )}
            </div>
          )}
        </div>
      ) : (
        <div className="rounded-lg border border-border bg-muted/40 p-3 space-y-4">
          <div className="inline-flex rounded-lg border border-border bg-background p-0.5">
            {(['daily', 'interval', 'custom'] as ScheduleMode[]).map((mode) => (
              <button
                key={mode}
                type="button"
                onClick={() => setScheduleMode(mode)}
                className={cn(
                  'px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
                  scheduleMode === mode
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {mode === 'daily' ? 'Daily' : mode === 'interval' ? 'Interval' : 'Custom'}
              </button>
            ))}
          </div>

          {scheduleMode === 'daily' && (
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-3">
                <div className="flex items-center gap-2">
                  <input
                    type="time"
                    value={dailyTime}
                    onChange={(e) => setDailyTime(e.target.value)}
                    className={cn(
                      'px-3 py-2 text-sm rounded-lg border border-border',
                      'bg-background text-foreground',
                      'focus:outline-none focus:ring-2 focus:ring-primary'
                    )}
                  />
                </div>
                <div className="flex flex-wrap gap-1">
                  {DAY_OPTIONS.map((day) => {
                    const isActive = dailyDays.includes(day.value)
                    return (
                      <button
                        key={day.value}
                        type="button"
                        onClick={() => {
                          setDailyDays((prev) => {
                            const set = new Set(prev)
                            if (set.has(day.value)) {
                              set.delete(day.value)
                            } else {
                              set.add(day.value)
                            }
                            const next = sortDays(Array.from(set))
                            return next.length ? next : prev
                          })
                        }}
                        className={cn(
                          'px-2.5 py-1 text-xs font-medium rounded-full transition-colors',
                          isActive
                            ? 'bg-foreground text-background'
                            : 'bg-muted text-muted-foreground hover:bg-muted/80'
                        )}
                      >
                        {day.label}
                      </button>
                    )
                  })}
                </div>
              </div>
            </div>
          )}

          {scheduleMode === 'interval' && (
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm text-muted-foreground">Every</span>
                <input
                  type="number"
                  min={1}
                  value={intervalValue}
                  onChange={(e) => setIntervalValue(Number(e.target.value))}
                  className={cn(
                    'w-20 px-2 py-1.5 text-sm rounded-lg border border-border',
                    'bg-background text-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                />
                <select
                  value={intervalUnit}
                  onChange={(e) => setIntervalUnit(e.target.value as IntervalUnit)}
                  className={cn(
                    'px-2 py-1.5 text-sm rounded-lg border border-border',
                    'bg-background text-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                >
                  <option value="minutes">minutes</option>
                  <option value="hours">hours</option>
                  <option value="days">days</option>
                </select>
              </div>
              {(intervalUnit === 'hours' || intervalUnit === 'days') && (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-muted-foreground">At</span>
                  <input
                    type="time"
                    value={intervalTime}
                    onChange={(e) => setIntervalTime(e.target.value)}
                    className={cn(
                      'px-3 py-2 text-sm rounded-lg border border-border',
                      'bg-background text-foreground',
                      'focus:outline-none focus:ring-2 focus:ring-primary'
                    )}
                  />
                </div>
              )}
            </div>
          )}

          {scheduleMode === 'custom' && (
            <div className="space-y-2">
              <label className="text-xs text-muted-foreground">Cron expression</label>
              <input
                type="text"
                value={customCron}
                onChange={(e) => setCustomCron(e.target.value)}
                className={cn(
                  'w-full px-3 py-2 text-sm rounded-lg border border-border',
                  'bg-background text-foreground',
                  'focus:outline-none focus:ring-2 focus:ring-primary'
                )}
                placeholder="0 */6 * * *"
              />
              <p className="text-[11px] text-muted-foreground">
                Use a standard 5-field cron expression.
              </p>
            </div>
          )}

          <div className="text-[11px] text-muted-foreground">
            Cron preview: {scheduleDraft || '—'}
          </div>
        </div>
      )}
    </div>
  )
}

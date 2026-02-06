import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { Clock, Play, Pause, Save, X, MoreVertical, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { HeartbeatConfig } from '@/services/heartbeatService'

type ScheduleMode = 'daily' | 'interval' | 'custom'
type IntervalUnit = 'minutes' | 'hours' | 'days'

interface ParsedSchedule {
  mode: ScheduleMode
  dailyTime?: string
  dailyDays?: number[]
  intervalValue?: number
  intervalUnit?: IntervalUnit
  intervalTime?: string
}

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
}

const DAY_OPTIONS = [
  { value: 1, label: 'Mo', longLabel: 'Mon' },
  { value: 2, label: 'Tu', longLabel: 'Tue' },
  { value: 3, label: 'We', longLabel: 'Wed' },
  { value: 4, label: 'Th', longLabel: 'Thu' },
  { value: 5, label: 'Fr', longLabel: 'Fri' },
  { value: 6, label: 'Sa', longLabel: 'Sat' },
  { value: 0, label: 'Su', longLabel: 'Sun' },
]

const DAY_ORDER = new Map(DAY_OPTIONS.map((day, index) => [day.value, index]))
const DEFAULT_DAYS = DAY_OPTIONS.map((day) => day.value)

function pad2(value: number): string {
  return value.toString().padStart(2, '0')
}

function toTimeValue(hour: number, minute: number): string {
  return `${pad2(hour)}:${pad2(minute)}`
}

function parseFixedNumber(field: string, min: number, max: number): number | null {
  if (!/^\d+$/.test(field)) return null
  const value = Number(field)
  if (Number.isNaN(value) || value < min || value > max) return null
  return value
}

function parseStepField(field: string, min: number, max: number): number | null {
  const match = field.match(/^\*\/(\d+)$/)
  if (!match) return null
  const value = Number(match[1])
  if (Number.isNaN(value) || value < min || value > max) return null
  return value
}

function normalizeDayValue(value: number): number | null {
  if (value === 7) return 0
  if (value < 0 || value > 6) return null
  return value
}

function sortDays(days: number[]): number[] {
  return [...new Set(days)].sort((a, b) => {
    const orderA = DAY_ORDER.get(a) ?? 0
    const orderB = DAY_ORDER.get(b) ?? 0
    return orderA - orderB
  })
}

function parseDaysOfWeek(field: string): number[] | null {
  if (field === '*') {
    return [...DEFAULT_DAYS]
  }

  const parts = field.split(',')
  const days: number[] = []
  for (const part of parts) {
    if (!part) return null
    if (part.includes('-')) {
      const [startRaw, endRaw] = part.split('-')
      if (!startRaw || !endRaw) return null
      const start = Number(startRaw)
      const end = Number(endRaw)
      if (Number.isNaN(start) || Number.isNaN(end)) return null
      const step = start <= end ? 1 : -1
      for (let current = start; step > 0 ? current <= end : current >= end; current += step) {
        const normalized = normalizeDayValue(current)
        if (normalized === null) return null
        days.push(normalized)
      }
    } else {
      const value = Number(part)
      if (Number.isNaN(value)) return null
      const normalized = normalizeDayValue(value)
      if (normalized === null) return null
      days.push(normalized)
    }
  }

  const sorted = sortDays(days)
  return sorted.length ? sorted : null
}

function parseSchedule(schedule: string): ParsedSchedule {
  const fields = schedule.trim().split(/\s+/)
  if (fields.length < 5) {
    return { mode: 'custom' }
  }
  const [minuteField, hourField, domField, monthField, dowField] = fields
  if (!minuteField || !hourField || !domField || !monthField || !dowField) {
    return { mode: 'custom' }
  }

  const minute = parseFixedNumber(minuteField, 0, 59)
  const hour = parseFixedNumber(hourField, 0, 23)

  const intervalMinutes = parseStepField(minuteField, 1, 59)
  if (intervalMinutes && hourField === '*' && domField === '*' && monthField === '*' && dowField === '*') {
    return {
      mode: 'interval',
      intervalUnit: 'minutes',
      intervalValue: intervalMinutes,
    }
  }

  const intervalHours = parseStepField(hourField, 1, 23)
  if (intervalHours && minute !== null && domField === '*' && monthField === '*' && dowField === '*') {
    return {
      mode: 'interval',
      intervalUnit: 'hours',
      intervalValue: intervalHours,
      intervalTime: toTimeValue(hour ?? 0, minute),
    }
  }

  const intervalDays = parseStepField(domField, 1, 31)
  if (intervalDays && minute !== null && hour !== null && monthField === '*' && dowField === '*') {
    return {
      mode: 'interval',
      intervalUnit: 'days',
      intervalValue: intervalDays,
      intervalTime: toTimeValue(hour, minute),
    }
  }

  if (minute !== null && hour !== null && domField === '*' && monthField === '*') {
    const days = parseDaysOfWeek(dowField)
    if (days) {
      return {
        mode: 'daily',
        dailyTime: toTimeValue(hour, minute),
        dailyDays: days,
      }
    }
  }

  return { mode: 'custom' }
}

function formatTimeLabel(value: string): string {
  const [hourRaw, minuteRaw] = value.split(':')
  const hour = Number(hourRaw)
  const minute = Number(minuteRaw ?? '0')
  if (Number.isNaN(hour) || Number.isNaN(minute)) return value
  const period = hour >= 12 ? 'PM' : 'AM'
  const hour12 = hour % 12 === 0 ? 12 : hour % 12
  return `${hour12}:${pad2(minute)} ${period}`
}

function formatDaysLabel(days: number[]): string {
  const normalized = sortDays(days)
  const daySet = new Set(normalized)
  const isEveryDay = DEFAULT_DAYS.every((day) => daySet.has(day))
  if (isEveryDay) return 'Every day'
  const isWeekdays = [1, 2, 3, 4, 5].every((day) => daySet.has(day)) && daySet.size === 5
  if (isWeekdays) return 'Weekdays'
  const isWeekends = daySet.size === 2 && daySet.has(0) && daySet.has(6)
  if (isWeekends) return 'Weekends'
  return normalized
    .map((day) => DAY_OPTIONS.find((opt) => opt.value === day)?.longLabel ?? '')
    .filter(Boolean)
    .join(', ')
}

function formatScheduleSummary(schedule: string): string {
  if (!schedule.trim()) return 'No schedule configured'
  const parsed = parseSchedule(schedule)
  if (parsed.mode === 'daily' && parsed.dailyTime && parsed.dailyDays) {
    return `${formatDaysLabel(parsed.dailyDays)} at ${formatTimeLabel(parsed.dailyTime)}`
  }
  if (parsed.mode === 'interval' && parsed.intervalValue && parsed.intervalUnit) {
    const unitLabel = parsed.intervalValue === 1 ? parsed.intervalUnit.slice(0, -1) : parsed.intervalUnit
    if (parsed.intervalUnit === 'minutes') {
      return `Every ${parsed.intervalValue} ${unitLabel}`
    }
    if (parsed.intervalUnit === 'hours' && parsed.intervalTime) {
      const minute = parsed.intervalTime.split(':')[1] ?? '00'
      return `Every ${parsed.intervalValue} ${unitLabel} at :${minute}`
    }
    const timeLabel = parsed.intervalTime ? ` at ${formatTimeLabel(parsed.intervalTime)}` : ''
    return `Every ${parsed.intervalValue} ${unitLabel}${timeLabel}`
  }
  return 'Custom schedule'
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
}: MemberScheduleSectionProps) {
  const [isScheduleEditing, setIsScheduleEditing] = useState(false)
  const [scheduleMode, setScheduleMode] = useState<ScheduleMode>('custom')
  const [dailyTime, setDailyTime] = useState('09:00')
  const [dailyDays, setDailyDays] = useState<number[]>(DEFAULT_DAYS)
  const [intervalValue, setIntervalValue] = useState(6)
  const [intervalUnit, setIntervalUnit] = useState<IntervalUnit>('hours')
  const [intervalTime, setIntervalTime] = useState('00:00')
  const [customCron, setCustomCron] = useState('')
  const [isScheduleMenuOpen, setIsScheduleMenuOpen] = useState(false)
  const scheduleMenuRef = useRef<HTMLDivElement>(null)

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

  useEffect(() => {
    if (!isScheduleMenuOpen) return
    const handleClickOutside = (event: MouseEvent) => {
      if (scheduleMenuRef.current && !scheduleMenuRef.current.contains(event.target as Node)) {
        setIsScheduleMenuOpen(false)
      }
    }
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsScheduleMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isScheduleMenuOpen])

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
        <div className="flex items-center gap-2 px-2 py-1.5 bg-purple-500/10 border border-purple-500/20 rounded-md">
          <Loader2 className="h-3.5 w-3.5 text-purple-500 animate-spin" />
          <span className="text-xs font-medium text-purple-500">
            Currently running{runDuration ? ` (${runDuration})` : ''}
          </span>
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
              <span
                className={cn(
                  'px-2 py-0.5 text-[11px] font-medium rounded-full border',
                  heartbeatConfig?.enabled
                    ? 'bg-emerald-500/15 text-emerald-500 border-emerald-500/30'
                    : 'bg-muted text-muted-foreground border-border'
                )}
              >
                {heartbeatConfig ? (heartbeatConfig.enabled ? 'Enabled' : 'Disabled') : 'Not configured'}
              </span>
              <div ref={scheduleMenuRef} className="relative">
                <button
                  type="button"
                  onClick={() => setIsScheduleMenuOpen((prev) => !prev)}
                  className={cn(
                    'p-1.5 rounded-lg transition-colors',
                    'bg-muted text-muted-foreground hover:bg-muted/80'
                  )}
                  aria-label="Schedule actions"
                >
                  <MoreVertical className="h-4 w-4" />
                </button>
                {isScheduleMenuOpen && (
                  <div
                    className={cn(
                      'absolute right-0 top-full mt-1 z-50',
                      'bg-card border border-border rounded-lg shadow-lg',
                      'min-w-[180px] overflow-hidden',
                      'animate-in fade-in-0 zoom-in-95 duration-100'
                    )}
                  >
                    <button
                      type="button"
                      onClick={() => {
                        setIsScheduleMenuOpen(false)
                        onTriggerHeartbeat()
                      }}
                      disabled={!heartbeatConfig || isSaving}
                      className={cn(
                        'w-full px-3 py-2 text-xs text-left flex items-center gap-2 transition-colors',
                        heartbeatConfig
                          ? 'text-foreground hover:bg-muted/60'
                          : 'text-muted-foreground cursor-not-allowed'
                      )}
                    >
                      <Play className="h-3.5 w-3.5" />
                      Run now
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setIsScheduleMenuOpen(false)
                        onSetHeartbeatEnabled(!(heartbeatConfig?.enabled ?? false))
                      }}
                      disabled={isSaving}
                      className={cn(
                        'w-full px-3 py-2 text-xs text-left flex items-center gap-2 transition-colors',
                        'text-foreground hover:bg-muted/60'
                      )}
                    >
                      {heartbeatConfig?.enabled ? (
                        <Pause className="h-3.5 w-3.5" />
                      ) : (
                        <Play className="h-3.5 w-3.5" />
                      )}
                      {heartbeatConfig?.enabled ? 'Disable heartbeat' : 'Enable heartbeat'}
                    </button>
                  </div>
                )}
              </div>
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

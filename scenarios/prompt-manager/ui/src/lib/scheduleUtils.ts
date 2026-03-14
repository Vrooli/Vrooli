/**
 * Cron schedule parsing and human-readable formatting utilities.
 *
 * Extracted from MemberScheduleSection for reuse in the team dashboard
 * and other schedule-displaying components.
 */

export type ScheduleMode = 'daily' | 'interval' | 'custom'
export type IntervalUnit = 'minutes' | 'hours' | 'days'

export interface ParsedSchedule {
  mode: ScheduleMode
  dailyTime?: string
  dailyDays?: number[]
  intervalValue?: number
  intervalUnit?: IntervalUnit
  intervalTime?: string
}

export const DAY_OPTIONS = [
  { value: 1, label: 'Mo', longLabel: 'Mon' },
  { value: 2, label: 'Tu', longLabel: 'Tue' },
  { value: 3, label: 'We', longLabel: 'Wed' },
  { value: 4, label: 'Th', longLabel: 'Thu' },
  { value: 5, label: 'Fr', longLabel: 'Fri' },
  { value: 6, label: 'Sa', longLabel: 'Sat' },
  { value: 0, label: 'Su', longLabel: 'Sun' },
]

export const DAY_ORDER = new Map(DAY_OPTIONS.map((day, index) => [day.value, index]))
export const DEFAULT_DAYS = DAY_OPTIONS.map((day) => day.value)

export function pad2(value: number): string {
  return value.toString().padStart(2, '0')
}

export function toTimeValue(hour: number, minute: number): string {
  return `${pad2(hour)}:${pad2(minute)}`
}

export function parseFixedNumber(field: string, min: number, max: number): number | null {
  if (!/^\d+$/.test(field)) return null
  const value = Number(field)
  if (Number.isNaN(value) || value < min || value > max) return null
  return value
}

export function parseStepField(field: string, min: number, max: number): number | null {
  const match = field.match(/^\*\/(\d+)$/)
  if (!match) return null
  const value = Number(match[1])
  if (Number.isNaN(value) || value < min || value > max) return null
  return value
}

export function normalizeDayValue(value: number): number | null {
  if (value === 7) return 0
  if (value < 0 || value > 6) return null
  return value
}

export function sortDays(days: number[]): number[] {
  return [...new Set(days)].sort((a, b) => {
    const orderA = DAY_ORDER.get(a) ?? 0
    const orderB = DAY_ORDER.get(b) ?? 0
    return orderA - orderB
  })
}

export function parseDaysOfWeek(field: string): number[] | null {
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

export function parseSchedule(schedule: string): ParsedSchedule {
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

export function formatTimeLabel(value: string): string {
  const [hourRaw, minuteRaw] = value.split(':')
  const hour = Number(hourRaw)
  const minute = Number(minuteRaw ?? '0')
  if (Number.isNaN(hour) || Number.isNaN(minute)) return value
  const period = hour >= 12 ? 'PM' : 'AM'
  const hour12 = hour % 12 === 0 ? 12 : hour % 12
  return `${hour12}:${pad2(minute)} ${period}`
}

export function formatDaysLabel(days: number[]): string {
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

export function formatScheduleSummary(schedule: string): string {
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

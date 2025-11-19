export interface User {
  id: string
  authUserId: string
  email: string
  displayName: string
  timezone: string
  createdAt: string
  updatedAt: string
}

export interface Event {
  id: string
  userId: string
  title: string
  description?: string
  startTime: string
  endTime: string
  timezone: string
  location?: string
  eventType: 'meeting' | 'task' | 'reminder' | 'personal' | 'other'
  status: 'scheduled' | 'in_progress' | 'completed' | 'cancelled'
  metadata?: Record<string, any>
  automationConfig?: AutomationConfig
  createdAt: string
  updatedAt: string
}

export interface AutomationConfig {
  enabled: boolean
  type: 'webhook' | 'notification' | 'recurring'
  webhookUrl?: string
  notificationTemplate?: string
  recurrence?: RecurrenceConfig
}

export interface RecurrenceConfig {
  pattern: 'daily' | 'weekly' | 'monthly' | 'yearly' | 'custom'
  interval: number
  endDate?: string
  maxOccurrences?: number
  daysOfWeek?: number[]
  dayOfMonth?: number
  exceptions?: string[]
}

export interface Reminder {
  id: string
  eventId: string
  userId: string
  minutesBefore: number
  type: 'email' | 'push' | 'sms'
  status: 'pending' | 'sent' | 'cancelled' | 'failed'
  sendAt: string
  sentAt?: string
  createdAt: string
}

export interface CreateEventRequest {
  title: string
  description?: string
  start_time: string
  end_time: string
  timezone?: string
  location?: string
  event_type?: Event['eventType']
  metadata?: Record<string, any>
  reminders?: ReminderRequest[]
  automation_config?: AutomationConfig
}

export interface ReminderRequest {
  minutesBefore: number
  notificationType: 'email' | 'push' | 'sms'
}

export interface ChatRequest {
  message: string
  conversationId?: string
  context?: Record<string, any>
}

export interface ChatResponse {
  response: string
  suggestedActions?: SuggestedAction[]
  requiresConfirmation?: boolean
  context?: Record<string, any>
}

export interface SuggestedAction {
  action: string
  confidence: number
  parameters?: Record<string, any>
}

export interface OptimizationRequest {
  optimizationGoal: 'minimize_gaps' | 'maximize_focus_time' | 'balance_workload'
  startDate: string
  endDate: string
  constraints?: Record<string, any>
}

export interface OptimizationResponse {
  optimizationGoal: string
  currentEfficiency: number
  suggestions: OptimizationSuggestion[]
  potentialTimeSaved: number
}

export interface OptimizationSuggestion {
  type: 'consolidate_gap' | 'batch_similar' | 'reschedule' | 'cancel'
  description: string
  eventId?: string
  eventIds?: string[]
  proposedStart?: string
  confidence: number
}

export interface CalendarView {
  type: 'week' | 'month' | 'day' | 'agenda'
  currentDate: Date
  events: Event[]
}

export interface CalendarFilters {
  eventTypes?: Event['eventType'][]
  status?: Event['status'][]
  searchQuery?: string
  dateRange?: {
    start: Date
    end: Date
  }
}
import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { format, addDays, startOfWeek, addWeeks } from 'date-fns'
import { X, Clock, MapPin, Repeat, Calendar as CalendarIcon, Plus, Minus } from 'lucide-react'
import { useCalendarStore } from '@/stores/calendarStore'
import { api } from '@/services/api'
import toast from 'react-hot-toast'
import type { CreateEventRequest, Event, RecurrenceConfig } from '@/types'

export function EventModal() {
  const queryClient = useQueryClient()
  const { selectedEvent, selectEvent, addEvent, updateEvent, deleteEvent } = useCalendarStore()
  
  const isNewEvent = selectedEvent?.id === 'new'
  
  const [formData, setFormData] = useState<CreateEventRequest>({
    title: '',
    description: '',
    startTime: '',
    endTime: '',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    location: '',
    eventType: 'meeting',
    reminders: [{ minutesBefore: 15, notificationType: 'email' }],
    automationConfig: {
      enabled: false,
      type: 'recurring'
    }
  })

  const [recurrenceEnabled, setRecurrenceEnabled] = useState(false)
  const [recurrenceConfig, setRecurrenceConfig] = useState<RecurrenceConfig>({
    pattern: 'weekly',
    interval: 1,
    daysOfWeek: [],
    exceptions: []
  })
  const [showRecurrencePreview, setShowRecurrencePreview] = useState(false)

  useEffect(() => {
    if (selectedEvent) {
      setFormData({
        title: selectedEvent.title || '',
        description: selectedEvent.description || '',
        startTime: selectedEvent.startTime ? format(new Date(selectedEvent.startTime), "yyyy-MM-dd'T'HH:mm") : '',
        endTime: selectedEvent.endTime ? format(new Date(selectedEvent.endTime), "yyyy-MM-dd'T'HH:mm") : '',
        timezone: selectedEvent.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
        location: selectedEvent.location || '',
        eventType: selectedEvent.eventType || 'meeting',
        automationConfig: selectedEvent.automationConfig || {
          enabled: false,
          type: 'recurring'
        }
      })

      // Handle recurrence data
      const hasRecurrence = selectedEvent.automationConfig?.recurrence
      setRecurrenceEnabled(!!hasRecurrence)
      if (hasRecurrence) {
        setRecurrenceConfig({
          pattern: hasRecurrence.pattern,
          interval: hasRecurrence.interval,
          endDate: hasRecurrence.endDate,
          maxOccurrences: hasRecurrence.maxOccurrences,
          daysOfWeek: hasRecurrence.daysOfWeek || [],
          dayOfMonth: hasRecurrence.dayOfMonth,
          exceptions: hasRecurrence.exceptions || []
        })
      }
    } else {
      // Reset form when no event selected
      setRecurrenceEnabled(false)
      setRecurrenceConfig({
        pattern: 'weekly',
        interval: 1,
        daysOfWeek: [],
        exceptions: []
      })
    }
  }, [selectedEvent])

  const createMutation = useMutation({
    mutationFn: api.createEvent,
    onSuccess: (event) => {
      addEvent(event)
      toast.success('Event created successfully')
      selectEvent(null)
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: () => {
      toast.error('Failed to create event')
    }
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateEventRequest> }) => 
      api.updateEvent(id, data),
    onSuccess: (event) => {
      updateEvent(event.id, event)
      toast.success('Event updated successfully')
      selectEvent(null)
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: () => {
      toast.error('Failed to update event')
    }
  })

  const deleteMutation = useMutation({
    mutationFn: api.deleteEvent,
    onSuccess: () => {
      if (selectedEvent) {
        deleteEvent(selectedEvent.id)
      }
      toast.success('Event deleted successfully')
      selectEvent(null)
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: () => {
      toast.error('Failed to delete event')
    }
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!formData.title || !formData.startTime || !formData.endTime) {
      toast.error('Please fill in required fields')
      return
    }

    const eventData = {
      ...formData,
      startTime: new Date(formData.startTime).toISOString(),
      endTime: new Date(formData.endTime).toISOString(),
      automationConfig: recurrenceEnabled ? {
        enabled: true,
        type: 'recurring' as const,
        recurrence: recurrenceConfig
      } : {
        enabled: false,
        type: 'recurring' as const
      }
    }

    if (isNewEvent) {
      createMutation.mutate(eventData)
    } else if (selectedEvent) {
      updateMutation.mutate({ id: selectedEvent.id, data: eventData })
    }
  }

  const handleDelete = () => {
    if (selectedEvent && selectedEvent.id !== 'new') {
      if (confirm('Are you sure you want to delete this event?')) {
        deleteMutation.mutate(selectedEvent.id)
      }
    }
  }

  // Helper functions for recurrence
  const toggleDayOfWeek = (dayIndex: number) => {
    const newDays = recurrenceConfig.daysOfWeek?.includes(dayIndex)
      ? recurrenceConfig.daysOfWeek.filter(d => d !== dayIndex)
      : [...(recurrenceConfig.daysOfWeek || []), dayIndex]
    
    setRecurrenceConfig({
      ...recurrenceConfig,
      daysOfWeek: newDays.sort()
    })
  }

  const weekDays = [
    { label: 'S', value: 0 },
    { label: 'M', value: 1 },
    { label: 'T', value: 2 },
    { label: 'W', value: 3 },
    { label: 'T', value: 4 },
    { label: 'F', value: 5 },
    { label: 'S', value: 6 }
  ]

  const getRecurrencePreview = () => {
    if (!recurrenceEnabled || !formData.startTime) return []
    
    const startDate = new Date(formData.startTime)
    const previews: Date[] = []
    const exceptions = new Set(recurrenceConfig.exceptions || [])
    let currentDate = new Date(startDate)
    let addedCount = 0
    let iterations = 0
    const maxIterations = 100 // Prevent infinite loop
    
    while (addedCount < 5 && iterations < maxIterations && 
           (!recurrenceConfig.maxOccurrences || iterations < recurrenceConfig.maxOccurrences)) {
      const dateString = format(currentDate, 'yyyy-MM-dd')
      
      // Check if current date is in exceptions
      if (!exceptions.has(dateString)) {
        previews.push(new Date(currentDate))
        addedCount++
      }
      
      // Check if we've reached the end date
      if (recurrenceConfig.endDate && currentDate >= new Date(recurrenceConfig.endDate)) {
        break
      }
      
      // Move to next occurrence
      switch (recurrenceConfig.pattern) {
        case 'daily':
          currentDate = addDays(currentDate, recurrenceConfig.interval)
          break
        case 'weekly':
          currentDate = addWeeks(currentDate, recurrenceConfig.interval)
          break
        case 'monthly':
          currentDate.setMonth(currentDate.getMonth() + recurrenceConfig.interval)
          break
        case 'yearly':
          currentDate.setFullYear(currentDate.getFullYear() + recurrenceConfig.interval)
          break
      }
      
      iterations++
    }
    
    return previews
  }

  if (!selectedEvent) return null

  return (
    <div className="modal-overlay animate-fade-in" onClick={() => selectEvent(null)}>
      <div className="modal-content max-w-4xl animate-slide-in" onClick={(e) => e.stopPropagation()}>
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
            {isNewEvent ? 'New Event' : 'Edit Event'}
          </h2>
          <button
            onClick={() => selectEvent(null)}
            className="btn-ghost p-2"
          >
            <X className="h-6 w-6" />
          </button>
        </div>

        <div className="flex gap-6">
          {/* Main Form */}
          <div className="flex-1">
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Title */}
              <div>
                <label className="label text-sm font-medium">Title *</label>
                <input
                  type="text"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  className="input"
                  placeholder="Event title"
                  required
                />
              </div>

              {/* Date & Time */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="label text-sm font-medium flex items-center space-x-1">
                    <Clock className="h-4 w-4" />
                    <span>Start *</span>
                  </label>
                  <input
                    type="datetime-local"
                    value={formData.startTime}
                    onChange={(e) => setFormData({ ...formData, startTime: e.target.value })}
                    className="input"
                    required
                  />
                </div>
                <div>
                  <label className="label text-sm font-medium flex items-center space-x-1">
                    <Clock className="h-4 w-4" />
                    <span>End *</span>
                  </label>
                  <input
                    type="datetime-local"
                    value={formData.endTime}
                    onChange={(e) => setFormData({ ...formData, endTime: e.target.value })}
                    className="input"
                    required
                  />
                </div>
              </div>

              {/* Location & Event Type */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="label text-sm font-medium flex items-center space-x-1">
                    <MapPin className="h-4 w-4" />
                    <span>Location</span>
                  </label>
                  <input
                    type="text"
                    value={formData.location}
                    onChange={(e) => setFormData({ ...formData, location: e.target.value })}
                    className="input"
                    placeholder="Add location"
                  />
                </div>
                <div>
                  <label className="label text-sm font-medium">Event Type</label>
                  <select
                    value={formData.eventType}
                    onChange={(e) => setFormData({ ...formData, eventType: e.target.value as Event['eventType'] })}
                    className="input"
                  >
                    <option value="meeting">Meeting</option>
                    <option value="task">Task</option>
                    <option value="reminder">Reminder</option>
                    <option value="personal">Personal</option>
                    <option value="other">Other</option>
                  </select>
                </div>
              </div>

              {/* Description */}
              <div>
                <label className="label text-sm font-medium">Description</label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="input min-h-[100px] resize-none"
                  placeholder="Add description"
                />
              </div>

              {/* Recurrence Toggle */}
              <div className="border-t pt-4">
                <label className="flex items-center space-x-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={recurrenceEnabled}
                    onChange={(e) => setRecurrenceEnabled(e.target.checked)}
                    className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                  />
                  <div className="flex items-center space-x-2">
                    <Repeat className="h-4 w-4 text-primary-600" />
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      Repeat this event
                    </span>
                  </div>
                </label>
              </div>

              {/* Recurrence Configuration */}
              {recurrenceEnabled && (
                <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-4">
                  <div className="flex items-center justify-between">
                    <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
                      Recurrence Settings
                    </h3>
                    <button
                      type="button"
                      onClick={() => setShowRecurrencePreview(!showRecurrencePreview)}
                      className="btn-ghost text-sm flex items-center space-x-1"
                    >
                      <CalendarIcon className="h-4 w-4" />
                      <span>Preview</span>
                    </button>
                  </div>

                  {/* Pattern & Interval */}
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="label text-sm font-medium">Repeat Pattern</label>
                      <select
                        value={recurrenceConfig.pattern}
                        onChange={(e) => setRecurrenceConfig({
                          ...recurrenceConfig,
                          pattern: e.target.value as RecurrenceConfig['pattern']
                        })}
                        className="input"
                      >
                        <option value="daily">Daily</option>
                        <option value="weekly">Weekly</option>
                        <option value="monthly">Monthly</option>
                        <option value="yearly">Yearly</option>
                      </select>
                    </div>
                    <div>
                      <label className="label text-sm font-medium">Every</label>
                      <div className="flex items-center space-x-2">
                        <button
                          type="button"
                          onClick={() => setRecurrenceConfig({
                            ...recurrenceConfig,
                            interval: Math.max(1, recurrenceConfig.interval - 1)
                          })}
                          className="btn-outline p-2"
                        >
                          <Minus className="h-4 w-4" />
                        </button>
                        <input
                          type="number"
                          min="1"
                          max="52"
                          value={recurrenceConfig.interval}
                          onChange={(e) => setRecurrenceConfig({
                            ...recurrenceConfig,
                            interval: parseInt(e.target.value) || 1
                          })}
                          className="input text-center w-20"
                        />
                        <button
                          type="button"
                          onClick={() => setRecurrenceConfig({
                            ...recurrenceConfig,
                            interval: Math.min(52, recurrenceConfig.interval + 1)
                          })}
                          className="btn-outline p-2"
                        >
                          <Plus className="h-4 w-4" />
                        </button>
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          {recurrenceConfig.pattern === 'daily' ? 'day(s)' :
                           recurrenceConfig.pattern === 'weekly' ? 'week(s)' :
                           recurrenceConfig.pattern === 'monthly' ? 'month(s)' :
                           'year(s)'}
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Days of Week (for weekly pattern) */}
                  {recurrenceConfig.pattern === 'weekly' && (
                    <div>
                      <label className="label text-sm font-medium">On these days</label>
                      <div className="flex space-x-1">
                        {weekDays.map((day) => (
                          <button
                            key={day.value}
                            type="button"
                            onClick={() => toggleDayOfWeek(day.value)}
                            className={`flex h-10 w-10 items-center justify-center rounded-full text-sm font-medium transition-colors ${
                              recurrenceConfig.daysOfWeek?.includes(day.value)
                                ? 'bg-primary-600 text-white'
                                : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'
                            }`}
                          >
                            {day.label}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* End Conditions */}
                  <div>
                    <label className="label text-sm font-medium">Ends</label>
                    <div className="space-y-3">
                      <label className="flex items-center space-x-2">
                        <input
                          type="radio"
                          name="endCondition"
                          checked={!recurrenceConfig.endDate && !recurrenceConfig.maxOccurrences}
                          onChange={() => setRecurrenceConfig({
                            ...recurrenceConfig,
                            endDate: undefined,
                            maxOccurrences: undefined
                          })}
                          className="h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500"
                        />
                        <span className="text-sm">Never</span>
                      </label>
                      <label className="flex items-center space-x-2">
                        <input
                          type="radio"
                          name="endCondition"
                          checked={!!recurrenceConfig.endDate}
                          onChange={() => setRecurrenceConfig({
                            ...recurrenceConfig,
                            endDate: format(new Date(new Date().setMonth(new Date().getMonth() + 3)), 'yyyy-MM-dd'),
                            maxOccurrences: undefined
                          })}
                          className="h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500"
                        />
                        <span className="text-sm">On date</span>
                        {recurrenceConfig.endDate && (
                          <input
                            type="date"
                            value={recurrenceConfig.endDate}
                            onChange={(e) => setRecurrenceConfig({
                              ...recurrenceConfig,
                              endDate: e.target.value
                            })}
                            className="input text-sm ml-2"
                          />
                        )}
                      </label>
                      <label className="flex items-center space-x-2">
                        <input
                          type="radio"
                          name="endCondition"
                          checked={!!recurrenceConfig.maxOccurrences}
                          onChange={() => setRecurrenceConfig({
                            ...recurrenceConfig,
                            endDate: undefined,
                            maxOccurrences: 10
                          })}
                          className="h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500"
                        />
                        <span className="text-sm">After</span>
                        {recurrenceConfig.maxOccurrences && (
                          <>
                            <input
                              type="number"
                              min="1"
                              max="365"
                              value={recurrenceConfig.maxOccurrences}
                              onChange={(e) => setRecurrenceConfig({
                                ...recurrenceConfig,
                                maxOccurrences: parseInt(e.target.value) || 1
                              })}
                              className="input text-sm w-20 ml-2"
                            />
                            <span className="text-sm">occurrences</span>
                          </>
                        )}
                      </label>
                    </div>
                  </div>

                  {/* Exceptions */}
                  <div>
                    <label className="label text-sm font-medium">Skip these dates</label>
                    <div className="space-y-2">
                      {recurrenceConfig.exceptions && recurrenceConfig.exceptions.length > 0 && (
                        <div className="space-y-1">
                          {recurrenceConfig.exceptions.map((exception, index) => (
                            <div key={index} className="flex items-center justify-between p-2 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800">
                              <span className="text-sm text-red-700 dark:text-red-300">
                                {format(new Date(exception), 'MMM d, yyyy')}
                              </span>
                              <button
                                type="button"
                                onClick={() => setRecurrenceConfig({
                                  ...recurrenceConfig,
                                  exceptions: recurrenceConfig.exceptions?.filter((_, i) => i !== index) || []
                                })}
                                className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                                title="Remove exception"
                              >
                                <X className="h-4 w-4" />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                      <div className="flex items-center space-x-2">
                        <input
                          type="date"
                          className="input text-sm flex-1"
                          onChange={(e) => {
                            if (e.target.value) {
                              const newExceptions = [...(recurrenceConfig.exceptions || []), e.target.value]
                              setRecurrenceConfig({
                                ...recurrenceConfig,
                                exceptions: newExceptions.sort()
                              })
                              e.target.value = '' // Reset input
                            }
                          }}
                          placeholder="Add exception date"
                        />
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                          Select date to skip
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Actions */}
              <div className="flex justify-between pt-6 border-t">
                {!isNewEvent && (
                  <button
                    type="button"
                    onClick={handleDelete}
                    className="btn-outline border-red-500 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                  >
                    Delete Event
                  </button>
                )}
                <div className="ml-auto flex space-x-3">
                  <button
                    type="button"
                    onClick={() => selectEvent(null)}
                    className="btn-outline px-6 py-2"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={createMutation.isPending || updateMutation.isPending}
                    className="btn-primary px-6 py-2"
                  >
                    {createMutation.isPending || updateMutation.isPending ? 'Saving...' : 'Save Event'}
                  </button>
                </div>
              </div>
            </form>
          </div>

          {/* Preview Panel */}
          {recurrenceEnabled && showRecurrencePreview && (
            <div className="w-80 bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
                Recurrence Preview
              </h3>
              <div className="space-y-2">
                {getRecurrencePreview().map((date, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-2 bg-white dark:bg-gray-700 rounded border"
                  >
                    <span className="text-sm text-gray-900 dark:text-gray-100">
                      {format(date, 'MMM d, yyyy')}
                    </span>
                    <span className="text-xs text-gray-500 dark:text-gray-400">
                      {format(date, 'EEEE')}
                    </span>
                  </div>
                ))}
                {recurrenceConfig.maxOccurrences && recurrenceConfig.maxOccurrences > 5 && (
                  <div className="text-sm text-gray-500 dark:text-gray-400 text-center py-2">
                    ... and {recurrenceConfig.maxOccurrences - 5} more
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
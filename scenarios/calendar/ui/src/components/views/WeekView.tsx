import { useMemo } from 'react'
import { format, startOfWeek, addDays, isSameDay, isToday } from 'date-fns'
import { useCalendarStore } from '@/stores/calendarStore'
import type { Event } from '@/types'

export function WeekView() {
  const { currentDate, getCurrentViewEvents, selectEvent } = useCalendarStore()
  
  const weekStart = startOfWeek(currentDate, { weekStartsOn: 0 })
  const weekDays = Array.from({ length: 7 }, (_, i) => addDays(weekStart, i))
  const hours = Array.from({ length: 24 }, (_, i) => i)
  
  const events = getCurrentViewEvents()

  const eventsByDay = useMemo(() => {
    const byDay: Record<string, Event[]> = {}
    
    weekDays.forEach(day => {
      const dayKey = format(day, 'yyyy-MM-dd')
      byDay[dayKey] = events.filter(event => {
        const eventStart = new Date(event.startTime)
        return isSameDay(eventStart, day)
      })
    })
    
    return byDay
  }, [events, weekDays])

  const getEventPosition = (event: Event) => {
    const start = new Date(event.startTime)
    const end = new Date(event.endTime)
    
    const startHour = start.getHours() + start.getMinutes() / 60
    const duration = (end.getTime() - start.getTime()) / (1000 * 60 * 60)
    
    return {
      top: `${startHour * 60}px`,
      height: `${Math.max(duration * 60 - 2, 20)}px`
    }
  }

  const getEventColor = (eventType: Event['eventType']) => {
    const colors = {
      meeting: 'bg-blue-500 hover:bg-blue-600',
      task: 'bg-green-500 hover:bg-green-600',
      reminder: 'bg-yellow-500 hover:bg-yellow-600',
      personal: 'bg-purple-500 hover:bg-purple-600',
      other: 'bg-gray-500 hover:bg-gray-600'
    }
    return colors[eventType] || colors.other
  }

  return (
    <div className="h-full overflow-auto">
      <div className="min-w-[800px]">
        {/* Header with day names */}
        <div className="sticky top-0 z-20 border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
          <div className="grid grid-cols-8">
            <div className="border-r border-gray-200 px-2 py-3 dark:border-gray-700">
              {/* Empty cell for time column */}
            </div>
            {weekDays.map((day) => (
              <div
                key={day.toISOString()}
                className={`border-r border-gray-200 px-2 py-3 text-center dark:border-gray-700 ${
                  isToday(day) ? 'bg-primary-50 dark:bg-primary-900/20' : ''
                }`}
              >
                <div className="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {format(day, 'EEE')}
                </div>
                <div className={`text-2xl font-semibold ${
                  isToday(day) 
                    ? 'flex h-10 w-10 items-center justify-center rounded-full bg-primary-600 text-white mx-auto'
                    : 'text-gray-900 dark:text-gray-100'
                }`}>
                  {format(day, 'd')}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Time grid */}
        <div className="relative">
          <div className="grid grid-cols-8">
            {/* Time labels */}
            <div className="border-r border-gray-200 dark:border-gray-700">
              {hours.map((hour) => (
                <div
                  key={hour}
                  className="relative h-[60px] border-b border-gray-100 dark:border-gray-800"
                >
                  <span className="absolute -top-2 right-2 text-xs text-gray-500 dark:text-gray-400">
                    {format(new Date().setHours(hour, 0, 0, 0), 'ha')}
                  </span>
                </div>
              ))}
            </div>

            {/* Day columns with events */}
            {weekDays.map((day) => {
              const dayKey = format(day, 'yyyy-MM-dd')
              const dayEvents = eventsByDay[dayKey] || []
              
              return (
                <div
                  key={day.toISOString()}
                  className={`relative border-r border-gray-200 dark:border-gray-700 ${
                    isToday(day) ? 'bg-primary-50/50 dark:bg-primary-900/10' : ''
                  }`}
                >
                  {/* Hour cells */}
                  {hours.map((hour) => (
                    <div
                      key={hour}
                      className="h-[60px] border-b border-gray-100 dark:border-gray-800"
                    />
                  ))}

                  {/* Events */}
                  {dayEvents.map((event) => {
                    const position = getEventPosition(event)
                    const color = getEventColor(event.eventType)
                    
                    return (
                      <button
                        key={event.id}
                        onClick={() => selectEvent(event)}
                        className={`event-block ${color} text-white`}
                        style={{
                          top: position.top,
                          height: position.height,
                          left: '2px',
                          right: '2px',
                          position: 'absolute'
                        }}
                      >
                        <div className="overflow-hidden">
                          <div className="font-medium truncate">{event.title}</div>
                          <div className="text-xs opacity-90">
                            {format(new Date(event.startTime), 'h:mm a')}
                          </div>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )
            })}
          </div>

          {/* Current time indicator */}
          {weekDays.some(day => isToday(day)) && (
            <div
              className="absolute left-0 right-0 z-10 border-t-2 border-red-500"
              style={{
                top: `${(new Date().getHours() + new Date().getMinutes() / 60) * 60}px`
              }}
            >
              <div className="absolute -left-1 -top-1 h-3 w-3 rounded-full bg-red-500" />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
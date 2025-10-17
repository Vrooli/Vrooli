import { useMemo } from 'react'
import { 
  format, 
  startOfMonth, 
  endOfMonth, 
  startOfWeek, 
  endOfWeek,
  eachDayOfInterval,
  isSameMonth,
  isSameDay,
  isToday
} from 'date-fns'
import { useCalendarStore } from '@/stores/calendarStore'
import type { Event } from '@/types'

export function MonthView() {
  const { currentDate, getCurrentViewEvents, selectEvent } = useCalendarStore()
  
  const monthStart = startOfMonth(currentDate)
  const monthEnd = endOfMonth(currentDate)
  const calendarStart = startOfWeek(monthStart, { weekStartsOn: 0 })
  const calendarEnd = endOfWeek(monthEnd, { weekStartsOn: 0 })
  
  const days = eachDayOfInterval({ start: calendarStart, end: calendarEnd })
  const events = getCurrentViewEvents()

  const eventsByDay = useMemo(() => {
    const byDay: Record<string, Event[]> = {}
    
    days.forEach(day => {
      const dayKey = format(day, 'yyyy-MM-dd')
      byDay[dayKey] = events.filter(event => {
        const eventStart = new Date(event.startTime)
        return isSameDay(eventStart, day)
      })
    })
    
    return byDay
  }, [events, days])

  const getEventColor = (eventType: Event['eventType']) => {
    const colors = {
      meeting: 'bg-blue-500',
      task: 'bg-green-500',
      reminder: 'bg-yellow-500',
      personal: 'bg-purple-500',
      other: 'bg-gray-500'
    }
    return colors[eventType] || colors.other
  }

  return (
    <div className="h-full p-4">
      {/* Week days header */}
      <div className="grid grid-cols-7 gap-px bg-gray-200 dark:bg-gray-700">
        {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(day => (
          <div
            key={day}
            className="bg-white px-2 py-3 text-center text-sm font-semibold text-gray-700 dark:bg-gray-800 dark:text-gray-300"
          >
            {day}
          </div>
        ))}
      </div>

      {/* Calendar grid */}
      <div className="flex-1 grid grid-cols-7 gap-px bg-gray-200 dark:bg-gray-700">
        {days.map(day => {
          const dayKey = format(day, 'yyyy-MM-dd')
          const dayEvents = eventsByDay[dayKey] || []
          const isCurrentMonth = isSameMonth(day, currentDate)
          const isCurrentDay = isToday(day)
          
          return (
            <div
              key={day.toISOString()}
              className={`min-h-[100px] bg-white p-2 dark:bg-gray-800 ${
                !isCurrentMonth ? 'bg-gray-50 dark:bg-gray-900' : ''
              } ${isCurrentDay ? 'bg-primary-50 dark:bg-primary-900/20' : ''}`}
            >
              <div className="mb-1 flex items-center justify-between">
                <span className={`text-sm font-medium ${
                  !isCurrentMonth 
                    ? 'text-gray-400 dark:text-gray-600' 
                    : 'text-gray-900 dark:text-gray-100'
                } ${
                  isCurrentDay 
                    ? 'flex h-7 w-7 items-center justify-center rounded-full bg-primary-600 text-white' 
                    : ''
                }`}>
                  {format(day, 'd')}
                </span>
                {dayEvents.length > 0 && (
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {dayEvents.length} event{dayEvents.length !== 1 ? 's' : ''}
                  </span>
                )}
              </div>

              <div className="space-y-1">
                {dayEvents.slice(0, 3).map(event => (
                  <button
                    key={event.id}
                    onClick={() => selectEvent(event)}
                    className={`w-full truncate rounded px-1 py-0.5 text-left text-xs text-white ${
                      getEventColor(event.eventType)
                    } hover:opacity-90`}
                  >
                    <span className="font-medium">
                      {format(new Date(event.startTime), 'h:mm a')}
                    </span>
                    {' '}
                    {event.title}
                  </button>
                ))}
                {dayEvents.length > 3 && (
                  <button
                    className="w-full text-left text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
                    onClick={() => {
                      // Could open a day view or modal with all events
                    }}
                  >
                    +{dayEvents.length - 3} more
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
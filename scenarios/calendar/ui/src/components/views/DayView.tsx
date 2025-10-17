import { format, isToday } from 'date-fns'
import { useCalendarStore } from '@/stores/calendarStore'
import type { Event } from '@/types'

export function DayView() {
  const { currentDate, getCurrentViewEvents, selectEvent } = useCalendarStore()
  
  const hours = Array.from({ length: 24 }, (_, i) => i)
  const events = getCurrentViewEvents()
  const isCurrentDay = isToday(currentDate)

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
      <div className="min-w-[600px]">
        {/* Header */}
        <div className="sticky top-0 z-20 border-b border-gray-200 bg-white px-20 py-4 dark:border-gray-700 dark:bg-gray-800">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {format(currentDate, 'EEEE, MMMM d, yyyy')}
          </h2>
        </div>

        {/* Time grid */}
        <div className="relative">
          <div className="absolute left-0 top-0 w-16 bg-white dark:bg-gray-800">
            {hours.map((hour) => (
              <div key={hour} className="h-[60px] pr-2 text-right">
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {format(new Date().setHours(hour, 0, 0, 0), 'ha')}
                </span>
              </div>
            ))}
          </div>

          <div className="ml-16 relative">
            {/* Hour lines */}
            {hours.map((hour) => (
              <div
                key={hour}
                className="h-[60px] border-b border-gray-100 dark:border-gray-800"
              />
            ))}

            {/* Events */}
            {events.map((event) => {
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
                    left: '8px',
                    right: '8px',
                    position: 'absolute'
                  }}
                >
                  <div className="p-2">
                    <div className="font-medium">{event.title}</div>
                    <div className="text-sm opacity-90">
                      {format(new Date(event.startTime), 'h:mm a')} - {format(new Date(event.endTime), 'h:mm a')}
                    </div>
                    {event.location && (
                      <div className="text-xs opacity-75 mt-1">{event.location}</div>
                    )}
                  </div>
                </button>
              )
            })}

            {/* Current time indicator */}
            {isCurrentDay && (
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
    </div>
  )
}
import { format, isToday } from 'date-fns'
import { useCalendarStore } from '@/stores/calendarStore'
import { Clock, MapPin, FileText } from 'lucide-react'
import type { Event } from '@/types'

export function AgendaView() {
  const { getCurrentViewEvents, selectEvent } = useCalendarStore()
  
  const events = getCurrentViewEvents().sort((a, b) => 
    new Date(a.startTime).getTime() - new Date(b.startTime).getTime()
  )

  // Group events by day
  const eventsByDay = events.reduce((acc, event) => {
    const day = format(new Date(event.startTime), 'yyyy-MM-dd')
    if (!acc[day]) {
      acc[day] = []
    }
    acc[day].push(event)
    return acc
  }, {} as Record<string, Event[]>)

  const getEventColor = (eventType: Event['eventType']) => {
    const colors = {
      meeting: 'border-blue-500 bg-blue-50 dark:bg-blue-900/20',
      task: 'border-green-500 bg-green-50 dark:bg-green-900/20',
      reminder: 'border-yellow-500 bg-yellow-50 dark:bg-yellow-900/20',
      personal: 'border-purple-500 bg-purple-50 dark:bg-purple-900/20',
      other: 'border-gray-500 bg-gray-50 dark:bg-gray-900/20'
    }
    return colors[eventType] || colors.other
  }

  const getStatusBadge = (status: Event['status']) => {
    const badges = {
      scheduled: 'bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-300',
      in_progress: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/50 dark:text-yellow-300',
      completed: 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300',
      cancelled: 'bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-300'
    }
    return badges[status] || badges.scheduled
  }

  if (events.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="text-gray-500 dark:text-gray-400">No events scheduled</p>
          <p className="mt-2 text-sm text-gray-400 dark:text-gray-500">
            Create your first event to get started
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full overflow-auto p-6">
      <div className="mx-auto max-w-4xl space-y-6">
        {Object.entries(eventsByDay).map(([day, dayEvents]) => {
          const date = new Date(day)
          const isCurrentDay = isToday(date)
          
          return (
            <div key={day}>
              {/* Day Header */}
              <div className="mb-3 flex items-center space-x-3">
                <h3 className={`text-lg font-semibold ${
                  isCurrentDay ? 'text-primary-600' : 'text-gray-900 dark:text-gray-100'
                }`}>
                  {format(date, 'EEEE, MMMM d')}
                </h3>
                {isCurrentDay && (
                  <span className="rounded-full bg-primary-600 px-2 py-0.5 text-xs font-medium text-white">
                    Today
                  </span>
                )}
                <span className="text-sm text-gray-500 dark:text-gray-400">
                  {dayEvents.length} event{dayEvents.length !== 1 ? 's' : ''}
                </span>
              </div>

              {/* Events List */}
              <div className="space-y-3">
                {dayEvents.map((event) => (
                  <button
                    key={event.id}
                    onClick={() => selectEvent(event)}
                    className={`w-full rounded-lg border-l-4 p-4 text-left transition-all hover:shadow-md ${
                      getEventColor(event.eventType)
                    }`}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="mb-1 flex items-center space-x-2">
                          <h4 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                            {event.title}
                          </h4>
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                            getStatusBadge(event.status)
                          }`}>
                            {event.status.replace('_', ' ')}
                          </span>
                        </div>

                        <div className="space-y-1 text-sm text-gray-600 dark:text-gray-400">
                          <div className="flex items-center space-x-4">
                            <div className="flex items-center space-x-1">
                              <Clock className="h-4 w-4" />
                              <span>
                                {format(new Date(event.startTime), 'h:mm a')} - {format(new Date(event.endTime), 'h:mm a')}
                              </span>
                            </div>
                            {event.location && (
                              <div className="flex items-center space-x-1">
                                <MapPin className="h-4 w-4" />
                                <span>{event.location}</span>
                              </div>
                            )}
                          </div>

                          {event.description && (
                            <div className="flex items-start space-x-1 pt-1">
                              <FileText className="h-4 w-4 mt-0.5" />
                              <p className="line-clamp-2">{event.description}</p>
                            </div>
                          )}
                        </div>
                      </div>

                      <div className="ml-4">
                        <span className={`inline-block h-3 w-3 rounded-full ${
                          event.eventType === 'meeting' ? 'bg-blue-500' :
                          event.eventType === 'task' ? 'bg-green-500' :
                          event.eventType === 'reminder' ? 'bg-yellow-500' :
                          event.eventType === 'personal' ? 'bg-purple-500' :
                          'bg-gray-500'
                        }`} />
                      </div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
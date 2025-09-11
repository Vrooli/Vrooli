import { useCalendarStore } from '@/stores/calendarStore'
import { WeekView } from './views/WeekView'
import { MonthView } from './views/MonthView'
import { DayView } from './views/DayView'
import { AgendaView } from './views/AgendaView'

export function CalendarView() {
  const { viewType, isLoading, error } = useCalendarStore()

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mb-4 h-12 w-12 animate-spin rounded-full border-4 border-primary-600 border-t-transparent"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading events...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="mb-2 text-red-600">Error loading calendar</p>
          <p className="text-sm text-gray-600 dark:text-gray-400">{error}</p>
        </div>
      </div>
    )
  }

  const renderView = () => {
    switch (viewType) {
      case 'day':
        return <DayView />
      case 'week':
        return <WeekView />
      case 'month':
        return <MonthView />
      case 'agenda':
        return <AgendaView />
      default:
        return <WeekView />
    }
  }

  return (
    <div className="h-full bg-white dark:bg-gray-900">
      {renderView()}
    </div>
  )
}
import { format } from 'date-fns'
import { useCalendarStore } from '@/stores/calendarStore'
import { ChevronLeft, ChevronRight, Calendar, Plus } from 'lucide-react'

export function Header() {
  const { 
    currentDate, 
    viewType, 
    navigateDate, 
    setViewType,
    selectEvent 
  } = useCalendarStore()

  const handleCreateEvent = () => {
    selectEvent({
      id: 'new',
      userId: '',
      title: '',
      startTime: new Date().toISOString(),
      endTime: new Date(Date.now() + 3600000).toISOString(),
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      eventType: 'meeting',
      status: 'scheduled',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    })
  }

  const getDateDisplay = () => {
    switch (viewType) {
      case 'day':
        return format(currentDate, 'EEEE, MMMM d, yyyy')
      case 'week':
        const weekStart = new Date(currentDate)
        weekStart.setDate(currentDate.getDate() - currentDate.getDay())
        const weekEnd = new Date(weekStart)
        weekEnd.setDate(weekStart.getDate() + 6)
        
        if (weekStart.getMonth() === weekEnd.getMonth()) {
          return `${format(weekStart, 'MMMM d')} - ${format(weekEnd, 'd, yyyy')}`
        } else if (weekStart.getFullYear() === weekEnd.getFullYear()) {
          return `${format(weekStart, 'MMM d')} - ${format(weekEnd, 'MMM d, yyyy')}`
        } else {
          return `${format(weekStart, 'MMM d, yyyy')} - ${format(weekEnd, 'MMM d, yyyy')}`
        }
      case 'month':
        return format(currentDate, 'MMMM yyyy')
      case 'agenda':
        return format(currentDate, 'MMMM yyyy')
      default:
        return format(currentDate, 'MMMM yyyy')
    }
  }

  return (
    <header className="flex items-center justify-between border-b border-gray-200 bg-white px-6 py-4 dark:border-gray-700 dark:bg-gray-800">
      <div className="flex items-center space-x-4">
        {/* Navigation Controls */}
        <div className="flex items-center space-x-1">
          <button
            onClick={() => navigateDate('prev')}
            className="btn-ghost p-2"
            aria-label="Previous"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <button
            onClick={() => navigateDate('today')}
            className="btn-outline px-3 py-1.5"
          >
            Today
          </button>
          <button
            onClick={() => navigateDate('next')}
            className="btn-ghost p-2"
            aria-label="Next"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
        </div>

        {/* Current Date Display */}
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {getDateDisplay()}
        </h1>
      </div>

      <div className="flex items-center space-x-3">
        {/* View Type Selector */}
        <div className="flex rounded-lg border border-gray-300 dark:border-gray-600">
          {(['day', 'week', 'month', 'agenda'] as const).map((view) => (
            <button
              key={view}
              onClick={() => setViewType(view)}
              className={`px-3 py-1.5 text-sm font-medium capitalize transition-colors ${
                viewType === view
                  ? 'bg-primary-600 text-white'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700'
              } ${
                view === 'day' ? 'rounded-l-md' : ''
              } ${
                view === 'agenda' ? 'rounded-r-md' : ''
              }`}
            >
              {view}
            </button>
          ))}
        </div>

        {/* Create Event Button */}
        <button
          onClick={handleCreateEvent}
          className="btn-primary flex items-center space-x-2 px-4 py-2"
        >
          <Plus className="h-4 w-4" />
          <span>New Event</span>
        </button>
      </div>
    </header>
  )
}
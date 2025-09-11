import { useState } from 'react'
import { useCalendarStore } from '@/stores/calendarStore'
import { 
  Calendar, 
  Filter, 
  Search, 
  Settings, 
  ChevronDown,
  Clock,
  Users,
  MapPin,
  Tag
} from 'lucide-react'
import type { Event } from '@/types'

export function Sidebar() {
  const { filters, setFilters, clearFilters, user } = useCalendarStore()
  const [isFilterExpanded, setFilterExpanded] = useState(true)
  const [searchQuery, setSearchQuery] = useState(filters.searchQuery || '')

  const eventTypes: Array<{ value: Event['eventType']; label: string; color: string }> = [
    { value: 'meeting', label: 'Meeting', color: 'bg-blue-500' },
    { value: 'task', label: 'Task', color: 'bg-green-500' },
    { value: 'reminder', label: 'Reminder', color: 'bg-yellow-500' },
    { value: 'personal', label: 'Personal', color: 'bg-purple-500' },
    { value: 'other', label: 'Other', color: 'bg-gray-500' }
  ]

  const statusOptions: Array<{ value: Event['status']; label: string }> = [
    { value: 'scheduled', label: 'Scheduled' },
    { value: 'in_progress', label: 'In Progress' },
    { value: 'completed', label: 'Completed' },
    { value: 'cancelled', label: 'Cancelled' }
  ]

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setFilters({ searchQuery })
  }

  const toggleEventType = (type: Event['eventType']) => {
    const currentTypes = filters.eventTypes || []
    const newTypes = currentTypes.includes(type)
      ? currentTypes.filter(t => t !== type)
      : [...currentTypes, type]
    
    setFilters({ eventTypes: newTypes.length > 0 ? newTypes : undefined })
  }

  const toggleStatus = (status: Event['status']) => {
    const currentStatuses = filters.status || []
    const newStatuses = currentStatuses.includes(status)
      ? currentStatuses.filter(s => s !== status)
      : [...currentStatuses, status]
    
    setFilters({ status: newStatuses.length > 0 ? newStatuses : undefined })
  }

  return (
    <aside className="w-64 border-r border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
      <div className="flex h-full flex-col">
        {/* Logo/Brand */}
        <div className="flex items-center space-x-2 border-b border-gray-200 px-6 py-4 dark:border-gray-700">
          <Calendar className="h-6 w-6 text-primary-600" />
          <span className="text-xl font-semibold">Vrooli Calendar</span>
        </div>

        {/* User Info */}
        {user && (
          <div className="border-b border-gray-200 px-6 py-4 dark:border-gray-700">
            <div className="text-sm text-gray-600 dark:text-gray-400">Logged in as</div>
            <div className="font-medium text-gray-900 dark:text-gray-100">{user.displayName}</div>
            <div className="text-sm text-gray-500 dark:text-gray-400">{user.email}</div>
          </div>
        )}

        {/* Search */}
        <div className="px-4 py-4">
          <form onSubmit={handleSearch} className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search events..."
              className="input pl-10"
            />
          </form>
        </div>

        {/* Filters */}
        <div className="flex-1 overflow-y-auto px-4 pb-4">
          <div className="space-y-4">
            {/* Filter Header */}
            <div className="flex items-center justify-between">
              <button
                onClick={() => setFilterExpanded(!isFilterExpanded)}
                className="flex items-center space-x-2 text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                <Filter className="h-4 w-4" />
                <span>Filters</span>
                <ChevronDown 
                  className={`h-4 w-4 transition-transform ${
                    isFilterExpanded ? 'rotate-180' : ''
                  }`} 
                />
              </button>
              {(filters.eventTypes?.length || filters.status?.length || filters.searchQuery) && (
                <button
                  onClick={clearFilters}
                  className="text-xs text-primary-600 hover:text-primary-700"
                >
                  Clear all
                </button>
              )}
            </div>

            {isFilterExpanded && (
              <>
                {/* Event Types */}
                <div>
                  <h3 className="mb-2 text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                    Event Types
                  </h3>
                  <div className="space-y-1">
                    {eventTypes.map(({ value, label, color }) => (
                      <label
                        key={value}
                        className="flex cursor-pointer items-center space-x-2 rounded px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
                      >
                        <input
                          type="checkbox"
                          checked={filters.eventTypes?.includes(value) || false}
                          onChange={() => toggleEventType(value)}
                          className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                        />
                        <span className={`h-2 w-2 rounded-full ${color}`} />
                        <span className="text-sm text-gray-700 dark:text-gray-300">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* Status */}
                <div>
                  <h3 className="mb-2 text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                    Status
                  </h3>
                  <div className="space-y-1">
                    {statusOptions.map(({ value, label }) => (
                      <label
                        key={value}
                        className="flex cursor-pointer items-center space-x-2 rounded px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-700"
                      >
                        <input
                          type="checkbox"
                          checked={filters.status?.includes(value) || false}
                          onChange={() => toggleStatus(value)}
                          className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                        />
                        <span className="text-sm text-gray-700 dark:text-gray-300">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>

        {/* Bottom Actions */}
        <div className="border-t border-gray-200 p-4 dark:border-gray-700">
          <button className="btn-ghost flex w-full items-center justify-center space-x-2 py-2">
            <Settings className="h-4 w-4" />
            <span>Settings</span>
          </button>
        </div>
      </div>
    </aside>
  )
}
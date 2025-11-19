import { useState } from 'react'
import { useCalendarStore } from '@/stores/calendarStore'
import {
  Calendar,
  Filter,
  Search,
  Settings,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  User
} from 'lucide-react'
import type { Event } from '@/types'

export function Sidebar() {
  const { 
    filters, 
    setFilters, 
    clearFilters, 
    user,
    sidebarCollapsed,
    toggleSidebar
  } = useCalendarStore()
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
    <aside
      className={`${
        sidebarCollapsed ? 'w-16' : 'w-64'
      } hidden md:flex relative h-full flex-shrink-0 flex-col border-r border-gray-200 bg-white transition-all duration-300 dark:border-gray-700 dark:bg-gray-800`}
    >
      <button
        type="button"
        onClick={toggleSidebar}
        className="absolute -right-3 top-4 z-10 flex h-7 w-7 items-center justify-center rounded-full border border-gray-200 bg-white text-gray-500 shadow-sm transition hover:bg-gray-100 hover:text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300 dark:hover:bg-gray-800"
        aria-label={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      >
        {sidebarCollapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
      </button>

      <div className="flex h-full flex-col">
        {/* Logo/Brand */}
        <div className={`flex items-center border-b border-gray-200 py-4 dark:border-gray-700 ${sidebarCollapsed ? 'justify-center px-3' : 'space-x-2 px-6'}`}>
          <Calendar className="h-6 w-6 text-primary-600 flex-shrink-0" />
          {!sidebarCollapsed && (
            <span className="text-xl font-semibold transition-opacity duration-200">Vrooli Calendar</span>
          )}
        </div>

        {/* User Info */}
        {user && (
          <div className={`border-b border-gray-200 py-4 dark:border-gray-700 ${sidebarCollapsed ? 'px-3' : 'px-6'}`}>
            {sidebarCollapsed ? (
              <div className="flex justify-center">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/20">
                  <User className="h-4 w-4 text-primary-600" />
                </div>
              </div>
            ) : (
              <div className="transition-opacity duration-200">
                <div className="text-sm text-gray-600 dark:text-gray-400">Logged in as</div>
                <div className="font-medium text-gray-900 dark:text-gray-100 truncate">{user.displayName}</div>
                <div className="text-sm text-gray-500 dark:text-gray-400 truncate">{user.email}</div>
              </div>
            )}
          </div>
        )}

        {/* Search */}
        {!sidebarCollapsed && (
          <div className="px-4 py-4 transition-opacity duration-200">
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
        )}

        {/* Filters */}
        <div className={`flex-1 overflow-y-auto pb-4 ${sidebarCollapsed ? 'px-2' : 'px-4'}`}>
          {sidebarCollapsed ? (
            /* Collapsed Filter Icons */
            <div className="space-y-3 py-4">
              <div className="flex justify-center">
                <button
                  className="flex h-10 w-10 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                  title="Search"
                >
                  <Search className="h-5 w-5" />
                </button>
              </div>
              <div className="flex justify-center">
                <button
                  className="flex h-10 w-10 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                  title="Filters"
                >
                  <Filter className="h-5 w-5" />
                </button>
              </div>
            </div>
          ) : (
            /* Expanded Filters */
            <div className="space-y-4 transition-opacity duration-200">
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
          )}
        </div>

        {/* Bottom Actions */}
        <div className={`border-t border-gray-200 p-4 dark:border-gray-700 ${sidebarCollapsed ? 'px-2' : ''}`}>
          {sidebarCollapsed ? (
            <div className="flex justify-center">
              <button 
                className="flex h-10 w-10 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                title="Settings"
              >
                <Settings className="h-5 w-5" />
              </button>
            </div>
          ) : (
            <button className="btn-ghost flex w-full items-center justify-center space-x-2 py-2 transition-opacity duration-200">
              <Settings className="h-4 w-4" />
              <span>Settings</span>
            </button>
          )}
        </div>
      </div>
    </aside>
  )
}

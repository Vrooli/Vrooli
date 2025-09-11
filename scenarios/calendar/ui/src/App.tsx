import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CalendarView } from './components/CalendarView'
import { Header } from './components/Header'
import { Sidebar } from './components/Sidebar'
import { EventModal } from './components/EventModal'
import { ChatPanel } from './components/ChatPanel'
import { ErrorBoundary } from './components/ErrorBoundary'
import { useCalendarStore } from './stores/calendarStore'
import { api } from './services/api'
import toast from 'react-hot-toast'

function App() {
  const { 
    setUser, 
    setEvents, 
    setLoading, 
    setError,
    selectedEvent 
  } = useCalendarStore()

  // Fetch user data
  const { data: user, error: userError } = useQuery({
    queryKey: ['user'],
    queryFn: api.validateToken,
    retry: false
  })

  // Fetch events
  const { data: eventsData, isLoading, error: eventsError } = useQuery({
    queryKey: ['events'],
    queryFn: () => api.getEvents(),
    enabled: !!user,
    refetchInterval: 60000 // Refetch every minute
  })

  // Handle user data
  useEffect(() => {
    if (user) {
      setUser(user)
    }
  }, [user, setUser])

  // Handle user errors
  useEffect(() => {
    if (userError) {
      toast.error('Authentication failed. Please log in.')
      // In production, redirect to login
    }
  }, [userError])

  // Handle events data
  useEffect(() => {
    if (eventsData) {
      setEvents(eventsData.events)
      setLoading(false)
    }
  }, [eventsData, setEvents, setLoading])

  // Handle events errors
  useEffect(() => {
    if (eventsError) {
      setError(eventsError.message)
      setLoading(false)
      toast.error('Failed to load events')
    }
  }, [eventsError, setError, setLoading])

  useEffect(() => {
    setLoading(isLoading)
  }, [isLoading, setLoading])

  return (
    <ErrorBoundary
      onError={(error, errorInfo) => {
        console.error('Application error:', error, errorInfo)
        // In production, report to error tracking service
      }}
    >
      <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
        {/* Sidebar */}
        <ErrorBoundary fallback={
          <div className="w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex items-center justify-center">
            <p className="text-sm text-gray-500">Sidebar unavailable</p>
          </div>
        }>
          <Sidebar />
        </ErrorBoundary>

        {/* Main Content */}
        <div className="flex flex-1 flex-col overflow-hidden">
          {/* Header */}
          <ErrorBoundary fallback={
            <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-4 py-3">
              <p className="text-sm text-gray-500">Header unavailable</p>
            </div>
          }>
            <Header />
          </ErrorBoundary>

          {/* Calendar View */}
          <main className="flex-1 overflow-auto">
            <ErrorBoundary fallback={
              <div className="flex items-center justify-center h-full">
                <div className="text-center">
                  <p className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                    Calendar temporarily unavailable
                  </p>
                  <p className="text-sm text-gray-500">
                    Please refresh the page to try again
                  </p>
                  <button 
                    onClick={() => window.location.reload()}
                    className="mt-4 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md"
                  >
                    Refresh
                  </button>
                </div>
              </div>
            }>
              <CalendarView />
            </ErrorBoundary>
          </main>
        </div>

        {/* Chat Panel */}
        <ErrorBoundary fallback={
          <div className="w-80 bg-white dark:bg-gray-800 border-l border-gray-200 dark:border-gray-700 flex items-center justify-center">
            <p className="text-sm text-gray-500">Chat unavailable</p>
          </div>
        }>
          <ChatPanel />
        </ErrorBoundary>

        {/* Event Modal */}
        {selectedEvent && (
          <ErrorBoundary>
            <EventModal />
          </ErrorBoundary>
        )}
      </div>
    </ErrorBoundary>
  )
}

export default App
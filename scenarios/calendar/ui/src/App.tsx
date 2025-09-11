import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CalendarView } from './components/CalendarView'
import { Header } from './components/Header'
import { Sidebar } from './components/Sidebar'
import { EventModal } from './components/EventModal'
import { ChatPanel } from './components/ChatPanel'
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
  const { data: user } = useQuery({
    queryKey: ['user'],
    queryFn: api.validateToken,
    retry: false,
    onSuccess: (data) => setUser(data),
    onError: () => {
      toast.error('Authentication failed. Please log in.')
      // In production, redirect to login
    }
  })

  // Fetch events
  const { isLoading, error } = useQuery({
    queryKey: ['events'],
    queryFn: () => api.getEvents(),
    enabled: !!user,
    refetchInterval: 60000, // Refetch every minute
    onSuccess: (data) => {
      setEvents(data.events)
      setLoading(false)
    },
    onError: (err: Error) => {
      setError(err.message)
      setLoading(false)
      toast.error('Failed to load events')
    }
  })

  useEffect(() => {
    setLoading(isLoading)
  }, [isLoading, setLoading])

  useEffect(() => {
    if (error) {
      setError(error.message)
    }
  }, [error, setError])

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <Sidebar />

      {/* Main Content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <Header />

        {/* Calendar View */}
        <main className="flex-1 overflow-auto">
          <CalendarView />
        </main>
      </div>

      {/* Chat Panel */}
      <ChatPanel />

      {/* Event Modal */}
      {selectedEvent && <EventModal />}
    </div>
  )
}

export default App
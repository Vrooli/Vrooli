import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import type { Event, CalendarView, CalendarFilters, User } from '@/types'
import { startOfWeek, endOfWeek, startOfMonth, endOfMonth, startOfDay, endOfDay } from 'date-fns'

const sanitizeEvents = (events: Event[] | null | undefined): Event[] => {
  if (!Array.isArray(events)) {
    return []
  }

  return events.filter((event): event is Event => Boolean(event))
}

const sortByStartTime = (events: Event[]): Event[] =>
  [...events].sort(
    (a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime()
  )

interface CalendarState {
  // User
  user: User | null
  setUser: (user: User | null) => void

  // Events
  events: Event[]
  selectedEvent: Event | null
  isLoading: boolean
  error: string | null

  // View
  viewType: CalendarView['type']
  currentDate: Date
  filters: CalendarFilters

  // UI State
  sidebarCollapsed: boolean
  setSidebarCollapsed: (collapsed: boolean) => void
  toggleSidebar: () => void

  // Auth
  authRequired: boolean
  authLoginUrl: string | null
  setAuthRequired: (required: boolean, loginUrl?: string | null) => void

  // Actions
  setEvents: (events: Event[] | null | undefined) => void
  addEvent: (event: Event) => void
  updateEvent: (id: string, event: Partial<Event>) => void
  deleteEvent: (id: string) => void
  selectEvent: (event: Event | null) => void
  
  setViewType: (type: CalendarView['type']) => void
  setCurrentDate: (date: Date) => void
  navigateDate: (direction: 'prev' | 'next' | 'today') => void
  
  setFilters: (filters: Partial<CalendarFilters>) => void
  clearFilters: () => void
  
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void

  // Computed
  getFilteredEvents: () => Event[]
  getEventsInRange: (start: Date, end: Date) => Event[]
  getCurrentViewEvents: () => Event[]
}

export const useCalendarStore = create<CalendarState>()(
  devtools(
    (set, get) => ({
      // Initial state
      user: null,
      events: [],
      selectedEvent: null,
      isLoading: false,
      error: null,
      viewType: 'month',
      currentDate: new Date(),
      filters: {},
      sidebarCollapsed: false,
      authRequired: false,
      authLoginUrl: null,

      // User actions
      setUser: (user) => set({ user }),

      setAuthRequired: (required, loginUrl) =>
        set({
          authRequired: required,
          authLoginUrl: loginUrl ?? null
        }),

      // UI State actions
      setSidebarCollapsed: (sidebarCollapsed) => set({ sidebarCollapsed }),
      toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      // Event actions
      setEvents: (events) => set({ events: sortByStartTime(sanitizeEvents(events)) }),
      
      addEvent: (event) => set((state) => ({
        events: sortByStartTime([...sanitizeEvents(state.events), event])
      })),
      
      updateEvent: (id, updatedEvent) => set((state) => ({
        events: sortByStartTime(
          sanitizeEvents(state.events).map((e) => 
            e.id === id ? { ...e, ...updatedEvent } : e
          )
        ),
        selectedEvent: state.selectedEvent?.id === id 
          ? { ...state.selectedEvent, ...updatedEvent }
          : state.selectedEvent
      })),
      
      deleteEvent: (id) => set((state) => ({
        events: sanitizeEvents(state.events).filter((e) => e.id !== id),
        selectedEvent: state.selectedEvent?.id === id ? null : state.selectedEvent
      })),
      
      selectEvent: (event) => set({ selectedEvent: event }),

      // View actions
      setViewType: (viewType) => set({ viewType }),
      
      setCurrentDate: (currentDate) => set({ currentDate }),
      
      navigateDate: (direction) => {
        const { currentDate, viewType } = get()
        let newDate = new Date(currentDate)
        
        if (direction === 'today') {
          newDate = new Date()
        } else {
          const increment = direction === 'next' ? 1 : -1
          
          switch (viewType) {
            case 'day':
              newDate.setDate(newDate.getDate() + increment)
              break
            case 'week':
              newDate.setDate(newDate.getDate() + (7 * increment))
              break
            case 'month':
              newDate.setMonth(newDate.getMonth() + increment)
              break
            case 'agenda':
              newDate.setDate(newDate.getDate() + (7 * increment))
              break
          }
        }
        
        set({ currentDate: newDate })
      },

      // Filter actions
      setFilters: (filters) => set((state) => ({
        filters: { ...state.filters, ...filters }
      })),
      
      clearFilters: () => set({ filters: {} }),

      // Loading/Error actions
      setLoading: (isLoading) => set({ isLoading }),
      setError: (error) => set({ error }),

      // Computed getters
      getFilteredEvents: () => {
        const { filters } = get()
        const events = sanitizeEvents(get().events)

        return events.filter((event) => {
          // Filter by event type
          if (filters.eventTypes?.length && !filters.eventTypes.includes(event.eventType)) {
            return false
          }
          
          // Filter by status
          if (filters.status?.length && !filters.status.includes(event.status)) {
            return false
          }
          
          // Filter by search query
          if (filters.searchQuery) {
            const query = filters.searchQuery.toLowerCase()
            const matchesTitle = event.title.toLowerCase().includes(query)
            const matchesDescription = event.description?.toLowerCase().includes(query)
            const matchesLocation = event.location?.toLowerCase().includes(query)
            
            if (!matchesTitle && !matchesDescription && !matchesLocation) {
              return false
            }
          }
          
          // Filter by date range
          if (filters.dateRange) {
            const eventStart = new Date(event.startTime)
            const eventEnd = new Date(event.endTime)
            
            if (eventEnd < filters.dateRange.start || eventStart > filters.dateRange.end) {
              return false
            }
          }
          
          return true
        })
      },

      getEventsInRange: (start: Date, end: Date) => {
        const filteredEvents = get().getFilteredEvents()
        
        return filteredEvents.filter((event) => {
          const eventStart = new Date(event.startTime)
          const eventEnd = new Date(event.endTime)
          
          // Check if event overlaps with the range
          return eventEnd >= start && eventStart <= end
        })
      },

      getCurrentViewEvents: () => {
        const { viewType, currentDate } = get()
        let start: Date
        let end: Date
        
        switch (viewType) {
          case 'day':
            start = startOfDay(currentDate)
            end = endOfDay(currentDate)
            break
          case 'week':
            start = startOfWeek(currentDate, { weekStartsOn: 0 })
            end = endOfWeek(currentDate, { weekStartsOn: 0 })
            break
          case 'month':
            start = startOfMonth(currentDate)
            end = endOfMonth(currentDate)
            break
          case 'agenda':
            start = startOfDay(currentDate)
            end = new Date(currentDate)
            end.setDate(end.getDate() + 14) // Show 2 weeks in agenda view
            break
          default:
            start = startOfWeek(currentDate, { weekStartsOn: 0 })
            end = endOfWeek(currentDate, { weekStartsOn: 0 })
        }
        
        return get().getEventsInRange(start, end)
      }
    }),
    {
      name: 'calendar-store'
    }
  )
)

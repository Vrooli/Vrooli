import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { format } from 'date-fns'
import { X, Calendar, Clock, MapPin, Users, Bell, Repeat } from 'lucide-react'
import { useCalendarStore } from '@/stores/calendarStore'
import { api } from '@/services/api'
import toast from 'react-hot-toast'
import type { CreateEventRequest, Event } from '@/types'

export function EventModal() {
  const queryClient = useQueryClient()
  const { selectedEvent, selectEvent, addEvent, updateEvent, deleteEvent } = useCalendarStore()
  
  const isNewEvent = selectedEvent?.id === 'new'
  
  const [formData, setFormData] = useState<CreateEventRequest>({
    title: '',
    description: '',
    startTime: '',
    endTime: '',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    location: '',
    eventType: 'meeting',
    reminders: [{ minutesBefore: 15, notificationType: 'email' }]
  })

  useEffect(() => {
    if (selectedEvent) {
      setFormData({
        title: selectedEvent.title || '',
        description: selectedEvent.description || '',
        startTime: selectedEvent.startTime ? format(new Date(selectedEvent.startTime), "yyyy-MM-dd'T'HH:mm") : '',
        endTime: selectedEvent.endTime ? format(new Date(selectedEvent.endTime), "yyyy-MM-dd'T'HH:mm") : '',
        timezone: selectedEvent.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
        location: selectedEvent.location || '',
        eventType: selectedEvent.eventType || 'meeting'
      })
    }
  }, [selectedEvent])

  const createMutation = useMutation({
    mutationFn: api.createEvent,
    onSuccess: (event) => {
      addEvent(event)
      toast.success('Event created successfully')
      selectEvent(null)
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: () => {
      toast.error('Failed to create event')
    }
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateEventRequest> }) => 
      api.updateEvent(id, data),
    onSuccess: (event) => {
      updateEvent(event.id, event)
      toast.success('Event updated successfully')
      selectEvent(null)
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: () => {
      toast.error('Failed to update event')
    }
  })

  const deleteMutation = useMutation({
    mutationFn: api.deleteEvent,
    onSuccess: () => {
      if (selectedEvent) {
        deleteEvent(selectedEvent.id)
      }
      toast.success('Event deleted successfully')
      selectEvent(null)
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: () => {
      toast.error('Failed to delete event')
    }
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!formData.title || !formData.startTime || !formData.endTime) {
      toast.error('Please fill in required fields')
      return
    }

    const eventData = {
      ...formData,
      startTime: new Date(formData.startTime).toISOString(),
      endTime: new Date(formData.endTime).toISOString()
    }

    if (isNewEvent) {
      createMutation.mutate(eventData)
    } else if (selectedEvent) {
      updateMutation.mutate({ id: selectedEvent.id, data: eventData })
    }
  }

  const handleDelete = () => {
    if (selectedEvent && selectedEvent.id !== 'new') {
      if (confirm('Are you sure you want to delete this event?')) {
        deleteMutation.mutate(selectedEvent.id)
      }
    }
  }

  if (!selectedEvent) return null

  return (
    <div className="modal-overlay animate-fade-in" onClick={() => selectEvent(null)}>
      <div className="modal-content animate-slide-in" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            {isNewEvent ? 'New Event' : 'Edit Event'}
          </h2>
          <button
            onClick={() => selectEvent(null)}
            className="btn-ghost p-1"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Title */}
          <div>
            <label className="label">Title *</label>
            <input
              type="text"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              className="input"
              placeholder="Event title"
              required
            />
          </div>

          {/* Date & Time */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label flex items-center space-x-1">
                <Clock className="h-4 w-4" />
                <span>Start *</span>
              </label>
              <input
                type="datetime-local"
                value={formData.startTime}
                onChange={(e) => setFormData({ ...formData, startTime: e.target.value })}
                className="input"
                required
              />
            </div>
            <div>
              <label className="label flex items-center space-x-1">
                <Clock className="h-4 w-4" />
                <span>End *</span>
              </label>
              <input
                type="datetime-local"
                value={formData.endTime}
                onChange={(e) => setFormData({ ...formData, endTime: e.target.value })}
                className="input"
                required
              />
            </div>
          </div>

          {/* Location */}
          <div>
            <label className="label flex items-center space-x-1">
              <MapPin className="h-4 w-4" />
              <span>Location</span>
            </label>
            <input
              type="text"
              value={formData.location}
              onChange={(e) => setFormData({ ...formData, location: e.target.value })}
              className="input"
              placeholder="Add location"
            />
          </div>

          {/* Event Type */}
          <div>
            <label className="label">Event Type</label>
            <select
              value={formData.eventType}
              onChange={(e) => setFormData({ ...formData, eventType: e.target.value as Event['eventType'] })}
              className="input"
            >
              <option value="meeting">Meeting</option>
              <option value="task">Task</option>
              <option value="reminder">Reminder</option>
              <option value="personal">Personal</option>
              <option value="other">Other</option>
            </select>
          </div>

          {/* Description */}
          <div>
            <label className="label">Description</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="input min-h-[100px] resize-none"
              placeholder="Add description"
            />
          </div>

          {/* Actions */}
          <div className="flex justify-between pt-4">
            {!isNewEvent && (
              <button
                type="button"
                onClick={handleDelete}
                className="btn-outline border-red-500 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
              >
                Delete
              </button>
            )}
            <div className="ml-auto flex space-x-3">
              <button
                type="button"
                onClick={() => selectEvent(null)}
                className="btn-outline px-4 py-2"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={createMutation.isPending || updateMutation.isPending}
                className="btn-primary px-4 py-2"
              >
                {createMutation.isPending || updateMutation.isPending ? 'Saving...' : 'Save'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
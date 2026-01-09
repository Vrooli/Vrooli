import { useCallback, useEffect, useMemo, useRef } from 'react';
import FullCalendar from '@fullcalendar/react';
import dayGridPlugin from '@fullcalendar/daygrid';
import timeGridPlugin from '@fullcalendar/timegrid';
import interactionPlugin from '@fullcalendar/interaction';
import type { DateClickArg } from '@fullcalendar/interaction';
import type { DatesSetArg, EventInput, EventClickArg } from '@fullcalendar/core';
import { useScheduleStore, type ScheduleOccurrence } from '../../../../../stores/scheduleStore';
import { Loader2 } from 'lucide-react';
import './calendarStyles.css';

interface CalendarViewProps {
  onCreateSchedule: (date: Date) => void;
  onEditSchedule: (scheduleId: string) => void;
}

// Color palette for different schedules (consistent colors per schedule)
const SCHEDULE_COLORS = [
  'var(--flow-accent)',
  '#10b981', // emerald
  '#f59e0b', // amber
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#06b6d4', // cyan
  '#f97316', // orange
  '#84cc16', // lime
];

function getScheduleColor(scheduleId: string): string {
  // Generate consistent color based on schedule ID
  let hash = 0;
  for (let i = 0; i < scheduleId.length; i++) {
    hash = scheduleId.charCodeAt(i) + ((hash << 5) - hash);
  }
  return SCHEDULE_COLORS[Math.abs(hash) % SCHEDULE_COLORS.length];
}

export function CalendarView({ onCreateSchedule, onEditSchedule }: CalendarViewProps) {
  const calendarRef = useRef<FullCalendar>(null);
  const { occurrences, aggregates, isLoadingOccurrences, fetchOccurrences } = useScheduleStore();

  // Convert occurrences to FullCalendar events
  const events = useMemo((): EventInput[] => {
    const eventList: EventInput[] = [];

    // Group occurrences by schedule for aggregate detection
    const occurrencesBySchedule = new Map<string, ScheduleOccurrence[]>();
    for (const occ of occurrences) {
      const existing = occurrencesBySchedule.get(occ.schedule_id) || [];
      existing.push(occ);
      occurrencesBySchedule.set(occ.schedule_id, existing);
    }

    // Add regular occurrence events
    for (const occ of occurrences) {
      const aggregate = aggregates[occ.schedule_id];
      const color = getScheduleColor(occ.schedule_id);

      eventList.push({
        id: `${occ.schedule_id}-${occ.run_at}`,
        title: aggregate?.truncated
          ? `${occ.schedule_name} (${aggregate.total_runs} runs)`
          : occ.schedule_name,
        start: occ.run_at,
        allDay: false,
        backgroundColor: color,
        borderColor: color,
        textColor: '#ffffff',
        extendedProps: {
          scheduleId: occ.schedule_id,
          workflowName: occ.workflow_name,
          cronExpression: occ.cron_expression,
          isAggregate: !!aggregate?.truncated,
          totalRuns: aggregate?.total_runs,
        },
      });
    }

    return eventList;
  }, [occurrences, aggregates]);

  // Handle date range changes (navigation)
  const handleDatesSet = useCallback((arg: DatesSetArg) => {
    fetchOccurrences(arg.start, arg.end);
  }, [fetchOccurrences]);

  // Handle clicking on a date (to create new schedule)
  const handleDateClick = useCallback((arg: DateClickArg) => {
    onCreateSchedule(arg.date);
  }, [onCreateSchedule]);

  // Handle clicking on an event (to edit schedule)
  const handleEventClick = useCallback((arg: EventClickArg) => {
    const scheduleId = arg.event.extendedProps.scheduleId as string;
    if (scheduleId) {
      onEditSchedule(scheduleId);
    }
  }, [onEditSchedule]);

  // Initial load
  useEffect(() => {
    const now = new Date();
    const start = new Date(now.getFullYear(), now.getMonth(), 1);
    const end = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59);
    fetchOccurrences(start, end);
  }, [fetchOccurrences]);

  return (
    <div className="calendar-container relative">
      {isLoadingOccurrences && (
        <div className="absolute inset-0 bg-flow-bg/50 flex items-center justify-center z-10 rounded-lg">
          <Loader2 className="w-8 h-8 animate-spin text-flow-accent" />
        </div>
      )}
      <FullCalendar
        ref={calendarRef}
        plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin]}
        initialView="dayGridMonth"
        headerToolbar={{
          left: 'prev,next today',
          center: 'title',
          right: 'dayGridMonth,timeGridWeek,timeGridDay',
        }}
        events={events}
        datesSet={handleDatesSet}
        dateClick={handleDateClick}
        eventClick={handleEventClick}
        nowIndicator={true}
        dayMaxEvents={3}
        eventDisplay="block"
        height="auto"
        aspectRatio={1.8}
        buttonText={{
          today: 'Today',
          month: 'Month',
          week: 'Week',
          day: 'Day',
        }}
        slotMinTime="00:00:00"
        slotMaxTime="24:00:00"
        allDaySlot={false}
        eventTimeFormat={{
          hour: 'numeric',
          minute: '2-digit',
          meridiem: 'short',
        }}
        eventContent={(arg) => {
          const isAggregate = arg.event.extendedProps.isAggregate;
          const totalRuns = arg.event.extendedProps.totalRuns;

          return (
            <div className="fc-event-content px-1 py-0.5 text-xs truncate">
              <span className="font-medium">{arg.event.title}</span>
              {isAggregate && totalRuns && (
                <span className="ml-1 opacity-75">
                  ({totalRuns})
                </span>
              )}
            </div>
          );
        }}
      />
    </div>
  );
}

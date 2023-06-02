export const scheduleRecurrence_update = {
  "schedule": {
    "labels": {
      "__typename": "Label"
    },
    "id": true,
    "created_at": true,
    "updated_at": true,
    "startTime": true,
    "endTime": true,
    "timezone": true,
    "__typename": "Schedule"
  },
  "id": true,
  "recurrenceType": true,
  "interval": true,
  "dayOfWeek": true,
  "dayOfMonth": true,
  "month": true,
  "endDate": true,
  "__typename": "ScheduleRecurrence"
} as const;

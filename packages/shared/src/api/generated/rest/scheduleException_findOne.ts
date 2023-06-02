export const scheduleException_findOne = {
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
    "recurrences": {
      "id": true,
      "recurrenceType": true,
      "interval": true,
      "dayOfWeek": true,
      "dayOfMonth": true,
      "month": true,
      "endDate": true,
      "__typename": "ScheduleRecurrence"
    },
    "__typename": "Schedule"
  },
  "id": true,
  "originalStartTime": true,
  "newStartTime": true,
  "newEndTime": true,
  "__typename": "ScheduleException"
};

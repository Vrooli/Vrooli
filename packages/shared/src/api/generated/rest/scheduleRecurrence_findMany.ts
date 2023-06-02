export const scheduleRecurrence_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "schedule": {
        "labels": {},
        "id": true,
        "created_at": true,
        "updated_at": true,
        "startTime": true,
        "endTime": true,
        "timezone": true
      },
      "id": true,
      "recurrenceType": true,
      "interval": true,
      "dayOfWeek": true,
      "dayOfMonth": true,
      "month": true,
      "endDate": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ScheduleRecurrence"
};

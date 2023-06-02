export const scheduleException_findMany = {
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
        "timezone": true,
        "recurrences": {
          "id": true,
          "recurrenceType": true,
          "interval": true,
          "dayOfWeek": true,
          "dayOfMonth": true,
          "month": true,
          "endDate": true
        }
      },
      "id": true,
      "originalStartTime": true,
      "newStartTime": true,
      "newEndTime": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ScheduleException"
} as const;

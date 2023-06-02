export const schedule_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "labels": {},
      "id": true,
      "created_at": true,
      "updated_at": true,
      "startTime": true,
      "endTime": true,
      "timezone": true,
      "exceptions": {
        "id": true,
        "originalStartTime": true,
        "newStartTime": true,
        "newEndTime": true
      },
      "recurrences": {
        "id": true,
        "recurrenceType": true,
        "interval": true,
        "dayOfWeek": true,
        "dayOfMonth": true,
        "month": true,
        "endDate": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Schedule"
};

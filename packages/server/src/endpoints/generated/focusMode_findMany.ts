export const focusMode_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "labels": {
        "id": true,
        "color": true,
        "label": true
      },
      "schedule": {
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
      },
      "id": true,
      "name": true,
      "description": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "FocusMode"
} as const;

export const runProject_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "projectVersion": {
        "id": true,
        "complexity": true,
        "isLatest": true,
        "isPrivate": true,
        "versionIndex": true,
        "versionLabel": true,
        "root": {
          "id": true,
          "isPrivate": true
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        }
      },
      "id": true,
      "isPrivate": true,
      "completedComplexity": true,
      "contextSwitches": true,
      "startedAt": true,
      "timeElapsed": true,
      "completedAt": true,
      "name": true,
      "status": true,
      "stepsCount": true,
      "organization": {},
      "schedule": {
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
      },
      "user": {},
      "you": {
        "canDelete": true,
        "canUpdate": true,
        "canRead": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "RunProject"
} as const;

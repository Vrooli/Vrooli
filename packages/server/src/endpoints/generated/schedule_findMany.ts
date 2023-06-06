export const schedule_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "labels": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "color": true,
        "label": true,
        "owner": {
          "Organization": {
            "id": true,
            "handle": true,
            "you": {
              "canAddMembers": true,
              "canDelete": true,
              "canBookmark": true,
              "canReport": true,
              "canUpdate": true,
              "canRead": true,
              "isBookmarked": true,
              "isViewed": true,
              "yourMembership": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "isAdmin": true,
                "permissions": true
              }
            }
          },
          "User": {
            "id": true,
            "isBot": true,
            "name": true,
            "handle": true
          }
        },
        "you": {
          "canDelete": true,
          "canUpdate": true
        }
      },
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
} as const;

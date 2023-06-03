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
      "organization": {
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
      "schedule": {
        "labels": {
          "apisCount": true,
          "focusModesCount": true,
          "issuesCount": true,
          "meetingsCount": true,
          "notesCount": true,
          "projectsCount": true,
          "routinesCount": true,
          "schedulesCount": true,
          "smartContractsCount": true,
          "standardsCount": true,
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
      },
      "user": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true
      },
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

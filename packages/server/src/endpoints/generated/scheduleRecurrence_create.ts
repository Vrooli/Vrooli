export const scheduleRecurrence_create = {
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
        "User": {
          "id": true,
          "isBot": true,
          "name": true,
          "handle": true,
          "__typename": "User"
        },
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
          },
          "__typename": "Organization"
        }
      },
      "you": {
        "canDelete": true,
        "canUpdate": true
      },
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

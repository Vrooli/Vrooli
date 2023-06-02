export const runProject_findOne = {
  "projectVersion": {
    "root": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "isPrivate": true,
      "issuesCount": true,
      "labels": {
        "__typename": "Label"
      },
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
      "permissions": true,
      "questionsCount": true,
      "score": true,
      "bookmarks": true,
      "tags": {
        "__typename": "Tag"
      },
      "transfersCount": true,
      "views": true,
      "you": {
        "canDelete": true,
        "canBookmark": true,
        "canTransfer": true,
        "canUpdate": true,
        "canRead": true,
        "canReact": true,
        "isBookmarked": true,
        "isViewed": true,
        "reaction": true
      },
      "__typename": "Project"
    },
    "translations": {
      "id": true,
      "language": true,
      "description": true,
      "name": true
    },
    "id": true,
    "created_at": true,
    "updated_at": true,
    "directoriesCount": true,
    "isLatest": true,
    "isPrivate": true,
    "reportsCount": true,
    "runProjectsCount": true,
    "simplicity": true,
    "versionIndex": true,
    "versionLabel": true,
    "__typename": "ProjectVersion"
  },
  "steps": {
    "id": true,
    "order": true,
    "contextSwitches": true,
    "startedAt": true,
    "timeElapsed": true,
    "completedAt": true,
    "name": true,
    "status": true,
    "step": true,
    "directory": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "childOrder": true,
      "isRoot": true,
      "projectVersion": {
        "id": true,
        "complexity": true,
        "isLatest": true,
        "isPrivate": true,
        "versionIndex": true,
        "versionLabel": true,
        "root": {
          "id": true,
          "isPrivate": true,
          "__typename": "Project"
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "__typename": "ProjectVersion"
      },
      "__typename": "ProjectVersionDirectory"
    },
    "__typename": "RunProjectStep"
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
    "__typename": "Organization"
  },
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
    "exceptions": {
      "id": true,
      "originalStartTime": true,
      "newStartTime": true,
      "newEndTime": true,
      "__typename": "ScheduleException"
    },
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
  "user": {
    "__typename": "User"
  },
  "you": {
    "canDelete": true,
    "canUpdate": true,
    "canRead": true
  },
  "__typename": "RunProject"
} as const;

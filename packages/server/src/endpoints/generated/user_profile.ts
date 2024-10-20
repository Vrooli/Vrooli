export const user_profile = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "bannerImage": true,
  "handle": true,
  "isPrivate": true,
  "isPrivateApis": true,
  "isPrivateApisCreated": true,
  "isPrivateMemberships": true,
  "isPrivateProjects": true,
  "isPrivateProjectsCreated": true,
  "isPrivatePullRequests": true,
  "isPrivateQuestionsAnswered": true,
  "isPrivateQuestionsAsked": true,
  "isPrivateQuizzesCreated": true,
  "isPrivateRoles": true,
  "isPrivateRoutines": true,
  "isPrivateRoutinesCreated": true,
  "isPrivateStandards": true,
  "isPrivateStandardsCreated": true,
  "isPrivateTeamsCreated": true,
  "isPrivateBookmarks": true,
  "isPrivateVotes": true,
  "name": true,
  "notificationSettings": true,
  "profileImage": true,
  "theme": true,
  "emails": {
    "id": true,
    "emailAddress": true,
    "verified": true,
    "__typename": "Email"
  },
  "focusModes": {
    "id": true,
    "name": true,
    "description": true,
    "you": {
      "canDelete": true,
      "canRead": true,
      "canUpdate": true
    },
    "filters": {
      "id": true,
      "filterType": true,
      "tag": {
        "id": true,
        "created_at": true,
        "tag": true,
        "bookmarks": true,
        "translations": {
          "id": true,
          "language": true,
          "description": true
        },
        "you": {
          "isOwn": true,
          "isBookmarked": true
        },
        "__typename": "Tag"
      },
      "focusMode": {
        "id": true,
        "name": true,
        "description": true,
        "you": {
          "canDelete": true,
          "canRead": true,
          "canUpdate": true
        },
        "labels": {
          "id": true,
          "color": true,
          "label": true,
          "__typename": "Label"
        },
        "reminderList": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "reminders": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "name": true,
            "description": true,
            "dueDate": true,
            "index": true,
            "isComplete": true,
            "reminderItems": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "name": true,
              "description": true,
              "dueDate": true,
              "index": true,
              "isComplete": true,
              "__typename": "ReminderItem"
            },
            "__typename": "Reminder"
          },
          "__typename": "ReminderList"
        },
        "resourceList": {
          "id": true,
          "created_at": true,
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
          },
          "resources": {
            "id": true,
            "index": true,
            "link": true,
            "usedFor": true,
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "name": true
            },
            "__typename": "Resource"
          },
          "__typename": "ResourceList"
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
        "__typename": "FocusMode"
      },
      "__typename": "FocusModeFilter"
    },
    "labels": {
      "id": true,
      "color": true,
      "label": true,
      "__typename": "Label"
    },
    "reminderList": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "reminders": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "name": true,
        "description": true,
        "dueDate": true,
        "index": true,
        "isComplete": true,
        "reminderItems": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "name": true,
          "description": true,
          "dueDate": true,
          "index": true,
          "isComplete": true,
          "__typename": "ReminderItem"
        },
        "__typename": "Reminder"
      },
      "__typename": "ReminderList"
    },
    "resourceList": {
      "id": true,
      "created_at": true,
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "resources": {
        "id": true,
        "index": true,
        "link": true,
        "usedFor": true,
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "__typename": "Resource"
      },
      "__typename": "ResourceList"
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
    "__typename": "FocusMode"
  },
  "phones": {
    "id": true,
    "phoneNumber": true,
    "verified": true
  },
  "pushDevices": {
    "id": true,
    "expires": true,
    "name": true,
    "__typename": "PushDevice"
  },
  "wallets": {
    "id": true,
    "name": true,
    "publicAddress": true,
    "stakingAddress": true,
    "verified": true
  },
  "notifications": {
    "id": true,
    "created_at": true,
    "category": true,
    "isRead": true,
    "title": true,
    "description": true,
    "link": true,
    "imgLink": true
  },
  "translations": {
    "id": true,
    "language": true,
    "bio": true
  },
  "you": {
    "canDelete": true,
    "canReport": true,
    "canUpdate": true,
    "isBookmarked": true,
    "isViewed": true
  },
  "__typename": "User"
} as const;

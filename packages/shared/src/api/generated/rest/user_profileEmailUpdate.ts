export const user_profileEmailUpdate = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "handle": true,
  "isPrivate": true,
  "isPrivateApis": true,
  "isPrivateApisCreated": true,
  "isPrivateMemberships": true,
  "isPrivateOrganizationsCreated": true,
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
  "isPrivateBookmarks": true,
  "isPrivateVotes": true,
  "name": true,
  "theme": true,
  "emails": {
    "id": true,
    "emailAddress": true,
    "verified": true,
    "__typename": "Email"
  },
  "focusModes": {
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
        "labels": {
          "id": true,
          "color": true,
          "label": true,
          "__typename": "Label"
        },
        "schedule": {
          "__typename": "Schedule"
        },
        "id": true,
        "name": true,
        "description": true,
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
        }
      }
    },
    "schedule": {
      "__typename": "Schedule"
    },
    "id": true,
    "name": true,
    "description": true,
    "__typename": "FocusMode"
  },
  "pushDevices": {
    "id": true,
    "expires": true,
    "name": true,
    "__typename": "PushDevice"
  },
  "wallets": {
    "id": true,
    "handles": {
      "id": true,
      "handle": true
    },
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
  "notificationSettings": true,
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

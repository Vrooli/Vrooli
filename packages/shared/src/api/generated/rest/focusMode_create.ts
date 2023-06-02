export const focusMode_create = {
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
};

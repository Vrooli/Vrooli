export const focusMode_setActive = {
  "mode": {
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
        }
      },
      "focusMode": {
        "labels": {
          "id": true,
          "color": true,
          "label": true
        },
        "schedule": {},
        "id": true,
        "name": true,
        "description": true
      }
    },
    "labels": {
      "id": true,
      "color": true,
      "label": true
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
          "isComplete": true
        }
      }
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
    "schedule": {},
    "id": true,
    "name": true,
    "description": true
  },
  "stopCondition": true,
  "stopTime": true,
  "__typename": "FocusMode"
};

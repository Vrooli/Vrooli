export const reminder_create = {
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
  "reminderList": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "focusMode": {
      "labels": {
        "id": true,
        "color": true,
        "label": true,
        "__typename": "Label"
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
      "id": true,
      "name": true,
      "description": true,
      "you": {
        "canDelete": true,
        "canRead": true,
        "canUpdate": true
      },
      "__typename": "FocusMode"
    },
    "__typename": "ReminderList"
  },
  "__typename": "Reminder"
} as const;

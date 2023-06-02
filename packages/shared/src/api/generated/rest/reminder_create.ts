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
      "schedule": {
        "__typename": "Schedule"
      },
      "id": true,
      "name": true,
      "description": true,
      "__typename": "FocusMode"
    },
    "__typename": "ReminderList"
  },
  "__typename": "Reminder"
} as const;

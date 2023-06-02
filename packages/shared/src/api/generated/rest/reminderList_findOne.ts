export const reminderList_findOne = {
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
} as const;

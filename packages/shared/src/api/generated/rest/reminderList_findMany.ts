export const reminderList_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "updated_at": true,
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
          "isComplete": true
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ReminderList"
} as const;

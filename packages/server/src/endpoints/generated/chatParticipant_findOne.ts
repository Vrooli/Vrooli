export const chatParticipant_findOne = {
  "user": {
    "id": true,
    "isBot": true,
    "name": true,
    "handle": true,
    "__typename": "User"
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "__typename": "ChatParticipant"
} as const;

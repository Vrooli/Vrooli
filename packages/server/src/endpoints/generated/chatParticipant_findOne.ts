export const chatParticipant_findOne = {
  "user": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "bannerImage": true,
    "handle": true,
    "isBot": true,
    "name": true,
    "profileImage": true,
    "__typename": "User"
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "__typename": "ChatParticipant"
} as const;

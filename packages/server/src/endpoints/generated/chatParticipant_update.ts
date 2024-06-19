export const chatParticipant_update = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "user": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "bannerImage": true,
    "handle": true,
    "isBot": true,
    "isBotDepictingPerson": true,
    "name": true,
    "profileImage": true,
    "__typename": "User"
  },
  "__typename": "ChatParticipant"
} as const;

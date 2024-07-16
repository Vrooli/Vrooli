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
    "bookmarks": true,
    "reportsReceivedCount": true,
    "you": {
      "canDelete": true,
      "canReport": true,
      "canUpdate": true,
      "isBookmarked": true,
      "isViewed": true
    },
    "translations": {
      "id": true,
      "language": true,
      "bio": true
    },
    "__typename": "User"
  },
  "__typename": "ChatParticipant"
} as const;

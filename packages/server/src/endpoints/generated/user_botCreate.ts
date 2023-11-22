export const user_botCreate = {
  "botSettings": true,
  "translations": {
    "id": true,
    "language": true,
    "bio": true
  },
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
  "__typename": "User"
} as const;

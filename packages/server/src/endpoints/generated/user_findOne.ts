export const user_findOne = {
  "botSettings": true,
  "translations": {
    "id": true,
    "language": true,
    "bio": true
  },
  "id": true,
  "created_at": true,
  "handle": true,
  "isBot": true,
  "name": true,
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

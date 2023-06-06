export const user_findMany = {
  "edges": {
    "cursor": true,
    "node": {
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
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "User"
} as const;

export const organization_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "handle": true,
      "created_at": true,
      "updated_at": true,
      "isOpenToNewMembers": true,
      "isPrivate": true,
      "commentsCount": true,
      "membersCount": true,
      "reportsCount": true,
      "bookmarks": true,
      "tags": {
        "id": true,
        "created_at": true,
        "tag": true,
        "bookmarks": true,
        "translations": {
          "id": true,
          "language": true,
          "description": true
        },
        "you": {
          "isOwn": true,
          "isBookmarked": true
        }
      },
      "translations": {
        "id": true,
        "language": true,
        "bio": true,
        "name": true
      },
      "you": {
        "canAddMembers": true,
        "canDelete": true,
        "canBookmark": true,
        "canReport": true,
        "canUpdate": true,
        "canRead": true,
        "isBookmarked": true,
        "isViewed": true,
        "yourMembership": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isAdmin": true,
          "permissions": true
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Organization"
} as const;

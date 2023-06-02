export const member_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "organization": {
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
        "tags": {},
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
      },
      "user": {
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
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "isAdmin": true,
      "permissions": true,
      "roles": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "name": true,
        "permissions": true,
        "membersCount": true,
        "organization": {
          "id": true,
          "handle": true,
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
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Member"
};

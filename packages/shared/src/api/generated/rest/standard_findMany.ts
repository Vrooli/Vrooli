export const standard_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "versions": {
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "jsonVariable": true,
          "name": true
        },
        "id": true,
        "created_at": true,
        "updated_at": true,
        "isComplete": true,
        "isFile": true,
        "isLatest": true,
        "isPrivate": true,
        "default": true,
        "standardType": true,
        "props": true,
        "yup": true,
        "versionIndex": true,
        "versionLabel": true,
        "commentsCount": true,
        "directoryListingsCount": true,
        "forksCount": true,
        "reportsCount": true,
        "you": {
          "canComment": true,
          "canCopy": true,
          "canDelete": true,
          "canReport": true,
          "canUpdate": true,
          "canUse": true,
          "canRead": true
        }
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "isPrivate": true,
      "issuesCount": true,
      "labels": {},
      "owner": {
        "Organization": {
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
        "User": {
          "id": true,
          "isBot": true,
          "name": true,
          "handle": true
        }
      },
      "permissions": true,
      "questionsCount": true,
      "score": true,
      "bookmarks": true,
      "tags": {},
      "transfersCount": true,
      "views": true,
      "you": {
        "canDelete": true,
        "canBookmark": true,
        "canTransfer": true,
        "canUpdate": true,
        "canRead": true,
        "canReact": true,
        "isBookmarked": true,
        "isViewed": true,
        "reaction": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Standard"
} as const;

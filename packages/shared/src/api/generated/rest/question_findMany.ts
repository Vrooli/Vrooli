export const question_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "createdBy": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true
      },
      "hasAcceptedAnswer": true,
      "isPrivate": true,
      "score": true,
      "bookmarks": true,
      "answersCount": true,
      "commentsCount": true,
      "reportsCount": true,
      "forObject": {
        "Api": {
          "id": true,
          "isPrivate": true
        },
        "Note": {
          "id": true,
          "isPrivate": true
        },
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
        "Project": {
          "id": true,
          "isPrivate": true
        },
        "Routine": {
          "id": true,
          "isInternal": true,
          "isPrivate": true
        },
        "SmartContract": {
          "id": true,
          "isPrivate": true
        },
        "Standard": {
          "id": true,
          "isPrivate": true
        }
      },
      "tags": {},
      "you": {}
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Question"
};

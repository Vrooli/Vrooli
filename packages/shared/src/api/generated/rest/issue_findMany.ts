export const issue_findMany = {
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
      "closedAt": true,
      "referencedVersionId": true,
      "status": true,
      "to": {
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
      "commentsCount": true,
      "reportsCount": true,
      "score": true,
      "bookmarks": true,
      "views": true,
      "labels": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "color": true,
        "label": true,
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
        "you": {
          "canDelete": true,
          "canUpdate": true
        }
      },
      "you": {
        "canComment": true,
        "canDelete": true,
        "canBookmark": true,
        "canReport": true,
        "canUpdate": true,
        "canRead": true,
        "canReact": true,
        "isBookmarked": true,
        "reaction": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Issue"
} as const;

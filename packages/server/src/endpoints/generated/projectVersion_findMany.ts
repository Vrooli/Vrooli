export const projectVersion_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "root": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "isPrivate": true,
        "issuesCount": true,
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
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "directoriesCount": true,
      "isLatest": true,
      "isPrivate": true,
      "reportsCount": true,
      "runProjectsCount": true,
      "simplicity": true,
      "versionIndex": true,
      "versionLabel": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ProjectVersion"
} as const;

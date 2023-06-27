export const routine_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "versions": {
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "instructions": true,
          "name": true
        },
        "id": true,
        "created_at": true,
        "updated_at": true,
        "completedAt": true,
        "isAutomatable": true,
        "isComplete": true,
        "isDeleted": true,
        "isLatest": true,
        "isPrivate": true,
        "simplicity": true,
        "timesStarted": true,
        "timesCompleted": true,
        "smartContractCallData": true,
        "apiCallData": true,
        "versionIndex": true,
        "versionLabel": true,
        "commentsCount": true,
        "directoryListingsCount": true,
        "forksCount": true,
        "inputsCount": true,
        "nodesCount": true,
        "nodeLinksCount": true,
        "outputsCount": true,
        "reportsCount": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "isInternal": true,
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
        "canComment": true,
        "canDelete": true,
        "canBookmark": true,
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
  "__typename": "Routine"
} as const;

export const runRoutineInput_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "data": true,
      "input": {
        "id": true,
        "index": true,
        "isRequired": true,
        "name": true,
        "standardVersion": {
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
                  "bannerImage": true,
                  "handle": true,
                  "profileImage": true,
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
                  "created_at": true,
                  "updated_at": true,
                  "bannerImage": true,
                  "handle": true,
                  "isBot": true,
                  "isBotDepictingPerson": true,
                  "name": true,
                  "profileImage": true
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
                "bannerImage": true,
                "handle": true,
                "profileImage": true,
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
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "isBotDepictingPerson": true,
                "name": true,
                "profileImage": true
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
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "RunRoutineInput"
} as const;

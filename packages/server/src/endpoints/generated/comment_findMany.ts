export const comment_findMany = {
  "endCursor": true,
  "threads": {
    "childThreads": {
      "childThreads": {
        "comment": {
          "translations": {
            "id": true,
            "language": true,
            "text": true
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
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
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
            }
          },
          "score": true,
          "bookmarks": true,
          "reportsCount": true,
          "you": {
            "canDelete": true,
            "canBookmark": true,
            "canReply": true,
            "canReport": true,
            "canUpdate": true,
            "canReact": true,
            "isBookmarked": true,
            "reaction": true
          }
        },
        "endCursor": true,
        "totalInThread": true
      },
      "comment": {
        "translations": {
          "id": true,
          "language": true,
          "text": true
        },
        "id": true,
        "created_at": true,
        "updated_at": true,
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
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "name": true,
            "profileImage": true
          }
        },
        "score": true,
        "bookmarks": true,
        "reportsCount": true,
        "you": {
          "canDelete": true,
          "canBookmark": true,
          "canReply": true,
          "canReport": true,
          "canUpdate": true,
          "canReact": true,
          "isBookmarked": true,
          "reaction": true
        }
      },
      "endCursor": true,
      "totalInThread": true
    },
    "comment": {
      "translations": {
        "id": true,
        "language": true,
        "text": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
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
          "bannerImage": true,
          "handle": true,
          "isBot": true,
          "name": true,
          "profileImage": true
        }
      },
      "score": true,
      "bookmarks": true,
      "reportsCount": true,
      "you": {
        "canDelete": true,
        "canBookmark": true,
        "canReply": true,
        "canReport": true,
        "canUpdate": true,
        "canReact": true,
        "isBookmarked": true,
        "reaction": true
      }
    },
    "endCursor": true,
    "totalInThread": true
  },
  "totalThreads": true,
  "__typename": "Comment"
} as const;

export const chatInvite_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "chat": {
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
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
            }
          },
          "you": {
            "canDelete": true,
            "canUpdate": true
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
        "openToAnyoneWithInvite": true,
        "organization": {
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
        "restrictedToRoles": {
          "members": {
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
              "translations": {
                "id": true,
                "language": true,
                "description": true
              }
            }
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "name": true,
          "permissions": true,
          "membersCount": true,
          "organization": {
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
          "translations": {
            "id": true,
            "language": true,
            "description": true
          }
        },
        "participants": {
          "user": {
            "id": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "name": true,
            "profileImage": true
          },
          "id": true,
          "created_at": true,
          "updated_at": true
        },
        "participantsCount": true,
        "invitesCount": true,
        "you": {
          "canDelete": true,
          "canInvite": true,
          "canUpdate": true
        }
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "message": true,
      "status": true,
      "user": {
        "id": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true
      },
      "you": {
        "canDelete": true,
        "canUpdate": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ChatInvite"
} as const;

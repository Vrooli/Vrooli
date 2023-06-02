export const meeting_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "labels": {},
      "schedule": {},
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "link": true,
        "name": true
      },
      "id": true,
      "openToAnyoneWithInvite": true,
      "showOnOrganizationProfile": true,
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
        },
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
      },
      "attendeesCount": true,
      "invitesCount": true,
      "you": {
        "canDelete": true,
        "canInvite": true,
        "canUpdate": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Meeting"
};

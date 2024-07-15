export const chat_update = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "openToAnyoneWithInvite": true,
  "participants": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "user": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "bannerImage": true,
      "handle": true,
      "isBot": true,
      "isBotDepictingPerson": true,
      "name": true,
      "profileImage": true,
      "bookmarks": true,
      "reportsReceivedCount": true,
      "you": {
        "canDelete": true,
        "canReport": true,
        "canUpdate": true,
        "isBookmarked": true,
        "isViewed": true
      },
      "translations": {
        "id": true,
        "language": true,
        "bio": true
      },
      "__typename": "User"
    },
    "__typename": "ChatParticipant"
  },
  "restrictedToRoles": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "name": true,
    "permissions": true,
    "membersCount": true,
    "team": {
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
      },
      "__typename": "Team"
    },
    "translations": {
      "id": true,
      "language": true,
      "description": true
    },
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
        "team": {
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
          },
          "__typename": "Team"
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true
        },
        "__typename": "Role"
      },
      "you": {
        "canDelete": true,
        "canUpdate": true
      },
      "user": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "isBotDepictingPerson": true,
        "name": true,
        "profileImage": true,
        "bookmarks": true,
        "reportsReceivedCount": true,
        "you": {
          "canDelete": true,
          "canReport": true,
          "canUpdate": true,
          "isBookmarked": true,
          "isViewed": true
        },
        "translations": {
          "id": true,
          "language": true,
          "bio": true
        },
        "__typename": "User"
      },
      "__typename": "Member"
    },
    "__typename": "Role"
  },
  "team": {
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
    },
    "__typename": "Team"
  },
  "participantsCount": true,
  "invitesCount": true,
  "you": {
    "canDelete": true,
    "canInvite": true,
    "canUpdate": true
  },
  "invites": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "message": true,
    "status": true,
    "user": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "bannerImage": true,
      "handle": true,
      "isBot": true,
      "isBotDepictingPerson": true,
      "name": true,
      "profileImage": true,
      "__typename": "User"
    },
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
    "__typename": "ChatInvite"
  },
  "labels": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "color": true,
    "label": true,
    "owner": {
      "Team": {
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
        },
        "__typename": "Team"
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
        "profileImage": true,
        "__typename": "User"
      }
    },
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
    "apisCount": true,
    "codesCount": true,
    "focusModesCount": true,
    "issuesCount": true,
    "meetingsCount": true,
    "notesCount": true,
    "projectsCount": true,
    "routinesCount": true,
    "schedulesCount": true,
    "standardsCount": true,
    "__typename": "Label"
  },
  "translations": {
    "id": true,
    "language": true,
    "description": true,
    "name": true
  },
  "__typename": "Chat"
} as const;

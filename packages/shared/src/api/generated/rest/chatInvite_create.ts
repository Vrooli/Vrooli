export const chatInvite_create = {
  "chat": {
    "participants": {
      "id": true,
      "isBot": true,
      "name": true,
      "handle": true,
      "__typename": "ChatParticipant"
    },
    "messages": {
      "translations": {
        "id": true,
        "language": true,
        "text": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "user": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true,
        "__typename": "User"
      },
      "score": true,
      "reportsCount": true,
      "you": {
        "canDelete": true,
        "canReply": true,
        "canReport": true,
        "canUpdate": true,
        "canReact": true,
        "reaction": true
      },
      "__typename": "ChatMessage"
    },
    "labels": {
      "__typename": "Label"
    },
    "translations": {
      "id": true,
      "language": true,
      "description": true,
      "name": true
    },
    "id": true,
    "openToAnyoneWithInvite": true,
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
      },
      "__typename": "Organization"
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
            },
            "__typename": "Organization"
          },
          "translations": {
            "id": true,
            "language": true,
            "description": true
          },
          "__typename": "Role"
        },
        "__typename": "Member"
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
        },
        "__typename": "Organization"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true
      },
      "__typename": "Role"
    },
    "participantsCount": true,
    "invitesCount": true,
    "you": {
      "canDelete": true,
      "canInvite": true,
      "canUpdate": true
    },
    "__typename": "Chat"
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "message": true,
  "status": true,
  "you": {
    "canDelete": true,
    "canUpdate": true
  },
  "__typename": "ChatInvite"
} as const;

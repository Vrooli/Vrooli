export const chat_update = {
  "participants": {
    "user": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "bannerImage": true,
      "handle": true,
      "isBot": true,
      "name": true,
      "profileImage": true,
      "__typename": "User"
    },
    "id": true,
    "created_at": true,
    "updated_at": true,
    "__typename": "ChatParticipant"
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
  "messages": {
    "translations": {
      "id": true,
      "language": true,
      "text": true
    },
    "id": true,
    "created_at": true,
    "updated_at": true,
    "fork": {
      "id": true,
      "created_at": true,
      "__typename": "ChatMessage"
    },
    "user": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "bannerImage": true,
      "handle": true,
      "isBot": true,
      "name": true,
      "profileImage": true,
      "__typename": "User"
    },
    "score": true,
    "reactionSummaries": {
      "emoji": true,
      "count": true,
      "__typename": "ReactionSummary"
    },
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
    "apisCount": true,
    "focusModesCount": true,
    "issuesCount": true,
    "meetingsCount": true,
    "notesCount": true,
    "projectsCount": true,
    "routinesCount": true,
    "schedulesCount": true,
    "smartContractsCount": true,
    "standardsCount": true,
    "id": true,
    "created_at": true,
    "updated_at": true,
    "color": true,
    "label": true,
    "owner": {
      "User": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true,
        "__typename": "User"
      },
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
        },
        "__typename": "Organization"
      }
    },
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
    "__typename": "Label"
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
          "__typename": "Organization"
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
} as const;

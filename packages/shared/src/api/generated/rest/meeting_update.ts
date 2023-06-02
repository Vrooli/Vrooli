export const meeting_update = {
  "attendees": {
    "id": true,
    "isBot": true,
    "name": true,
    "handle": true,
    "__typename": "User"
  },
  "invites": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "message": true,
    "status": true,
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
    "__typename": "MeetingInvite"
  },
  "labels": {
    "__typename": "Label"
  },
  "schedule": {
    "__typename": "Schedule"
  },
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
  "attendeesCount": true,
  "invitesCount": true,
  "you": {
    "canDelete": true,
    "canInvite": true,
    "canUpdate": true
  },
  "__typename": "Meeting"
};

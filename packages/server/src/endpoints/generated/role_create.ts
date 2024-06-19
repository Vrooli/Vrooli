export const role_create = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "name": true,
  "permissions": true,
  "membersCount": true,
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
    "__typename": "Member"
  },
  "__typename": "Role"
} as const;

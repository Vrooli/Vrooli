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
} as const;

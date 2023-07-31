export const role_update = {
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
    "__typename": "Member"
  },
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
  "__typename": "Role"
} as const;

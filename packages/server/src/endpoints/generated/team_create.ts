export const team_create = {
  "id": true,
  "bannerImage": true,
  "handle": true,
  "created_at": true,
  "updated_at": true,
  "isOpenToNewMembers": true,
  "isPrivate": true,
  "commentsCount": true,
  "membersCount": true,
  "profileImage": true,
  "reportsCount": true,
  "bookmarks": true,
  "tags": {
    "id": true,
    "created_at": true,
    "tag": true,
    "bookmarks": true,
    "translations": {
      "id": true,
      "language": true,
      "description": true
    },
    "you": {
      "isOwn": true,
      "isBookmarked": true
    },
    "__typename": "Tag"
  },
  "translations": {
    "id": true,
    "language": true,
    "bio": true,
    "name": true
  },
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
  "roles": {
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
  },
  "__typename": "Team"
} as const;

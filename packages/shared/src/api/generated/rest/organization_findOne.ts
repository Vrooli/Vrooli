export const organization_findOne = {
  "roles": {
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
    "translations": {
      "id": true,
      "language": true,
      "description": true
    },
    "__typename": "Role"
  },
  "id": true,
  "handle": true,
  "created_at": true,
  "updated_at": true,
  "isOpenToNewMembers": true,
  "isPrivate": true,
  "commentsCount": true,
  "membersCount": true,
  "reportsCount": true,
  "bookmarks": true,
  "tags": {
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
  "__typename": "Organization"
};

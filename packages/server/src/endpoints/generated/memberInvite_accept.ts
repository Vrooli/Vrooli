export const memberInvite_accept = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "message": true,
  "status": true,
  "willBeAdmin": true,
  "willHavePermissions": true,
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
  "__typename": "MemberInvite"
} as const;

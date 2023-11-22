export const label_findOne = {
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
      "isBotDepictingPerson": true,
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
} as const;

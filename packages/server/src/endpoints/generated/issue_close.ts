export const issue_close = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "closedAt": true,
  "referencedVersionId": true,
  "status": true,
  "to": {
    "Api": {
      "id": true,
      "isPrivate": true,
      "__typename": "Api"
    },
    "Code": {
      "id": true,
      "isPrivate": true,
      "__typename": "Code"
    },
    "Note": {
      "id": true,
      "isPrivate": true,
      "__typename": "Note"
    },
    "Project": {
      "id": true,
      "isPrivate": true,
      "__typename": "Project"
    },
    "Routine": {
      "id": true,
      "isInternal": true,
      "isPrivate": true,
      "__typename": "Routine"
    },
    "Standard": {
      "id": true,
      "isPrivate": true,
      "__typename": "Standard"
    },
    "Team": {
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
    }
  },
  "commentsCount": true,
  "reportsCount": true,
  "score": true,
  "bookmarks": true,
  "views": true,
  "labels": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "color": true,
    "label": true,
    "owner": {
      "Team": {
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
      }
    },
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
    "__typename": "Label"
  },
  "you": {
    "canComment": true,
    "canDelete": true,
    "canBookmark": true,
    "canReport": true,
    "canUpdate": true,
    "canRead": true,
    "canReact": true,
    "isBookmarked": true,
    "reaction": true
  },
  "closedBy": {
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
  "createdBy": {
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
  "translations": {
    "id": true,
    "language": true,
    "description": true,
    "name": true
  },
  "__typename": "Issue"
} as const;

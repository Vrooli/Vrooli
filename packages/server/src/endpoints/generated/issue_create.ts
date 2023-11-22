export const issue_create = {
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
    "SmartContract": {
      "id": true,
      "isPrivate": true,
      "__typename": "SmartContract"
    },
    "Standard": {
      "id": true,
      "isPrivate": true,
      "__typename": "Standard"
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
  "__typename": "Issue"
} as const;

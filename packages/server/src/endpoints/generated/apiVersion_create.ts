export const apiVersion_create = {
  "pullRequest": {
    "translations": {
      "id": true,
      "language": true,
      "text": true
    },
    "id": true,
    "created_at": true,
    "updated_at": true,
    "mergedOrRejectedAt": true,
    "commentsCount": true,
    "status": true,
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
    "you": {
      "canComment": true,
      "canDelete": true,
      "canReport": true,
      "canUpdate": true
    },
    "__typename": "PullRequest"
  },
  "root": {
    "versionsCount": true,
    "parent": {
      "id": true,
      "isLatest": true,
      "isPrivate": true,
      "versionIndex": true,
      "versionLabel": true,
      "root": {
        "id": true,
        "isPrivate": true,
        "__typename": "Api"
      },
      "translations": {
        "id": true,
        "language": true,
        "details": true,
        "name": true,
        "summary": true
      },
      "__typename": "ApiVersion"
    },
    "stats": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "calls": true,
      "routineVersions": true,
      "__typename": "StatsApi"
    },
    "id": true,
    "created_at": true,
    "updated_at": true,
    "isPrivate": true,
    "issuesCount": true,
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
    "permissions": true,
    "questionsCount": true,
    "score": true,
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
    "transfersCount": true,
    "views": true,
    "you": {
      "canDelete": true,
      "canBookmark": true,
      "canTransfer": true,
      "canUpdate": true,
      "canRead": true,
      "canReact": true,
      "isBookmarked": true,
      "isViewed": true,
      "reaction": true
    },
    "__typename": "Api"
  },
  "translations": {
    "id": true,
    "language": true,
    "details": true,
    "name": true,
    "summary": true
  },
  "schemaText": true,
  "versionNotes": true,
  "id": true,
  "created_at": true,
  "updated_at": true,
  "callLink": true,
  "commentsCount": true,
  "documentationLink": true,
  "forksCount": true,
  "isLatest": true,
  "isPrivate": true,
  "reportsCount": true,
  "versionIndex": true,
  "versionLabel": true,
  "you": {
    "canComment": true,
    "canCopy": true,
    "canDelete": true,
    "canReport": true,
    "canUpdate": true,
    "canUse": true,
    "canRead": true
  },
  "__typename": "ApiVersion"
} as const;

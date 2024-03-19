export const smartContractVersion_update = {
  "versionNotes": true,
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
  "resourceList": {
    "id": true,
    "created_at": true,
    "translations": {
      "id": true,
      "language": true,
      "description": true,
      "name": true
    },
    "resources": {
      "id": true,
      "index": true,
      "link": true,
      "usedFor": true,
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      }
    }
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
        "__typename": "SmartContract"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "jsonVariable": true,
        "name": true
      },
      "__typename": "SmartContractVersion"
    },
    "stats": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "calls": true,
      "routineVersions": true
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
    "__typename": "SmartContract"
  },
  "translations": {
    "id": true,
    "language": true,
    "description": true,
    "jsonVariable": true,
    "name": true
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "isComplete": true,
  "isDeleted": true,
  "isLatest": true,
  "isPrivate": true,
  "default": true,
  "contractType": true,
  "content": true,
  "versionIndex": true,
  "versionLabel": true,
  "commentsCount": true,
  "directoryListingsCount": true,
  "forksCount": true,
  "reportsCount": true,
  "you": {
    "canComment": true,
    "canCopy": true,
    "canDelete": true,
    "canReport": true,
    "canUpdate": true,
    "canUse": true,
    "canRead": true
  },
  "__typename": "SmartContractVersion"
} as const;

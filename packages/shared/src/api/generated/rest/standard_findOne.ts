export const standard_findOne = {
  "parent": {
    "id": true,
    "isLatest": true,
    "isPrivate": true,
    "versionIndex": true,
    "versionLabel": true,
    "root": {
      "id": true,
      "isPrivate": true
    },
    "translations": {
      "id": true,
      "language": true,
      "description": true,
      "jsonVariable": true,
      "name": true
    },
    "__typename": "Project"
  },
  "versions": {
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
        "isBot": true,
        "name": true,
        "handle": true,
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
    "isFile": true,
    "isLatest": true,
    "isPrivate": true,
    "default": true,
    "standardType": true,
    "props": true,
    "yup": true,
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
    "__typename": "StandardVersion"
  },
  "stats": {
    "id": true,
    "periodStart": true,
    "periodEnd": true,
    "periodType": true,
    "linksToInputs": true,
    "linksToOutputs": true
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "isPrivate": true,
  "issuesCount": true,
  "labels": {
    "__typename": "Label"
  },
  "owner": {
    "User": {
      "id": true,
      "isBot": true,
      "name": true,
      "handle": true,
      "__typename": "User"
    },
    "Organization": {
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
    }
  },
  "permissions": true,
  "questionsCount": true,
  "score": true,
  "bookmarks": true,
  "tags": {
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
  "__typename": "Standard"
} as const;

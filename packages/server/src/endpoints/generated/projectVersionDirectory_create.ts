export const projectVersionDirectory_create = {
  "children": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "childOrder": true,
    "isRoot": true,
    "projectVersion": {
      "id": true,
      "complexity": true,
      "isLatest": true,
      "isPrivate": true,
      "versionIndex": true,
      "versionLabel": true,
      "root": {
        "id": true,
        "isPrivate": true,
        "__typename": "Project"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "__typename": "ProjectVersion"
    },
    "__typename": "ProjectVersionDirectory"
  },
  "childApiVersions": {
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
  },
  "childNoteVersions": {
    "id": true,
    "created_at": true,
    "updated_at": true,
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
    "__typename": "NoteVersion"
  },
  "childOrganizations": {
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
    "__typename": "Organization"
  },
  "childProjectVersions": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "directoriesCount": true,
    "isLatest": true,
    "isPrivate": true,
    "reportsCount": true,
    "runProjectsCount": true,
    "simplicity": true,
    "versionIndex": true,
    "versionLabel": true,
    "__typename": "ProjectVersion"
  },
  "childRoutineVersions": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "completedAt": true,
    "isAutomatable": true,
    "isComplete": true,
    "isDeleted": true,
    "isLatest": true,
    "isPrivate": true,
    "simplicity": true,
    "timesStarted": true,
    "timesCompleted": true,
    "smartContractCallData": true,
    "apiCallData": true,
    "versionIndex": true,
    "versionLabel": true,
    "commentsCount": true,
    "directoryListingsCount": true,
    "forksCount": true,
    "inputsCount": true,
    "nodesCount": true,
    "nodeLinksCount": true,
    "outputsCount": true,
    "reportsCount": true,
    "__typename": "RoutineVersion"
  },
  "childSmartContractVersions": {
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
  },
  "childStandardVersions": {
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
  "parentDirectory": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "childOrder": true,
    "isRoot": true,
    "projectVersion": {
      "id": true,
      "complexity": true,
      "isLatest": true,
      "isPrivate": true,
      "versionIndex": true,
      "versionLabel": true,
      "root": {
        "id": true,
        "isPrivate": true,
        "__typename": "Project"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "__typename": "ProjectVersion"
    },
    "__typename": "ProjectVersionDirectory"
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
  "childOrder": true,
  "isRoot": true,
  "projectVersion": {
    "id": true,
    "complexity": true,
    "isLatest": true,
    "isPrivate": true,
    "versionIndex": true,
    "versionLabel": true,
    "root": {
      "id": true,
      "isPrivate": true,
      "__typename": "Project"
    },
    "translations": {
      "id": true,
      "language": true,
      "description": true,
      "name": true
    },
    "__typename": "ProjectVersion"
  },
  "__typename": "ProjectVersionDirectory"
} as const;

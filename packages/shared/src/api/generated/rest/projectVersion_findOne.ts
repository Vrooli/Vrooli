export const projectVersion_findOne = {
  "directories": {
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
    "childNoteVersions": {
      "id": true,
      "isLatest": true,
      "isPrivate": true,
      "versionIndex": true,
      "versionLabel": true,
      "root": {
        "id": true,
        "isPrivate": true,
        "__typename": "Note"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true,
        "text": true
      },
      "__typename": "NoteVersion"
    },
    "childOrganizations": {
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
    "childProjectVersions": {
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
    "childRoutineVersions": {
      "id": true,
      "complexity": true,
      "isAutomatable": true,
      "isComplete": true,
      "isDeleted": true,
      "isLatest": true,
      "isPrivate": true,
      "root": {
        "id": true,
        "isInternal": true,
        "isPrivate": true,
        "__typename": "Routine"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "instructions": true,
        "name": true
      },
      "versionIndex": true,
      "versionLabel": true,
      "__typename": "RoutineVersion"
    },
    "childSmartContractVersions": {
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
    "childStandardVersions": {
      "id": true,
      "isLatest": true,
      "isPrivate": true,
      "versionIndex": true,
      "versionLabel": true,
      "root": {
        "id": true,
        "isPrivate": true,
        "__typename": "Standard"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "jsonVariable": true,
        "name": true
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
  },
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
  "root": {
    "parent": {
      "id": true,
      "complexity": true,
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
        "name": true
      },
      "__typename": "Project"
    },
    "stats": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "directories": true,
      "apis": true,
      "notes": true,
      "organizations": true,
      "projects": true,
      "routines": true,
      "smartContracts": true,
      "standards": true,
      "runsStarted": true,
      "runsCompleted": true,
      "runCompletionTimeAverage": true,
      "runContextSwitchesAverage": true
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
    "__typename": "Project"
  },
  "translations": {
    "id": true,
    "language": true,
    "description": true,
    "name": true
  },
  "versionNotes": true,
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
  "you": {
    "canComment": true,
    "canCopy": true,
    "canDelete": true,
    "canReport": true,
    "canUpdate": true,
    "canUse": true,
    "canRead": true
  },
  "__typename": "ProjectVersion"
};

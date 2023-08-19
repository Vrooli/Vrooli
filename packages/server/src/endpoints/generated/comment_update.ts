export const comment_update = {
  "commentedOn": {
    "ApiVersion": {
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
    "Issue": {
      "id": true,
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "__typename": "Issue"
    },
    "NoteVersion": {
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
        "pages": {
          "id": true,
          "pageIndex": true,
          "text": true
        }
      },
      "__typename": "NoteVersion"
    },
    "Post": {
      "id": true,
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "__typename": "Post"
    },
    "ProjectVersion": {
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
    "PullRequest": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "mergedOrRejectedAt": true,
      "status": true,
      "__typename": "PullRequest"
    },
    "Question": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "createdBy": {
        "id": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true,
        "__typename": "User"
      },
      "hasAcceptedAnswer": true,
      "isPrivate": true,
      "score": true,
      "bookmarks": true,
      "answersCount": true,
      "commentsCount": true,
      "reportsCount": true,
      "forObject": {
        "Api": {
          "id": true,
          "isPrivate": true,
          "__typename": "Api"
        },
        "Note": {
          "id": true,
          "isPrivate": true,
          "__typename": "Note"
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
      "you": {
        "reaction": true
      },
      "__typename": "Question"
    },
    "QuestionAnswer": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "createdBy": {
        "id": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true,
        "__typename": "User"
      },
      "score": true,
      "bookmarks": true,
      "isAccepted": true,
      "commentsCount": true,
      "__typename": "QuestionAnswer"
    },
    "RoutineVersion": {
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
    "SmartContractVersion": {
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
    "StandardVersion": {
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
    }
  },
  "translations": {
    "id": true,
    "language": true,
    "text": true
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "owner": {
    "User": {
      "id": true,
      "bannerImage": true,
      "handle": true,
      "isBot": true,
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
  "score": true,
  "bookmarks": true,
  "reportsCount": true,
  "you": {
    "canDelete": true,
    "canBookmark": true,
    "canReply": true,
    "canReport": true,
    "canUpdate": true,
    "canReact": true,
    "isBookmarked": true,
    "reaction": true
  },
  "__typename": "Comment"
} as const;

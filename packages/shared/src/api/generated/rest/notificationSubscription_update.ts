export const notificationSubscription_update = {
  "id": true,
  "created_at": true,
  "silent": true,
  "object": {
    "Api": {
      "versions": {
        "translations": {
          "id": true,
          "language": true,
          "details": true,
          "name": true,
          "summary": true
        },
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
      "__typename": "Api"
    },
    "Comment": {
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
    },
    "Issue": {
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
    },
    "Meeting": {
      "labels": {
        "__typename": "Label"
      },
      "schedule": {
        "__typename": "Schedule"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "link": true,
        "name": true
      },
      "id": true,
      "openToAnyoneWithInvite": true,
      "showOnOrganizationProfile": true,
      "organization": {
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
      "restrictedToRoles": {
        "members": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isAdmin": true,
          "permissions": true,
          "roles": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "name": true,
            "permissions": true,
            "membersCount": true,
            "organization": {
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
            "translations": {
              "id": true,
              "language": true,
              "description": true
            },
            "__typename": "Role"
          },
          "__typename": "Member"
        },
        "id": true,
        "created_at": true,
        "updated_at": true,
        "name": true,
        "permissions": true,
        "membersCount": true,
        "organization": {
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
        "translations": {
          "id": true,
          "language": true,
          "description": true
        },
        "__typename": "Role"
      },
      "attendeesCount": true,
      "invitesCount": true,
      "you": {
        "canDelete": true,
        "canInvite": true,
        "canUpdate": true
      },
      "__typename": "Meeting"
    },
    "Note": {
      "versions": {
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true,
          "text": true
        },
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
      "__typename": "Note"
    },
    "Organization": {
      "id": true,
      "handle": true,
      "created_at": true,
      "updated_at": true,
      "isOpenToNewMembers": true,
      "isPrivate": true,
      "commentsCount": true,
      "membersCount": true,
      "reportsCount": true,
      "bookmarks": true,
      "tags": {
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
    "Project": {
      "versions": {
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
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
    "PullRequest": {
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
    "Question": {
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "createdBy": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true,
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
        "__typename": "Tag"
      },
      "you": {},
      "__typename": "Question"
    },
    "Quiz": {
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "createdBy": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true,
        "__typename": "User"
      },
      "score": true,
      "bookmarks": true,
      "attemptsCount": true,
      "quizQuestionsCount": true,
      "project": {
        "id": true,
        "isPrivate": true,
        "__typename": "Project"
      },
      "routine": {
        "id": true,
        "isInternal": true,
        "isPrivate": true,
        "__typename": "Routine"
      },
      "you": {
        "canDelete": true,
        "canBookmark": true,
        "canUpdate": true,
        "canRead": true,
        "canReact": true,
        "hasCompleted": true,
        "isBookmarked": true,
        "reaction": true
      },
      "__typename": "Quiz"
    },
    "Report": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "details": true,
      "language": true,
      "reason": true,
      "responsesCount": true,
      "you": {
        "canDelete": true,
        "canRespond": true,
        "canUpdate": true
      },
      "__typename": "Report"
    },
    "Routine": {
      "versions": {
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "instructions": true,
          "name": true
        },
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
      "id": true,
      "created_at": true,
      "updated_at": true,
      "isInternal": true,
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
        "canComment": true,
        "canDelete": true,
        "canBookmark": true,
        "canUpdate": true,
        "canRead": true,
        "canReact": true,
        "isBookmarked": true,
        "isViewed": true,
        "reaction": true
      },
      "__typename": "Routine"
    },
    "SmartContract": {
      "versions": {
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
        "__typename": "NoteVersion"
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
      "__typename": "SmartContract"
    },
    "Standard": {
      "versions": {
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
    }
  },
  "__typename": "NotificationSubscription"
} as const;

export const bookmarkList_update = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "label": true,
  "bookmarksCount": true,
  "bookmarks": {
    "id": true,
    "to": {
      "Api": {
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
        "versions": {
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
          "schemaLanguage": true,
          "translations": {
            "id": true,
            "language": true,
            "details": true,
            "name": true,
            "summary": true
          },
          "__typename": "ApiVersion"
        },
        "__typename": "Api"
      },
      "Code": {
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
        "versions": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isComplete": true,
          "isDeleted": true,
          "isLatest": true,
          "isPrivate": true,
          "codeLanguage": true,
          "codeType": true,
          "default": true,
          "versionIndex": true,
          "versionLabel": true,
          "calledByRoutineVersionsCount": true,
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
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "jsonVariable": true,
            "name": true
          },
          "__typename": "CodeVersion"
        },
        "__typename": "Code"
      },
      "Comment": {
        "id": true,
        "created_at": true,
        "updated_at": true,
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
        "id": true,
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "__typename": "Issue"
      },
      "Note": {
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
        "versions": {
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
        "__typename": "Note"
      },
      "Post": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "commentsCount": true,
        "repostsCount": true,
        "score": true,
        "bookmarks": true,
        "views": true,
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
            },
            "__typename": "Resource"
          },
          "__typename": "ResourceList"
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "__typename": "Post"
      },
      "Project": {
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
        "versions": {
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
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
          },
          "__typename": "ProjectVersion"
        },
        "__typename": "Project"
      },
      "Question": {
        "id": true,
        "created_at": true,
        "updated_at": true,
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
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "__typename": "Question"
      },
      "QuestionAnswer": {
        "id": true,
        "created_at": true,
        "updated_at": true,
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
        "score": true,
        "bookmarks": true,
        "isAccepted": true,
        "commentsCount": true,
        "translations": {
          "id": true,
          "language": true,
          "text": true
        },
        "__typename": "QuestionAnswer"
      },
      "Quiz": {
        "id": true,
        "created_at": true,
        "updated_at": true,
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
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "__typename": "Quiz"
      },
      "Routine": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "isInternal": true,
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
        "versions": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "completedAt": true,
          "isAutomatable": true,
          "isComplete": true,
          "isDeleted": true,
          "isLatest": true,
          "isPrivate": true,
          "routineType": true,
          "simplicity": true,
          "timesStarted": true,
          "timesCompleted": true,
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
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "instructions": true,
            "name": true
          },
          "__typename": "RoutineVersion"
        },
        "__typename": "Routine"
      },
      "Standard": {
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
        "versions": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "codeLanguage": true,
          "default": true,
          "isComplete": true,
          "isFile": true,
          "isLatest": true,
          "isPrivate": true,
          "props": true,
          "variant": true,
          "versionIndex": true,
          "versionLabel": true,
          "yup": true,
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
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "jsonVariable": true,
            "name": true
          },
          "__typename": "StandardVersion"
        },
        "__typename": "Standard"
      },
      "Tag": {
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
      "Team": {
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
        "bookmarks": true,
        "reportsReceivedCount": true,
        "you": {
          "canDelete": true,
          "canReport": true,
          "canUpdate": true,
          "isBookmarked": true,
          "isViewed": true
        },
        "translations": {
          "id": true,
          "language": true,
          "bio": true
        },
        "__typename": "User"
      }
    },
    "__typename": "Bookmark"
  },
  "__typename": "BookmarkList"
} as const;

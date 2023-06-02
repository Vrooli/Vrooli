export const notificationSubscription_findMany = {
  "edges": {
    "cursor": true,
    "node": {
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
            }
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isPrivate": true,
          "issuesCount": true,
          "labels": {},
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "isBot": true,
              "name": true,
              "handle": true
            }
          },
          "permissions": true,
          "questionsCount": true,
          "score": true,
          "bookmarks": true,
          "tags": {},
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
          }
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
              }
            },
            "User": {
              "id": true,
              "isBot": true,
              "name": true,
              "handle": true
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
          }
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
              "isPrivate": true
            },
            "Note": {
              "id": true,
              "isPrivate": true
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
              }
            },
            "Project": {
              "id": true,
              "isPrivate": true
            },
            "Routine": {
              "id": true,
              "isInternal": true,
              "isPrivate": true
            },
            "SmartContract": {
              "id": true,
              "isPrivate": true
            },
            "Standard": {
              "id": true,
              "isPrivate": true
            }
          },
          "commentsCount": true,
          "reportsCount": true,
          "score": true,
          "bookmarks": true,
          "views": true,
          "labels": {},
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
          }
        },
        "Meeting": {
          "labels": {},
          "schedule": {},
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
            }
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
                  }
                },
                "translations": {
                  "id": true,
                  "language": true,
                  "description": true
                }
              }
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
              }
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true
            }
          },
          "attendeesCount": true,
          "invitesCount": true,
          "you": {
            "canDelete": true,
            "canInvite": true,
            "canUpdate": true
          }
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
            }
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isPrivate": true,
          "issuesCount": true,
          "labels": {},
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "isBot": true,
              "name": true,
              "handle": true
            }
          },
          "permissions": true,
          "questionsCount": true,
          "score": true,
          "bookmarks": true,
          "tags": {},
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
          }
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
          "tags": {},
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
          }
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
            "versionLabel": true
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isPrivate": true,
          "issuesCount": true,
          "labels": {},
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "isBot": true,
              "name": true,
              "handle": true
            }
          },
          "permissions": true,
          "questionsCount": true,
          "score": true,
          "bookmarks": true,
          "tags": {},
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
          }
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
            "handle": true
          },
          "you": {
            "canComment": true,
            "canDelete": true,
            "canReport": true,
            "canUpdate": true
          }
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
            "handle": true
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
              "isPrivate": true
            },
            "Note": {
              "id": true,
              "isPrivate": true
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
              }
            },
            "Project": {
              "id": true,
              "isPrivate": true
            },
            "Routine": {
              "id": true,
              "isInternal": true,
              "isPrivate": true
            },
            "SmartContract": {
              "id": true,
              "isPrivate": true
            },
            "Standard": {
              "id": true,
              "isPrivate": true
            }
          },
          "tags": {},
          "you": {}
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
            "handle": true
          },
          "score": true,
          "bookmarks": true,
          "attemptsCount": true,
          "quizQuestionsCount": true,
          "project": {
            "id": true,
            "isPrivate": true
          },
          "routine": {
            "id": true,
            "isInternal": true,
            "isPrivate": true
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
          }
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
          }
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
            "reportsCount": true
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isInternal": true,
          "isPrivate": true,
          "issuesCount": true,
          "labels": {},
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "isBot": true,
              "name": true,
              "handle": true
            }
          },
          "permissions": true,
          "questionsCount": true,
          "score": true,
          "bookmarks": true,
          "tags": {},
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
          }
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
            }
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isPrivate": true,
          "issuesCount": true,
          "labels": {},
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "isBot": true,
              "name": true,
              "handle": true
            }
          },
          "permissions": true,
          "questionsCount": true,
          "score": true,
          "bookmarks": true,
          "tags": {},
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
          }
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
            }
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isPrivate": true,
          "issuesCount": true,
          "labels": {},
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "isBot": true,
              "name": true,
              "handle": true
            }
          },
          "permissions": true,
          "questionsCount": true,
          "score": true,
          "bookmarks": true,
          "tags": {},
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
          }
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "NotificationSubscription"
};

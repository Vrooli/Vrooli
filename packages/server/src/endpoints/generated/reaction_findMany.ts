export const reaction_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "to": {
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
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
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
                }
              },
              "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "name": true,
                "profileImage": true
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
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
            }
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
          }
        },
        "ChatMessage": {
          "translations": {
            "id": true,
            "language": true,
            "text": true
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "sequence": true,
          "versionIndex": true,
          "parent": {
            "id": true,
            "created_at": true
          },
          "user": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "name": true,
            "profileImage": true
          },
          "score": true,
          "reactionSummaries": {
            "emoji": true,
            "count": true
          },
          "reportsCount": true,
          "you": {
            "canDelete": true,
            "canReply": true,
            "canReport": true,
            "canUpdate": true,
            "canReact": true,
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
              }
            },
            "User": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
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
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
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
                }
              },
              "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "name": true,
                "profileImage": true
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
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
          }
        },
        "Note": {
          "versions": {
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
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
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
                }
              },
              "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "name": true,
                "profileImage": true
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
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
            }
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
          }
        },
        "Post": {
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
            "name": true
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "commentsCount": true,
          "repostsCount": true,
          "score": true,
          "bookmarks": true,
          "views": true
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
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
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
                }
              },
              "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "name": true,
                "profileImage": true
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
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
            }
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
            "created_at": true,
            "updated_at": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "name": true,
            "profileImage": true
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
            }
          },
          "you": {
            "reaction": true
          }
        },
        "QuestionAnswer": {
          "translations": {
            "id": true,
            "language": true,
            "text": true
          },
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
            "name": true,
            "profileImage": true
          },
          "score": true,
          "bookmarks": true,
          "isAccepted": true,
          "commentsCount": true
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
            "created_at": true,
            "updated_at": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "name": true,
            "profileImage": true
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
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
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
                }
              },
              "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "name": true,
                "profileImage": true
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
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
            }
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
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
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
                }
              },
              "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "name": true,
                "profileImage": true
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
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
            }
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
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
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
                }
              },
              "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "name": true,
                "profileImage": true
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
          "owner": {
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
              }
            },
            "User": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "bannerImage": true,
              "handle": true,
              "isBot": true,
              "name": true,
              "profileImage": true
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
            }
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
          }
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Reaction"
} as const;

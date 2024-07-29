export const project_findOne = {
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
  "versionsCount": true,
  "parent": {
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
    "directories": {
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
        "root": {
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
        "__typename": "ApiVersion"
      },
      "childCodeVersions": {
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
        "root": {
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
          "__typename": "Code"
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
        "root": {
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
        "root": {
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
        "root": {
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
          "__typename": "Routine"
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "instructions": true,
          "name": true
        },
        "__typename": "RoutineVersion"
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
        "root": {
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
      "childTeams": {
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
      "__typename": "ProjectVersionDirectory"
    },
    "pullRequest": {
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
      "translations": {
        "id": true,
        "language": true,
        "text": true
      },
      "__typename": "PullRequest"
    },
    "translations": {
      "id": true,
      "language": true,
      "description": true,
      "name": true
    },
    "versionNotes": true,
    "__typename": "ProjectVersion"
  },
  "__typename": "Project"
} as const;

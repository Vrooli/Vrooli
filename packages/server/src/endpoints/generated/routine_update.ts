export const routine_update = {
  "parent": {
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
      "isPrivate": true
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
    "__typename": "Routine"
  },
  "versions": {
    "versionNotes": true,
    "apiVersion": {
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
            "details": true,
            "name": true,
            "summary": true
          },
          "__typename": "Api"
        },
        "stats": {
          "id": true,
          "periodStart": true,
          "periodEnd": true,
          "periodType": true,
          "calls": true,
          "routineVersions": true,
          "__typename": "StatsApi"
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
          "you": {
            "canDelete": true,
            "canUpdate": true
          },
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
      "versionNotes": true,
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
    "inputs": {
      "id": true,
      "index": true,
      "isRequired": true,
      "name": true,
      "standardVersion": {
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
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "helpText": true
      },
      "__typename": "RoutineVersionInput"
    },
    "nodes": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "columnIndex": true,
      "nodeType": true,
      "rowIndex": true,
      "end": {
        "id": true,
        "wasSuccessful": true,
        "suggestedNextRoutineVersions": {
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
        "__typename": "NodeEnd"
      },
      "routineList": {
        "id": true,
        "isOrdered": true,
        "isOptional": true,
        "items": {
          "id": true,
          "index": true,
          "isOptional": true,
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
          },
          "__typename": "NodeRoutineListItem"
        },
        "__typename": "NodeRoutineList"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "__typename": "Node"
    },
    "nodeLinks": {
      "id": true,
      "from": {
        "id": true
      },
      "operation": true,
      "to": {
        "id": true
      },
      "whens": {
        "id": true,
        "condition": true,
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        }
      }
    },
    "outputs": {
      "id": true,
      "index": true,
      "name": true,
      "standardVersion": {
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
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "helpText": true
      },
      "__typename": "RoutineVersionOutput"
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
    "smartContractVersion": {
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
      "root": {
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
          "__typename": "SmartContract"
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
          "you": {
            "canDelete": true,
            "canUpdate": true
          },
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
    },
    "suggestedNextByRoutineVersion": {
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
    "you": {
      "runs": {
        "inputs": {
          "id": true,
          "data": true,
          "input": {
            "id": true,
            "index": true,
            "isRequired": true,
            "name": true,
            "standardVersion": {
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
            }
          }
        },
        "steps": {
          "id": true,
          "order": true,
          "contextSwitches": true,
          "startedAt": true,
          "timeElapsed": true,
          "completedAt": true,
          "name": true,
          "status": true,
          "step": true,
          "subroutine": {
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
              "isPrivate": true
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "instructions": true,
              "name": true
            },
            "versionIndex": true,
            "versionLabel": true
          }
        },
        "id": true,
        "isPrivate": true,
        "completedComplexity": true,
        "contextSwitches": true,
        "startedAt": true,
        "timeElapsed": true,
        "completedAt": true,
        "name": true,
        "status": true,
        "stepsCount": true,
        "inputsCount": true,
        "wasRunAutomatically": true,
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
        "schedule": {
          "labels": {
            "apisCount": true,
            "focusModesCount": true,
            "issuesCount": true,
            "meetingsCount": true,
            "notesCount": true,
            "projectsCount": true,
            "routinesCount": true,
            "schedulesCount": true,
            "smartContractsCount": true,
            "standardsCount": true,
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
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
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "startTime": true,
          "endTime": true,
          "timezone": true,
          "exceptions": {
            "id": true,
            "originalStartTime": true,
            "newStartTime": true,
            "newEndTime": true
          },
          "recurrences": {
            "id": true,
            "recurrenceType": true,
            "interval": true,
            "dayOfWeek": true,
            "dayOfMonth": true,
            "month": true,
            "endDate": true
          }
        },
        "user": {
          "id": true,
          "isBot": true,
          "name": true,
          "handle": true
        },
        "you": {
          "canDelete": true,
          "canUpdate": true,
          "canRead": true
        }
      },
      "canComment": true,
      "canCopy": true,
      "canDelete": true,
      "canBookmark": true,
      "canReport": true,
      "canRun": true,
      "canUpdate": true,
      "canRead": true,
      "canReact": true
    },
    "__typename": "RoutineVersion"
  },
  "stats": {
    "id": true,
    "periodStart": true,
    "periodEnd": true,
    "periodType": true,
    "runsStarted": true,
    "runsCompleted": true,
    "runCompletionTimeAverage": true,
    "runContextSwitchesAverage": true
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
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
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
} as const;

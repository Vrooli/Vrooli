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
      "__typename": "ApiVersion"
    },
    "inputs": {
      "__typename": "RoutineVersionInput"
    },
    "nodes": {
      "__typename": "Node"
    },
    "nodeLinks": {},
    "outputs": {
      "__typename": "RoutineVersionOutput"
    },
    "pullRequest": {
      "__typename": "PullRequest"
    },
    "resourceList": {
      "__typename": "ResourceList"
    },
    "smartContractVersion": {
      "__typename": "SmartContractVersion"
    },
    "suggestedNextByRoutineVersion": {
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
        "organization": {},
        "schedule": {
          "labels": {},
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
        "user": {},
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
} as const;

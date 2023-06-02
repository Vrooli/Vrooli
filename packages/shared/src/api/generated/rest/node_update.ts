export const node_update = {
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
  "routineVersion": {
    "versionNotes": true,
    "apiVersion": {
      "__typename": "ApiVersion"
    },
    "inputs": {
      "__typename": "RoutineVersionInput"
    },
    "outputs": {
      "__typename": "RoutineVersionOutput"
    },
    "pullRequest": {
      "__typename": "PullRequest"
    },
    "resourceList": {
      "__typename": "ResourceList"
    },
    "root": {
      "__typename": "Routine"
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
  "__typename": "Node"
};

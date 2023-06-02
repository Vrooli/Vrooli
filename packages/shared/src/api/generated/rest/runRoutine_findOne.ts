export const runRoutine_findOne = {
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
        },
        "__typename": "StandardVersion"
      },
      "__typename": "RoutineVersionInput"
    },
    "__typename": "RunRoutineInput"
  },
  "routineVersion": {
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
      "versionLabel": true,
      "__typename": "Routine"
    },
    "__typename": "RunRoutineStep"
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
    "__typename": "Organization"
  },
  "schedule": {
    "labels": {
      "__typename": "Label"
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
      "newEndTime": true,
      "__typename": "ScheduleException"
    },
    "recurrences": {
      "id": true,
      "recurrenceType": true,
      "interval": true,
      "dayOfWeek": true,
      "dayOfMonth": true,
      "month": true,
      "endDate": true,
      "__typename": "ScheduleRecurrence"
    },
    "__typename": "Schedule"
  },
  "user": {
    "__typename": "User"
  },
  "you": {
    "canDelete": true,
    "canUpdate": true,
    "canRead": true
  },
  "__typename": "RunRoutine"
} as const;

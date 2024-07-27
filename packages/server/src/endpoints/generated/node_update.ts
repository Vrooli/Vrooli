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
      "routineType": true,
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
      "routineVersion": {
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
        "routineType": true,
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
} as const;

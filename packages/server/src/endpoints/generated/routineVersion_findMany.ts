export const routineVersion_findMany = {
  "edges": {
    "cursor": true,
    "node": {
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
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "RoutineVersion"
} as const;

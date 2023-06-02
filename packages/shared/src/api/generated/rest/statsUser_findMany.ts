export const statsUser_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "apisCreated": true,
      "organizationsCreated": true,
      "projectsCreated": true,
      "projectsCompleted": true,
      "projectCompletionTimeAverage": true,
      "quizzesPassed": true,
      "quizzesFailed": true,
      "routinesCreated": true,
      "routinesCompleted": true,
      "routineCompletionTimeAverage": true,
      "runProjectsStarted": true,
      "runProjectsCompleted": true,
      "runProjectCompletionTimeAverage": true,
      "runProjectContextSwitchesAverage": true,
      "runRoutinesStarted": true,
      "runRoutinesCompleted": true,
      "runRoutineCompletionTimeAverage": true,
      "runRoutineContextSwitchesAverage": true,
      "smartContractsCreated": true,
      "smartContractsCompleted": true,
      "smartContractCompletionTimeAverage": true,
      "standardsCreated": true,
      "standardsCompleted": true,
      "standardCompletionTimeAverage": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "StatsUser"
} as const;

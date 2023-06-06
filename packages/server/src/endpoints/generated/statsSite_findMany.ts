export const statsSite_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "activeUsers": true,
      "apiCalls": true,
      "apisCreated": true,
      "organizationsCreated": true,
      "projectsCreated": true,
      "projectsCompleted": true,
      "projectCompletionTimeAverage": true,
      "quizzesCreated": true,
      "quizzesCompleted": true,
      "routinesCreated": true,
      "routinesCompleted": true,
      "routineCompletionTimeAverage": true,
      "routineSimplicityAverage": true,
      "routineComplexityAverage": true,
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
      "smartContractCalls": true,
      "standardsCreated": true,
      "standardsCompleted": true,
      "standardCompletionTimeAverage": true,
      "verifiedEmailsCreated": true,
      "verifiedWalletsCreated": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "StatsSite"
} as const;

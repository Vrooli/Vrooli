export const statsProject_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "directories": true,
      "apis": true,
      "notes": true,
      "organizations": true,
      "projects": true,
      "routines": true,
      "smartContracts": true,
      "standards": true,
      "runsStarted": true,
      "runsCompleted": true,
      "runCompletionTimeAverage": true,
      "runContextSwitchesAverage": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "StatsProject"
};

export const statsCode_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "calls": true,
      "routineVersions": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "StatsCode"
} as const;

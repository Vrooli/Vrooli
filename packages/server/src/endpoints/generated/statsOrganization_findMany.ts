export const statsOrganization_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "periodStart": true,
      "periodEnd": true,
      "periodType": true,
      "apis": true,
      "members": true,
      "notes": true,
      "projects": true,
      "routines": true,
      "smartContracts": true,
      "standards": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "StatsOrganization"
} as const;

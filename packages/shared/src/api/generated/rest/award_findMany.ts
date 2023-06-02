export const award_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "timeCurrentTierCompleted": true,
      "category": true,
      "progress": true,
      "title": true,
      "description": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Award"
} as const;

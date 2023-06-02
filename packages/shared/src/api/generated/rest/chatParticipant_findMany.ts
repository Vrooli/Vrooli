export const chatParticipant_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "user": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ChatParticipant"
} as const;

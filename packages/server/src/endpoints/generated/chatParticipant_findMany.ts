export const chatParticipant_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "user": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true
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

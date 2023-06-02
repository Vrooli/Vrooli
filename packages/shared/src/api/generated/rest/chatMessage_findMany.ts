export const chatMessage_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "translations": {
        "id": true,
        "language": true,
        "text": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "user": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true
      },
      "score": true,
      "reportsCount": true,
      "you": {
        "canDelete": true,
        "canReply": true,
        "canReport": true,
        "canUpdate": true,
        "canReact": true,
        "reaction": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ChatMessage"
} as const;

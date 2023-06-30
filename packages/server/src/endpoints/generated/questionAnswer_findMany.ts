export const questionAnswer_findMany = {
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
      "createdBy": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true
      },
      "score": true,
      "bookmarks": true,
      "isAccepted": true,
      "commentsCount": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "QuestionAnswer"
} as const;

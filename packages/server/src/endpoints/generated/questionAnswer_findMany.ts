export const questionAnswer_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "createdBy": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "isBotDepictingPerson": true,
        "name": true,
        "profileImage": true
      },
      "score": true,
      "bookmarks": true,
      "isAccepted": true,
      "commentsCount": true,
      "translations": {
        "id": true,
        "language": true,
        "text": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "QuestionAnswer"
} as const;

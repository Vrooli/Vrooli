export const quizQuestion_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "translations": {
        "id": true,
        "language": true,
        "helpText": true,
        "questionText": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "order": true,
      "points": true,
      "responsesCount": true,
      "standardVersion": {
        "id": true,
        "isLatest": true,
        "isPrivate": true,
        "versionIndex": true,
        "versionLabel": true,
        "root": {
          "id": true,
          "isPrivate": true
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "jsonVariable": true,
          "name": true
        }
      },
      "you": {
        "canDelete": true,
        "canUpdate": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "QuizQuestion"
};

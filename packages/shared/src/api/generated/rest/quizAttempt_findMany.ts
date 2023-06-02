export const quizAttempt_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "pointsEarned": true,
      "status": true,
      "contextSwitches": true,
      "timeTaken": true,
      "responsesCount": true,
      "quiz": {
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
        "attemptsCount": true,
        "quizQuestionsCount": true,
        "project": {
          "id": true,
          "isPrivate": true
        },
        "routine": {
          "id": true,
          "isInternal": true,
          "isPrivate": true
        },
        "you": {
          "canDelete": true,
          "canBookmark": true,
          "canUpdate": true,
          "canRead": true,
          "canReact": true,
          "hasCompleted": true,
          "isBookmarked": true,
          "reaction": true
        }
      },
      "user": {
        "id": true,
        "isBot": true,
        "name": true,
        "handle": true
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
  "__typename": "QuizAttempt"
};

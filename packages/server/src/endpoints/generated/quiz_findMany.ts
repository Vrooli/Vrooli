export const quiz_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "createdBy": {
        "id": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true
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
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Quiz"
} as const;

export const quizQuestion_findOne = {
  "responses": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "response": true,
    "quizAttempt": {
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
          "bannerImage": true,
          "handle": true,
          "isBot": true,
          "name": true,
          "profileImage": true,
          "__typename": "User"
        },
        "score": true,
        "bookmarks": true,
        "attemptsCount": true,
        "quizQuestionsCount": true,
        "project": {
          "id": true,
          "isPrivate": true,
          "__typename": "Project"
        },
        "routine": {
          "id": true,
          "isInternal": true,
          "isPrivate": true,
          "__typename": "Routine"
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
        },
        "__typename": "Quiz"
      },
      "user": {
        "id": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true,
        "__typename": "User"
      },
      "you": {
        "canDelete": true,
        "canUpdate": true
      },
      "__typename": "QuizAttempt"
    },
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
    "__typename": "QuizQuestionResponse"
  },
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
      "isPrivate": true,
      "__typename": "Standard"
    },
    "translations": {
      "id": true,
      "language": true,
      "description": true,
      "jsonVariable": true,
      "name": true
    },
    "__typename": "StandardVersion"
  },
  "you": {
    "canDelete": true,
    "canUpdate": true
  },
  "__typename": "QuizQuestion"
} as const;

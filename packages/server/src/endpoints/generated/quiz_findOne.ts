export const quiz_findOne = {
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
    },
    "quizQuestions": {
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
        },
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
                    "created_at": true,
                    "updated_at": true,
                    "bannerImage": true,
                    "handle": true,
                    "isBot": true,
                    "isBotDepictingPerson": true,
                    "name": true,
                    "profileImage": true
                },
                "you": {
                    "canDelete": true,
                    "canUpdate": true
                }
            },
            "you": {
                "canDelete": true,
                "canUpdate": true
            }
        },
        "translations": {
            "id": true,
            "language": true,
            "helpText": true,
            "questionText": true
        }
    },
    "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
    }
};
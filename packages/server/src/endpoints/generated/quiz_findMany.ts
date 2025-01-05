export const quiz_findMany = {
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
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    }
};
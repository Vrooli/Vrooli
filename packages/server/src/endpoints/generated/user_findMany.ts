export const user_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "publicId": true,
            "createdAt": true,
            "updatedAt": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "isBotDepictingPerson": true,
            "name": true,
            "profileImage": true,
            "bookmarks": true,
            "reportsReceivedCount": true,
            "you": {
                "canDelete": true,
                "canReport": true,
                "canUpdate": true,
                "isBookmarked": true,
                "isViewed": true
            },
            "translations": {
                "id": true,
                "language": true,
                "bio": true
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "-2079387191"
};
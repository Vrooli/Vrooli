export const tag_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "createdAt": true,
            "tag": true,
            "bookmarks": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
            },
            "you": {
                "isOwn": true,
                "isBookmarked": true,
            },
        },
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true,
    },
    "__cacheKey": "473956902",
};
